package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PayoutMethodPurpose string

const (
	PayoutMethodPurposePayout PayoutMethodPurpose = "payout"
)

type PayoutMethodType string

const (
	PayoutMethodTypeBank   PayoutMethodType = "bank"
	PayoutMethodTypeUPI    PayoutMethodType = "upi"
	PayoutMethodTypeCard   PayoutMethodType = "card"
	PayoutMethodTypeWallet PayoutMethodType = "wallet"
)

type PayoutMethodStatus string

const (
	PayoutMethodStatusPendingVerification PayoutMethodStatus = "pending_verification"
	PayoutMethodStatusVerified            PayoutMethodStatus = "verified"
	PayoutMethodStatusFailed              PayoutMethodStatus = "failed"
	PayoutMethodStatusDisabled            PayoutMethodStatus = "disabled"
)

type PayoutMethod struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Currently owner is always a user (individual / agency / party all mapped to user).
	// When team support is added, make this nullable and add TeamID + CHECK constraint.
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Purpose PayoutMethodPurpose `gorm:"type:varchar(20);not null;default:'payout';index"`

	Type PayoutMethodType `gorm:"type:varchar(20);not null"`

	Provider         string  `gorm:"type:varchar(50);not null;uniqueIndex:uniq_provider_payout_method_id,where:provider_method_id IS NOT NULL"`
	ProviderMethodID *string `gorm:"type:varchar(255);uniqueIndex:uniq_provider_payout_method_id,where:provider_method_id IS NOT NULL"`

	// Bank display / metadata (masked only, never full account number)
	BankName      *string `gorm:"type:varchar(100)"`
	AccountHolder *string `gorm:"type:varchar(100)"`
	MaskedAccount *string `gorm:"type:varchar(30)"`
	IFSC          *string `gorm:"type:varchar(20)"`

	// UPI
	VPA *string `gorm:"type:varchar(100)"`

	// Card / wallet display
	Last4 *string `gorm:"type:varchar(4)"`
	Brand *string `gorm:"type:varchar(50)"`

	Status PayoutMethodStatus `gorm:"type:varchar(30);not null;default:'pending_verification';index"`

	IsPrimary bool `gorm:"not null;default:false;index"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (p *PayoutMethod) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
