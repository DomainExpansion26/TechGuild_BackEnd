// models/user_two_factor_authentication.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TwoFAStatus string

const (
	TwoFAStatusDisabled TwoFAStatus = "disabled"
	TwoFAStatusPending  TwoFAStatus = "pending"
	TwoFAStatusEnabled  TwoFAStatus = "enabled"
)

type UserTwoFactorAuthentication struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	SecretEncrypted string      `gorm:"type:text;not null"`
	Status          TwoFAStatus `gorm:"type:varchar(20);not null;default:'disabled'"`

	VerifiedAt       *time.Time
	PendingExpiresAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (t *UserTwoFactorAuthentication) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
