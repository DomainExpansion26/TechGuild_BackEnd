package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrWebhookUnknownProvider = errors.New("webhook provider not registered")
	ErrWebhookInvalidSig      = errors.New("webhook signature verification failed")
	ErrWebhookMalformed       = errors.New("webhook payload malformed")
)

type WebhookService struct {
	verifiers  map[string]WebhookVerifier
	eventRepo  repository.PaymentWebhookEventRepository
	payoutRepo repository.PayoutMethodRepository

	// For async payment events
	fundingSvc *MilestoneFundingService
}

func NewWebhookService(
	eventRepo repository.PaymentWebhookEventRepository,
	payoutRepo repository.PayoutMethodRepository,
	fundingSvc *MilestoneFundingService,
) *WebhookService {
	return &WebhookService{
		verifiers:  make(map[string]WebhookVerifier),
		eventRepo:  eventRepo,
		payoutRepo: payoutRepo,
		fundingSvc: fundingSvc,
	}
}

func (s *WebhookService) RegisterVerifier(v WebhookVerifier) {
	s.verifiers[v.Provider()] = v
}

// ProcessWebhook — idempotent
func (s *WebhookService) ProcessWebhook(provider string, payload []byte, signature string) error {
	verifier, ok := s.verifiers[provider]
	if !ok {
		return ErrWebhookUnknownProvider
	}
	if err := verifier.Verify(payload, signature); err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookInvalidSig, err)
	}

	event, err := verifier.ParseEvent(payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookMalformed, err)
	}

	exists, err := s.eventRepo.ExistsByProviderAndEventID(event.Provider, event.ProviderEventID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		log.Printf("webhook duplicate: provider=%s event_id=%s — ignoring",
			event.Provider, event.ProviderEventID)
		return nil
	}

	rawBytes, _ := json.Marshal(event.Raw)
	record := &models.PaymentWebhookEvent{
		Provider:        event.Provider,
		ProviderEventID: event.ProviderEventID,
		EventType:       event.EventType,
		Payload:         datatypes.JSON(rawBytes),
		Signature:       signature,
		Status:          models.WebhookEventStatusReceived,
	}
	if err := s.eventRepo.Create(record); err != nil {
		if isUniqueConstraint(err) {
			log.Printf("webhook race: provider=%s event_id=%s", event.Provider, event.ProviderEventID)
			return nil
		}
		return fmt.Errorf("failed to persist webhook: %w", err)
	}

	if err := s.dispatch(event); err != nil {
		_ = s.eventRepo.MarkFailed(record.ID, err.Error())
		return err
	}

	return s.eventRepo.MarkProcessed(record.ID)
}

func (s *WebhookService) dispatch(event *WebhookEvent) error {
	switch event.Provider {
	case "razorpay":
		return s.handleRazorpayEvent(event)
	case "mock":
		return s.handleMockEvent(event)
	default:
		log.Printf("webhook: no dispatcher for provider=%s event=%s",
			event.Provider, event.EventType)
		return nil
	}
}

// ============================================================
// Razorpay event routing
// ============================================================

func (s *WebhookService) handleRazorpayEvent(event *WebhookEvent) error {
	switch event.EventType {
	// Payment lifecycle
	case "payment.captured":
		return s.handlePaymentCaptured(event)
	case "payment.failed":
		return s.handlePaymentFailed(event)

	// Fund account validation (RazorpayX)
	case "fund_account.validation.completed":
		return s.handleFundAccountVerified(event)
	case "fund_account.validation.failed":
		return s.handleFundAccountFailed(event)

	// Payout lifecycle (RazorpayX)
	case "payout.processed":
		return s.handlePayoutProcessed(event)
	case "payout.failed":
		return s.handlePayoutFailed(event)
	case "payout.reversed":
		return s.handlePayoutReversed(event)

	// Refund lifecycle
	case "refund.processed":
		return s.handleRefundProcessed(event)
	case "refund.failed":
		return s.handleRefundFailed(event)

	default:
		log.Printf("razorpay webhook: unhandled event type=%s", event.EventType)
		return nil
	}
}

// ============================================================
// Payment handlers
// ============================================================

func (s *WebhookService) handlePaymentCaptured(event *WebhookEvent) error {
	orderID, paymentID := extractPaymentIDs(event.Raw)
	if orderID == "" || paymentID == "" {
		log.Printf("razorpay: payment.captured missing order/payment id")
		return nil
	}

	if s.fundingSvc == nil {
		log.Printf("razorpay: funding service not wired — skipping payment.captured")
		return nil
	}

	log.Printf("razorpay: payment.captured order=%s payment=%s", orderID, paymentID)

	_, err := s.fundingSvc.MarkPaymentCaptured(orderID, paymentID)
	if err != nil {
		return fmt.Errorf("mark captured: %w", err)
	}
	return nil
}

func (s *WebhookService) handlePaymentFailed(event *WebhookEvent) error {
	orderID, _ := extractPaymentIDs(event.Raw)
	if orderID == "" {
		return nil
	}
	reason := extractPaymentFailureReason(event.Raw)
	if reason == "" {
		reason = "razorpay payment failed"
	}

	if s.fundingSvc == nil {
		return nil
	}

	log.Printf("razorpay: payment.failed order=%s reason=%s", orderID, reason)
	return s.fundingSvc.MarkPaymentFailed(orderID, reason)
}

// ============================================================
// Fund account handlers (RazorpayX)
// ============================================================

