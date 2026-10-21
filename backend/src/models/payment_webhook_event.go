package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WebhookEventStatus string

const (
	WebhookEventStatusReceived   WebhookEventStatus = "received"
	WebhookEventStatusProcessing WebhookEventStatus = "processing"
	WebhookEventStatusProcessed  WebhookEventStatus = "processed"
	WebhookEventStatusFailed     WebhookEventStatus = "failed"
)

// PaymentWebhookEvent stores every webhook delivered by a payment provider.
// Idempotency: unique index on (provider, provider_event_id) ensures duplicate
// deliveries have exactly one financial effect.
type PaymentWebhookEvent struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	Provider        string `gorm:"type:varchar(50);not null;uniqueIndex:uniq_provider_event"`
	ProviderEventID string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_provider_event"`

	EventType string `gorm:"type:varchar(100);not null;index"`

	Payload datatypes.JSON `gorm:"type:jsonb;not null"`

	Signature string `gorm:"type:text"`

	Status WebhookEventStatus `gorm:"type:varchar(30);not null;default:'received';index"`

	ProcessedAt  *time.Time
	ErrorMessage *string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e *PaymentWebhookEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
