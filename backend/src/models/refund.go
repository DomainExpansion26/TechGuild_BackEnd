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

// Refund records a full or partial refund to the client.
//
// Rules:
//   - Refund amount must NOT exceed refundable amount.
//   - Refund creation must be idempotent.
type Refund struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	MilestonePaymentID *uuid.UUID `gorm:"type:uuid;index"`
	ProjectID          uuid.UUID  `gorm:"type:uuid;not null;index"`

	Provider         string  `gorm:"type:varchar(50);not null"`
	ProviderRefundID *string `gorm:"type:varchar(255);index"`

	// Amount, minor units
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
