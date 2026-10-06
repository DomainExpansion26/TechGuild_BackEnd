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

// Payout records money going OUT to a freelancer/agency.
//
// Rule: a payout must NOT be created before the milestone is released.
type Payout struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;not null;index"` // freelancer/agency receiving

	MilestonePaymentID uuid.UUID `gorm:"type:uuid;not null;index"`
	ProjectID          uuid.UUID `gorm:"type:uuid;not null;index"`
	ContractID         uuid.UUID `gorm:"type:uuid;not null;index"`

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
