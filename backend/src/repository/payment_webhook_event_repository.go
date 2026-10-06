package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentWebhookEventRepository interface {
	Create(event *models.PaymentWebhookEvent) error
	GetByProviderAndEventID(provider, eventID string) (*models.PaymentWebhookEvent, error)
	MarkProcessed(id uuid.UUID) error
	MarkFailed(id uuid.UUID, reason string) error

	// ExistsByProviderAndEventID is the fast idempotency check.
	ExistsByProviderAndEventID(provider, eventID string) (bool, error)
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
	return r.db.Create(event).Error
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

func (r *paymentWebhookEventRepository) ExistsByProviderAndEventID(provider, eventID string) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.PaymentWebhookEvent{}).
		Where("provider = ? AND provider_event_id = ?", provider, eventID).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}
