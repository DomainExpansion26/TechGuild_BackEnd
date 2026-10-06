package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusAuthorized        PaymentStatus = "authorized"
	PaymentStatusCaptured          PaymentStatus = "captured"
	PaymentStatusFailed            PaymentStatus = "failed"
	PaymentStatusCancelled         PaymentStatus = "cancelled"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
)

// Payment represents a client's funding transaction for a milestone.
// Money is stored as int64 minor units of Currency (INR paise, USD cents).
type Payment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Who is paying
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   User      `gorm:"foreignKey:UserID"`

	// What they're paying for (denormalized for fast queries + audit)
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ContractID  uuid.UUID `gorm:"type:uuid;not null;index"`
	MilestoneID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Provider details
	Provider          string  `gorm:"type:varchar(50);not null;index"`
	ProviderPaymentID *string `gorm:"type:varchar(255);index"`

	// Amount (in minor units of Currency)
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// Settlement (for cross-currency, e.g. USD → INR)
	// When same currency, these mirror Amount/Currency.
	SettlementAmount   int64  `gorm:"not null;default:0"`
	SettlementCurrency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// FX rate if applicable, stored as rational string "num/den"
	// Empty string when not applicable.
	FxRate         string `gorm:"type:varchar(50)"`
	FxRateSource   string `gorm:"type:varchar(30)"` // razorpay / rbi_ref / manual
	FxRateLockedAt *time.Time

	// Status
	Status PaymentStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	// Idempotency (unique per provider)
	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_idem"`

	// Timing
	CapturedAt    *time.Time
	FailedAt      *time.Time
	FailureReason string `gorm:"type:text"`

	// Extra provider metadata (safe subset only)
	Metadata datatypes.JSON `gorm:"type:jsonb"`

	Description string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
