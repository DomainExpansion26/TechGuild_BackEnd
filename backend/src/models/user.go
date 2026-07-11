package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountType string

const (
	AccountTypeIndividual  AccountType = "individual"
	AccountTypeAgencyAdmin AccountType = "agency_admin"
	AccountTypeClientAdmin AccountType = "client_admin"
)

type UserStatus string

const (
	StatusPendingVerification UserStatus = "pending_verification"
	StatusActive              UserStatus = "active"
	StatusSuspended           UserStatus = "suspended"
	StatusRejected            UserStatus = "rejected"
)

type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	FullName string `gorm:"type:varchar(255);not null"`

	Email string `gorm:"type:varchar(255);uniqueIndex;not null"`

	Phone string `gorm:"type:varchar(20);uniqueIndex"`

	PasswordHash string `gorm:"type:text"`

	TwoFASecret string `gorm:"type:text"`

	AccountType *AccountType `gorm:"type:varchar(30)"`

	Status UserStatus `gorm:"type:varchar(30);default:'pending_verification'"`

	EmailVerified bool `gorm:"default:false"`

	OAuthProvider string `gorm:"type:varchar(20)"`

	OAuthID string `gorm:"type:varchar(255);uniqueIndex"`

	CreatedAt time.Time

	UpdatedAt time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = uuid.New()
	return nil
}