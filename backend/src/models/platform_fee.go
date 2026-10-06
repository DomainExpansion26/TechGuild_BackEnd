package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlatformFeeStatus string

const (
	PlatformFeePending  PlatformFeeStatus = "pending"
	PlatformFeeAccrued  PlatformFeeStatus = "accrued"
	PlatformFeeSettled  PlatformFeeStatus = "settled"
	PlatformFeeReversed PlatformFeeStatus = "reversed"
)

// PlatformFee records the platform's commission for one milestone payment.
//
// RateBps is the rate APPLIED (frozen), not the current config. Recompute
// should never happen — read from MilestonePayment.FeeRateBps instead.
type PlatformFee struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID          uuid.UUID `gorm:"type:uuid;not null;index"`
	MilestonePaymentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	ProjectID          uuid.UUID `gorm:"type:uuid;not null;index"`
	ContractID         uuid.UUID `gorm:"type:uuid;not null;index"`

	// Who earns this fee — the platform account
	UserID *uuid.UUID `gorm:"type:uuid;index"` // nullable, reserved

	FeeType     FeeType `gorm:"type:varchar(20);not null;default:'percentage'"`
	RateBps     int64   `gorm:"not null;default:0"`
	FixedAmount int64   `gorm:"not null;default:0"`

	// Computed fee, minor units
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Status PlatformFeeStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	SettledAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f *PlatformFee) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
