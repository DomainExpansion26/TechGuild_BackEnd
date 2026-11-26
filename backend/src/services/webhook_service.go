package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	// Idempotency: a `processed` event is done, but a `failed` one must be
	// retryable — otherwise Razorpay's redelivery is silently swallowed and the
	// money never reconciles.
	var record *models.PaymentWebhookEvent

	existing, err := s.eventRepo.GetByProviderAndEventID(event.Provider, event.ProviderEventID)
	switch {
	case err == nil && existing != nil:
		if existing.Status == models.WebhookEventStatusProcessed {
			log.Printf("webhook duplicate: provider=%s event_id=%s already processed",
				event.Provider, event.ProviderEventID)
			return nil
		}
		log.Printf("webhook retry: provider=%s event_id=%s status=%s — re-dispatching",
			event.Provider, event.ProviderEventID, existing.Status)
		record = existing // reuse the row; fall through to the shared tail below

	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		return fmt.Errorf("idempotency check failed: %w", err)

	default:
		rawBytes, _ := json.Marshal(event.Raw)
		record = &models.PaymentWebhookEvent{
			Provider:        event.Provider,
			ProviderEventID: event.ProviderEventID,
			EventType:       event.EventType,
			Payload:         datatypes.JSON(rawBytes),
			Signature:       signature,
			Status:          models.WebhookEventStatusReceived,
		}
		if err := s.eventRepo.Create(record); err != nil {
			if errors.Is(err, repository.ErrWebhookEventDuplicate) {
				log.Printf("webhook race: provider=%s event_id=%s", event.Provider, event.ProviderEventID)
				return nil
			}
			return fmt.Errorf("failed to persist webhook: %w", err)
		}
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
	case "fund_account.validation.completed", "fund_account.validated":
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
	payoutRepo := repository.NewPayoutRepository()
	p, err := payoutRepo.GetByProviderPayoutID("razorpay", payoutID)
	if err != nil || p == nil {
		log.Printf("razorpay: payout.processed id=%s — no matching payout row", payoutID)
		return nil
	}
	// Only an in-flight payout may become paid: a late `processed` must never
	// resurrect a payout that was already failed or reversed.
	if p.Status != models.PayoutStatusPending && p.Status != models.PayoutStatusProcessing {
		log.Printf("razorpay: payout.processed id=%s ignored (status=%s)", payoutID, p.Status)
		return nil
	}
	if err := payoutRepo.MarkPaid(p.ID); err != nil {
		return fmt.Errorf("mark payout paid: %w", err)
	}
	return nil
}

func (s *WebhookService) handlePayoutFailed(event *WebhookEvent) error {
	payoutID := extractPayoutID(event.Raw)
	reason := extractPayoutFailureReason(event.Raw)
	if payoutID == "" {
		return nil
	}
	log.Printf("razorpay: payout.failed id=%s reason=%s", payoutID, reason)
	return s.reversePayout(payoutID, models.PayoutStatusFailed, reason)
}

func (s *WebhookService) handlePayoutReversed(event *WebhookEvent) error {
	payoutID := extractPayoutID(event.Raw)
	if payoutID == "" {
		return nil
	}
	log.Printf("razorpay: payout.reversed id=%s", payoutID)
	return s.reversePayout(payoutID, models.PayoutStatusReversed, "payout reversed by provider")
}

