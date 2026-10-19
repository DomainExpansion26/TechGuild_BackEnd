package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TxTypePayment               TransactionType = "payment"
	TxTypeCapture               TransactionType = "capture"
	TxTypeEscrowHold            TransactionType = "escrow_hold"
	TxTypeEscrowRelease         TransactionType = "escrow_release"
	TxTypeEscrowRefund          TransactionType = "escrow_refund"
	TxTypePlatformFeeClient     TransactionType = "platform_fee_client"
	TxTypePlatformFeeFreelancer TransactionType = "platform_fee_freelancer"
	TxTypePayout                TransactionType = "payout"
	TxTypeWalletCredit          TransactionType = "wallet_credit"
	TxTypeWithdrawal            TransactionType = "withdrawal"
	TxTypeRefund                TransactionType = "refund"
	TxTypeChargeback            TransactionType = "chargeback"
	TxTypeAdjustment            TransactionType = "adjustment"
	TxTypeFxConversion          TransactionType = "fx_conversion"
)

// PaymentTransaction — immutable ledger.
//
// PaymentID and QuestID are NULLABLE because wallet withdrawals don't
// reference a specific payment or quest (they debit the user's wallet).
type PaymentTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Nullable: wallet withdrawals have no payment/quest
	PaymentID        *uuid.UUID `gorm:"type:uuid;index"`
	QuestID          *uuid.UUID `gorm:"type:uuid;index"`
	QuestMilestoneID *uuid.UUID `gorm:"type:uuid;index"`

	TransactionType TransactionType `gorm:"type:varchar(40);not null;index"`

	// Signed amount, minor units
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	Provider              string  `gorm:"type:varchar(50)"`
	ProviderTransactionID *string `gorm:"type:varchar(255);index"`

	Reference string `gorm:"type:varchar(255);index"`

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
