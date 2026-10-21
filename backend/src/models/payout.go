package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusPaid       PayoutStatus = "paid"
	PayoutStatusFailed     PayoutStatus = "failed"
	PayoutStatusCancelled  PayoutStatus = "cancelled"
	PayoutStatusReversed   PayoutStatus = "reversed"
)

type PayoutSource string

const (
	PayoutSourceMilestone        PayoutSource = "milestone"         // auto-created at milestone release
	PayoutSourceWalletWithdrawal PayoutSource = "wallet_withdrawal" // manual from wallet
)

// Payout records money going OUT to a freelancer.
//
// Two sources:
//   - Source=milestone: auto-payout at milestone release (legacy flow)
//   - Source=wallet_withdrawal: manual withdrawal from wallet
//
// For wallet_withdrawal, MilestonePaymentID and QuestID are NULL.
type Payout struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;not null;index"` // freelancer receiving

	// Which flow created this payout
	Source PayoutSource `gorm:"type:varchar(30);not null;default:'milestone';index"`

	// Nullable — wallet withdrawals don't reference a specific milestone/quest
	MilestonePaymentID *uuid.UUID `gorm:"type:uuid;index"`
	QuestID            *uuid.UUID `gorm:"type:uuid;index"`

	PayoutMethodID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Provider details
	Provider         string  `gorm:"type:varchar(50);not null;index"`
	ProviderPayoutID *string `gorm:"type:varchar(255);index"`

	// Amount, minor units
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	SettlementAmount   int64  `gorm:"not null;default:0"`
	SettlementCurrency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Status PayoutStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payout_idem"`

	FailureReason string `gorm:"type:text"`
	ProcessedAt   *time.Time

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Payout) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
