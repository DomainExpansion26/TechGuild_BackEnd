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

type PlatformFeeSide string

const (
	PlatformFeeSideClient     PlatformFeeSide = "client"
	PlatformFeeSideFreelancer PlatformFeeSide = "freelancer"
)

// PlatformFee records one side of the dual-fee split.
// Two rows per milestone: one for client fee, one for freelancer fee.
type PlatformFee struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID          uuid.UUID `gorm:"type:uuid;not null;index"`
	MilestonePaymentID uuid.UUID `gorm:"type:uuid;not null;index"`
	QuestID            uuid.UUID `gorm:"type:uuid;not null;index"`

	// Which side of the fee
	Side PlatformFeeSide `gorm:"type:varchar(20);not null;index"`

	UserID *uuid.UUID `gorm:"type:uuid;index"`

	FeeType     FeeType `gorm:"type:varchar(20);not null;default:'percentage'"`
	RateBps     int64   `gorm:"not null;default:0"`
	FixedAmount int64   `gorm:"not null;default:0"`

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
