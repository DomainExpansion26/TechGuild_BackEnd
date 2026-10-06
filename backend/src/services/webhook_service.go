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
}

func NewWebhookService(
	eventRepo repository.PaymentWebhookEventRepository,
	payoutRepo repository.PayoutMethodRepository,
) *WebhookService {
	return &WebhookService{
		verifiers:  make(map[string]WebhookVerifier),
		eventRepo:  eventRepo,
		payoutRepo: payoutRepo,
	}
}

// RegisterVerifier adds a provider verifier. Call this during startup for
// each provider you have credentials for.
func (s *WebhookService) RegisterVerifier(v WebhookVerifier) {
	s.verifiers[v.Provider()] = v
}

// ProcessWebhook handles a webhook delivery. It is idempotent:
// duplicate deliveries with the same (provider, event_id) have exactly one effect.
func (s *WebhookService) ProcessWebhook(provider string, payload []byte, signature string) error {
	// 1. Verify signature BEFORE any DB write
	verifier, ok := s.verifiers[provider]
	if !ok {
		return ErrWebhookUnknownProvider
	}
	if err := verifier.Verify(payload, signature); err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookInvalidSig, err)
	}

	// 2. Parse into provider-neutral event
	event, err := verifier.ParseEvent(payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebhookMalformed, err)
	}

	// 3. Idempotency check — has this event been delivered before?
	exists, err := s.eventRepo.ExistsByProviderAndEventID(event.Provider, event.ProviderEventID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		log.Printf("webhook duplicate: provider=%s event_id=%s — ignoring",
			event.Provider, event.ProviderEventID)
		return nil
	}

	// 4. Persist raw event before processing
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
		// Unique constraint race — another delivery won. Treat as duplicate.
		if isUniqueConstraint(err) {
			log.Printf("webhook race: provider=%s event_id=%s — duplicate insert",
				event.Provider, event.ProviderEventID)
			return nil
		}
		return fmt.Errorf("failed to persist webhook: %w", err)
	}

	// 5. Dispatch to handler
	if err := s.dispatch(event); err != nil {
		_ = s.eventRepo.MarkFailed(record.ID, err.Error())
		return err
	}

	// 6. Mark processed
	return s.eventRepo.MarkProcessed(record.ID)
}

// dispatch routes the event to the correct handler based on type.
// Add more cases as you wire more providers/events.
func (s *WebhookService) dispatch(event *WebhookEvent) error {
	switch event.Provider {
	case "razorpay":
		return s.handleRazorpayEvent(event)
	case "mock":
		return s.handleMockEvent(event)
	default:
		log.Printf("webhook: no dispatcher for provider=%s event=%s",
			event.Provider, event.EventType)
		return nil // don't fail — just ignore
	}
}

// handleRazorpayEvent routes Razorpay events to specific handlers.
func (s *WebhookService) handleRazorpayEvent(event *WebhookEvent) error {
	switch event.EventType {
	case "fund_account.validation.completed":
		return s.handleFundAccountVerified(event)
	case "fund_account.validation.failed":
		return s.handleFundAccountFailed(event)
	default:
		log.Printf("razorpay webhook: unhandled event type=%s", event.EventType)
		return nil
	}
}

func (s *WebhookService) handleFundAccountVerified(event *WebhookEvent) error {
	log.Printf("razorpay: fund_account.validation.completed received event_id=%s", event.ProviderEventID)

	fundAccountID := extractFundAccountID(event.Raw)
	if fundAccountID == "" {
		log.Printf("razorpay: webhook missing fund_account id, event_id=%s", event.ProviderEventID)
		return nil // not fatal, just log
	}

	method, err := s.payoutRepo.GetByProviderMethodID("razorpay", fundAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("razorpay: no payout method found for fund_account_id=%s (orphan webhook)", fundAccountID)
			return nil // orphan account — probably from another source
		}
		return fmt.Errorf("failed to lookup payout method: %w", err)
	}

	// Already verified? Skip.
	if method.Status == models.PayoutMethodStatusVerified {
		log.Printf("razorpay: payout method %s already verified, skipping", method.ID)
		return nil
	}

	if err := s.payoutRepo.UpdateStatus(method.ID, models.PayoutMethodStatusVerified); err != nil {
		return fmt.Errorf("failed to update payout method status: %w", err)
	}

	log.Printf("razorpay: payout method %s marked verified (fund_account=%s)", method.ID, fundAccountID)
	return nil
}

func (s *WebhookService) handleFundAccountFailed(event *WebhookEvent) error {
	log.Printf("razorpay: fund_account.validation.failed received event_id=%s", event.ProviderEventID)

	fundAccountID := extractFundAccountID(event.Raw)
	if fundAccountID == "" {
		return nil
	}

	method, err := s.payoutRepo.GetByProviderMethodID("razorpay", fundAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("razorpay: no payout method found for fund_account_id=%s", fundAccountID)
			return nil
		}
		return err
	}

	if err := s.payoutRepo.UpdateStatus(method.ID, models.PayoutMethodStatusFailed); err != nil {
		return err
	}

	log.Printf("razorpay: payout method %s marked failed (fund_account=%s)", method.ID, fundAccountID)
	return nil
}

func (s *WebhookService) handleMockEvent(event *WebhookEvent) error {
	log.Printf("mock webhook: type=%s event_id=%s", event.EventType, event.ProviderEventID)
	return nil
}

// isUniqueConstraint detects Postgres unique-violation (SQLSTATE 23505).
func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return false
}

// extractFundAccountID pulls fund_account.id from Razorpay's webhook payload.
// Expected shape:
//
//	{ "payload": { "fund_account": { "entity": { "id": "fa_xxx" } } } }
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
