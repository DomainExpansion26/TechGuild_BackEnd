package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // 👈 Ye naya import add karna hai
)

// ErrWebhookEventDuplicate — the (provider, provider_event_id) row already
// existed, so the insert was a no-op. Callers treat it as a benign concurrent
// duplicate, not a failure.
var ErrWebhookEventDuplicate = errors.New("duplicate webhook event skipped")

type PaymentWebhookEventRepository interface {
	Create(event *models.PaymentWebhookEvent) error
	GetByProviderAndEventID(provider, eventID string) (*models.PaymentWebhookEvent, error)
	MarkProcessed(id uuid.UUID) error
	MarkFailed(id uuid.UUID, reason string) error
}

type paymentWebhookEventRepository struct {
	db *gorm.DB
}

func NewPaymentWebhookEventRepository() PaymentWebhookEventRepository {
	return &paymentWebhookEventRepository{db: postgres.DB}
}

func NewPaymentWebhookEventRepositoryTx(tx *gorm.DB) PaymentWebhookEventRepository {
	return &paymentWebhookEventRepository{db: tx}
}

func (r *paymentWebhookEventRepository) Create(event *models.PaymentWebhookEvent) error {
	// 👇 Yahan humne OnConflict lagaya hai taaki race condition me 500 error na aaye
	result := r.db.Clauses(clause.OnConflict{
		DoNothing: true, // Duplicate insert hone par DB error nahi dega
	}).Create(event)

	if result.Error != nil {
		return result.Error
	}

	// Agar RowsAffected 0 hai, matlab event pehle se database me maujood tha
	if result.RowsAffected == 0 {
		return ErrWebhookEventDuplicate
	}

	return nil
}

func (r *paymentWebhookEventRepository) GetByProviderAndEventID(provider, eventID string) (*models.PaymentWebhookEvent, error) {
	var event models.PaymentWebhookEvent
	err := r.db.
		Where("provider = ? AND provider_event_id = ?", provider, eventID).
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *paymentWebhookEventRepository) MarkProcessed(id uuid.UUID) error {
	return r.db.
		Model(&models.PaymentWebhookEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.WebhookEventStatusProcessed,
			"processed_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *paymentWebhookEventRepository) MarkFailed(id uuid.UUID, reason string) error {
	return r.db.
		Model(&models.PaymentWebhookEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        models.WebhookEventStatusFailed,
			"processed_at":  gorm.Expr("NOW()"),
			"error_message": reason,
		}).Error
}
