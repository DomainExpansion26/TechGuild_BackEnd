package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClientPaymentMethodType string

const (
	ClientPaymentMethodTypeCard       ClientPaymentMethodType = "card"
	ClientPaymentMethodTypeUPI        ClientPaymentMethodType = "upi"
	ClientPaymentMethodTypeNetbanking ClientPaymentMethodType = "netbanking"
)

type ClientPaymentMethodStatus string

const (
	ClientPaymentMethodStatusActive   ClientPaymentMethodStatus = "active"
	ClientPaymentMethodStatusExpired  ClientPaymentMethodStatus = "expired"
	ClientPaymentMethodStatusDisabled ClientPaymentMethodStatus = "disabled"
	ClientPaymentMethodStatusFailed   ClientPaymentMethodStatus = "failed"
)

type ClientPaymentMethod struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	PaymentCustomerID uuid.UUID `gorm:"type:uuid;not null;index"`

	Provider         string `gorm:"type:varchar(50);not null;uniqueIndex:uniq_provider_client_method_id"`
	ProviderMethodID string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_provider_client_method_id"`

	Type ClientPaymentMethodType `gorm:"type:varchar(20);not null"`

	// Display-only card metadata
	Last4    *string `gorm:"type:varchar(4)"`
	Brand    *string `gorm:"type:varchar(50)"`
	ExpMonth *int    `gorm:"type:int"`
	ExpYear  *int    `gorm:"type:int"`

	// UPI display metadata
	VPAMasked *string `gorm:"type:varchar(100)"`

	IsPrimary bool                      `gorm:"not null;default:false;index"`
	Status    ClientPaymentMethodStatus `gorm:"type:varchar(30);not null;default:'active'"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (m *ClientPaymentMethod) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
