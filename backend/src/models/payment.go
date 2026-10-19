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

// Payment represents a client's funding transaction for a quest milestone.
//
// Amount = TOTAL client charged (milestone + client fee).
// Example: milestone ₹10,000, client fee ₹500 → Amount = ₹10,500.
type Payment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Who is paying (quest client)
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   User      `gorm:"foreignKey:UserID"`

	// What they're paying for (denormalized for fast queries + audit)
	QuestID          uuid.UUID `gorm:"type:uuid;not null;index"`
	QuestMilestoneID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Provider details
	Provider          string  `gorm:"type:varchar(50);not null;index"`
	ProviderPaymentID *string `gorm:"type:varchar(255);index"` // pay_xxx (after capture)
	ProviderOrderID   *string `gorm:"type:varchar(255);index"` // order_xxx (at creation, Razorpay)

	// Amount = gross charged to client (milestone + client fee), minor units
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// Settlement (cross-currency)
	SettlementAmount   int64  `gorm:"not null;default:0"`
	SettlementCurrency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// FX rate (rational string)
	FxRate         string `gorm:"type:varchar(50)"`
	FxRateSource   string `gorm:"type:varchar(30)"`
	FxRateLockedAt *time.Time

	Status PaymentStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_idem"`

	CapturedAt    *time.Time
	FailedAt      *time.Time
	FailureReason string `gorm:"type:text"`

	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	Description string         `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