// reversePayout returns a failed/reversed payout's money to the user's available
// balance. It shares its idempotency key with the synchronous compensation path and
// re-checks status under a row lock, so the two can never both re-credit the debit.
func (s *WebhookService) reversePayout(providerPayoutID string, target models.PayoutStatus, reason string) error {
	payoutRepo := repository.NewPayoutRepository()
	p, err := payoutRepo.GetByProviderPayoutID("razorpay", providerPayoutID)
	if err != nil || p == nil {
		log.Printf("razorpay: payout %s id=%s — no matching payout row", target, providerPayoutID)
		return nil
	}

	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.Payout
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", p.ID).
			First(&locked).Error; err != nil {
			return fmt.Errorf("lock payout: %w", err)
		}
		if locked.Status == models.PayoutStatusFailed || locked.Status == models.PayoutStatusReversed {
			return nil // already reversed (sync path or an earlier webhook)
		}

		if err := tx.Model(&models.Payout{}).
			Where("id = ?", locked.ID).
			Updates(map[string]interface{}{
				"status":         target,
				"failure_reason": reason,
			}).Error; err != nil {
			return fmt.Errorf("mark payout %s: %w", target, err)
		}

		var w models.Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", locked.UserID).
			First(&w).Error; err != nil {
			return fmt.Errorf("lock wallet: %w", err)
		}

		newAvailable := w.AvailableBalanceMinor + locked.Amount
		totalWithdrawn := w.TotalWithdrawnMinor - locked.Amount
		if totalWithdrawn < 0 {
			totalWithdrawn = 0 // chk_wallet_balances_nonneg
		}
		if err := tx.Model(&models.Wallet{}).
			Where("id = ?", w.ID).
			Updates(map[string]interface{}{
				"available_balance_minor": newAvailable,
				"total_withdrawn_minor":   totalWithdrawn,
				"updated_at":              time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("re-credit wallet: %w", err)
		}

		return tx.Create(&models.WalletTransaction{
			WalletID:          w.ID,
			UserID:            locked.UserID,
			Type:              models.WalletTxTypeWithdrawalFailed,
			AmountMinor:       locked.Amount,
			BalanceAfterMinor: newAvailable,
			Currency:          w.Currency,
			Reference:         providerPayoutID,
			Description:       "Withdrawal reversed: funds returned",
			IdempotencyKey:    fmt.Sprintf("payout:%s:reversal", locked.ID),
		}).Error
	})
}

// ============================================================
// Refund handlers
// ============================================================

func (s *WebhookService) handleRefundProcessed(event *WebhookEvent) error {
	refundID := extractRefundID(event.Raw)
	if refundID == "" {
		return nil
	}
	refundRepo := repository.NewRefundRepository()
	ref, err := refundRepo.GetByProviderRefundID("razorpay", refundID)
	if err != nil || ref == nil {
		log.Printf("razorpay: refund.processed id=%s — no matching refund row", refundID)
		return nil
	}
	if ref.Status == models.RefundStatusCompleted {
		return nil
	}
	if err := refundRepo.MarkCompleted(ref.ID); err != nil {
		return fmt.Errorf("mark refund completed: %w", err)
	}
	log.Printf("razorpay: refund %s marked completed", ref.ID)
	return nil
}

func (s *WebhookService) handleRefundFailed(event *WebhookEvent) error {
	refundID := extractRefundID(event.Raw)
	if refundID == "" {
		return nil
	}
	refundRepo := repository.NewRefundRepository()
	ref, err := refundRepo.GetByProviderRefundID("razorpay", refundID)
	if err != nil || ref == nil {
		log.Printf("razorpay: refund.failed id=%s — no matching refund row", refundID)
		return nil
	}
	// A locally-completed refund already released the escrow. Flipping it to
	// failed would leave the milestone marked refunded with no money returned,
	// so leave the state alone and shout instead.
	if ref.Status == models.RefundStatusCompleted {
		log.Printf("CRITICAL: razorpay reports refund %s FAILED but it is already completed locally — manual reconciliation required", refundID)
		return nil
	}
	reason := extractRefundFailureReason(event.Raw)
	if reason == "" {
		reason = "razorpay refund failed"
	}
	if err := refundRepo.MarkFailed(ref.ID, reason); err != nil {
		return fmt.Errorf("mark refund failed: %w", err)
	}
	return nil
}

func (s *WebhookService) handleMockEvent(event *WebhookEvent) error {
	log.Printf("mock webhook: type=%s event_id=%s", event.EventType, event.ProviderEventID)
	return nil
}

// ============================================================
// Helpers
// ============================================================

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

func extractRefundFailureReason(raw map[string]any) string {
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
	reason, _ := entity["failure_reason"].(string)
	return reason
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
