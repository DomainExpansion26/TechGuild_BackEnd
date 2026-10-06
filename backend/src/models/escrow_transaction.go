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

// EscrowTransaction tracks the lifecycle of funds held in escrow for a milestone.
//
// State machine:
//
//	pending → held → release_pending → released
//	             ↓              ↓
//	        refund_pending → refunded
//	             ↓
//	          failed
type EscrowTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ContractID  uuid.UUID `gorm:"type:uuid;not null;index"`
	MilestoneID uuid.UUID `gorm:"type:uuid;not null;index"`

	Provider         string  `gorm:"type:varchar(50);not null"`
	ProviderEscrowID *string `gorm:"type:varchar(255);index"`

	// Amounts in minor units
	Amount             int64  `gorm:"not null"`
	Currency           string `gorm:"type:varchar(10);not null;default:'INR'"`
	SettlementAmount   int64  `gorm:"not null;default:0"`
	SettlementCurrency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Status EscrowStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	// Idempotency for the release operation
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
