package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WalletTransactionType string

const (
	WalletTxTypeCredit           WalletTransactionType = "credit" // milestone released → available
	WalletTxTypeDebit            WalletTransactionType = "debit"  // withdrawal from wallet
	WalletTxTypeHold             WalletTransactionType = "hold"   // pending (future)
	WalletTxTypeHoldRelease      WalletTransactionType = "hold_release"
	WalletTxTypeWithdrawalFailed WalletTransactionType = "withdrawal_failed" // reversed
)

// WalletTransaction is an immutable audit log for every wallet change.
// AmountMinor is SIGNED: positive for credits, negative for debits.
type WalletTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	WalletID uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index"`

	Type WalletTransactionType `gorm:"type:varchar(30);not null;index"`

	// Signed amount in minor units
	AmountMinor int64 `gorm:"not null"`

	// Balance snapshot AFTER this transaction
	BalanceAfterMinor int64 `gorm:"not null"`

	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// Cross-reference (milestone_payment_id, payout_id, etc.)
	Reference string `gorm:"type:varchar(255);index"`

	Description string `gorm:"type:text"`

	// Idempotency for dedup
	IdempotencyKey string `gorm:"type:varchar(255);not null;uniqueIndex"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"index"`
}

func (wt *WalletTransaction) BeforeCreate(tx *gorm.DB) error {
	if wt.ID == uuid.Nil {
		wt.ID = uuid.New()
	}
	return nil
}
