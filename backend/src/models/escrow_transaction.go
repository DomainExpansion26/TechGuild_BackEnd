package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EscrowStatus string

const (
	EscrowStatusPending        EscrowStatus = "pending"
	EscrowStatusHeld           EscrowStatus = "held"
	EscrowStatusReleasePending EscrowStatus = "release_pending"
	EscrowStatusReleased       EscrowStatus = "released"
	EscrowStatusRefundPending  EscrowStatus = "refund_pending"
	EscrowStatusRefunded       EscrowStatus = "refunded"
	EscrowStatusFailed         EscrowStatus = "failed"
)

// EscrowTransaction tracks funds held in escrow for a quest milestone.
//
// Amount = milestone amount ONLY (client fee is NOT held in escrow;
// client fee is credited to platform immediately at payment time).
type EscrowTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID        uuid.UUID `gorm:"type:uuid;not null;index"`
	QuestID          uuid.UUID `gorm:"type:uuid;not null;index"`
	QuestMilestoneID uuid.UUID `gorm:"type:uuid;not null;index"`

	Provider         string  `gorm:"type:varchar(50);not null"`
	ProviderEscrowID *string `gorm:"type:varchar(255);index"`

	// Amount = milestone amount only (not the client charged amount)
	Amount             int64  `gorm:"not null"`
	Currency           string `gorm:"type:varchar(10);not null;default:'INR'"`
	SettlementAmount   int64  `gorm:"not null;default:0"`
	SettlementCurrency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Status EscrowStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	ReleaseIdempotencyKey *string `gorm:"type:varchar(255);uniqueIndex:uniq_escrow_release_idem"`

	HeldAt        *time.Time
	ReleasedAt    *time.Time
	RefundedAt    *time.Time
	FailureReason string `gorm:"type:text"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e *EscrowTransaction) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
