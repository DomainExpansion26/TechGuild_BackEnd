package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentCustomer struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uniq_user_provider_customer"`

	Provider           string `gorm:"type:varchar(50);not null;uniqueIndex:uniq_user_provider_customer;uniqueIndex:uniq_provider_customer_id"`
	ProviderCustomerID string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_provider_customer_id"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (pc *PaymentCustomer) BeforeCreate(tx *gorm.DB) error {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	return nil
}