func (s *WebhookService) handleFundAccountVerified(event *WebhookEvent) error {
	fundAccountID := extractFundAccountID(event.Raw)
	if fundAccountID == "" {
		return nil
	}

	method, err := s.payoutRepo.GetByProviderMethodID("razorpay", fundAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("razorpay: no payout method for fund_account_id=%s", fundAccountID)
			return nil
		}
		return err
	}

	if method.Status == models.PayoutMethodStatusVerified {
		return nil
	}

	if err := s.payoutRepo.UpdateStatus(method.ID, models.PayoutMethodStatusVerified); err != nil {
		return err
	}
	log.Printf("razorpay: payout method %s verified", method.ID)
	return nil
}

func (s *WebhookService) handleFundAccountFailed(event *WebhookEvent) error {
	fundAccountID := extractFundAccountID(event.Raw)
	if fundAccountID == "" {
		return nil
	}

	method, err := s.payoutRepo.GetByProviderMethodID("razorpay", fundAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if err := s.payoutRepo.UpdateStatus(method.ID, models.PayoutMethodStatusFailed); err != nil {
		return err
	}
	log.Printf("razorpay: payout method %s failed", method.ID)
	return nil
}

// ============================================================
// Payout handlers (RazorpayX)
// ============================================================

func (s *WebhookService) handlePayoutProcessed(event *WebhookEvent) error {
	payoutID := extractPayoutID(event.Raw)
	if payoutID == "" {
		return nil
	}
	log.Printf("razorpay: payout.processed id=%s (handler TODO)", payoutID)
	// TODO: find Payout row by provider_payout_id, mark paid
	return nil
}

func (s *WebhookService) handlePayoutFailed(event *WebhookEvent) error {
	payoutID := extractPayoutID(event.Raw)
	reason := extractPayoutFailureReason(event.Raw)
	if payoutID == "" {
		return nil
	}
	log.Printf("razorpay: payout.failed id=%s reason=%s (handler TODO)", payoutID, reason)
	// TODO: reverse wallet debit, mark payout failed
	return nil
}

func (s *WebhookService) handlePayoutReversed(event *WebhookEvent) error {
	payoutID := extractPayoutID(event.Raw)
	if payoutID == "" {
		return nil
	}
	log.Printf("razorpay: payout.reversed id=%s (handler TODO)", payoutID)
	return nil
}

// ============================================================
// Refund handlers
// ============================================================

func (s *WebhookService) handleRefundProcessed(event *WebhookEvent) error {
	refundID := extractRefundID(event.Raw)
	if refundID == "" {
		return nil
	}
	log.Printf("razorpay: refund.processed id=%s (handler TODO)", refundID)
	// Our refund flow already marks completed synchronously; this is a safety net.
	return nil
}

func (s *WebhookService) handleRefundFailed(event *WebhookEvent) error {
	refundID := extractRefundID(event.Raw)
	if refundID == "" {
		return nil
	}
	log.Printf("razorpay: refund.failed id=%s (handler TODO)", refundID)
	return nil
}

func (s *WebhookService) handleMockEvent(event *WebhookEvent) error {
	log.Printf("mock webhook: type=%s event_id=%s", event.EventType, event.ProviderEventID)
	return nil
}

// ============================================================
// Helpers
// ============================================================

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

func extractFundAccountID(raw map[string]any) string {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return ""
	}
	fa, ok := payload["fund_account"].(map[string]any)
	if !ok {
		return ""
	}
	entity, ok := fa["entity"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := entity["id"].(string)
	return id
}

// extractPaymentIDs returns (order_id, payment_id) from payment event payload.
//
// Expected shape:
//
//	{ "payload": { "payment": { "entity": { "id": "pay_xxx", "order_id": "order_xxx" } } } }
func extractPaymentIDs(raw map[string]any) (string, string) {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return "", ""
	}
	payment, ok := payload["payment"].(map[string]any)
	if !ok {
		return "", ""
	}
	entity, ok := payment["entity"].(map[string]any)
	if !ok {
		return "", ""
	}
	orderID, _ := entity["order_id"].(string)
	paymentID, _ := entity["id"].(string)
	return orderID, paymentID
}

func extractPaymentFailureReason(raw map[string]any) string {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return ""
	}
	payment, ok := payload["payment"].(map[string]any)
	if !ok {
		return ""
	}
	entity, ok := payment["entity"].(map[string]any)
	if !ok {
		return ""
	}
	if errObj, ok := entity["error"].(map[string]any); ok {
		if desc, ok := errObj["description"].(string); ok {
			return desc
		}
	}
	return ""
}

func extractPayoutID(raw map[string]any) string {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return ""
	}
	payout, ok := payload["payout"].(map[string]any)
	if !ok {
		return ""
	}
	entity, ok := payout["entity"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := entity["id"].(string)
	return id
}

func extractPayoutFailureReason(raw map[string]any) string {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return ""
	}
	payout, ok := payload["payout"].(map[string]any)
	if !ok {
		return ""
	}
	entity, ok := payout["entity"].(map[string]any)
	if !ok {
		return ""
	}
	if reason, ok := entity["failure_reason"].(string); ok {
		return reason
	}
	return ""
}

func extractRefundID(raw map[string]any) string {
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return ""
	}
	refund, ok := payload["refund"].(map[string]any)
	if !ok {
		return ""
	}
	entity, ok := refund["entity"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := entity["id"].(string)
	return id
}
