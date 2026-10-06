package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TxTypePayment       TransactionType = "payment"
	TxTypeCapture       TransactionType = "capture"
	TxTypeEscrowHold    TransactionType = "escrow_hold"
	TxTypeEscrowRelease TransactionType = "escrow_release"
	TxTypePlatformFee   TransactionType = "platform_fee"
	TxTypePayout        TransactionType = "payout"
	TxTypeRefund        TransactionType = "refund"
	TxTypeChargeback    TransactionType = "chargeback"
	TxTypeAdjustment    TransactionType = "adjustment"
	TxTypeFxConversion  TransactionType = "fx_conversion"
)

// PaymentTransaction is an IMMUTABLE ledger entry. Once written, never
// modify. Corrections are new compensating rows, never edits.
//
// Amount is SIGNED: negative for credits out, positive for debits in.
type PaymentTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PaymentID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	ProjectID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	ContractID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	MilestoneID *uuid.UUID `gorm:"type:uuid;index"`

	TransactionType TransactionType `gorm:"type:varchar(30);not null;index"`

	// Signed amount, minor units of Currency
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Provider              string  `gorm:"type:varchar(50)"`
	ProviderTransactionID *string `gorm:"type:varchar(255);index"`

	// Cross-reference to source entity (payout_id, refund_id, etc.)
	Reference string `gorm:"type:varchar(255);index"`

	// Idempotency for ledger dedup
	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_ledger_idem"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"index"`
}

func (t *PaymentTransaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
