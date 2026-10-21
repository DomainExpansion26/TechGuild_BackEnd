package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusCompleted  RefundStatus = "completed"
	RefundStatusFailed     RefundStatus = "failed"
	RefundStatusCancelled  RefundStatus = "cancelled"
)

// Refund records a refund to the client.
//
// For now (per discussion): only milestone amount is refunded.
// Client fee is non-refundable.
type Refund struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	MilestonePaymentID *uuid.UUID `gorm:"type:uuid;index"`
	QuestID            uuid.UUID  `gorm:"type:uuid;not null;index"`

	Provider         string  `gorm:"type:varchar(50);not null"`
	ProviderRefundID *string `gorm:"type:varchar(255);index"`

	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Status RefundStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	Reason        string `gorm:"type:text"`
	FailureReason string `gorm:"type:text"`

	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_refund_idem"`

	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *Refund) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
