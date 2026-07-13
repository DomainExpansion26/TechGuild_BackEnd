package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountType string

const (
	AccountTypeIndividual  AccountType = "individual"
	AccountTypeAgencyAdmin AccountType = "agency"
	AccountTypeClientAdmin AccountType = "client"
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

	FirstName string `gorm:"type:varchar(100);not null"`
	LastName  string `gorm:"type:varchar(100)"`

	Email string `gorm:"type:varchar(255);uniqueIndex;not null"`

	PasswordHash string `gorm:"type:text"`

	TwoFASecret string `gorm:"type:text"`

	AccountType *AccountType `gorm:"type:varchar(30)"`

	Status UserStatus `gorm:"type:varchar(30);default:'pending_verification'"`

	EmailVerified bool `gorm:"default:false"`

	OAuthProvider *string `gorm:"type:varchar(20)"`

	OAuthID *string `gorm:"type:varchar(255);uniqueIndex"`

	Points int `gorm:"default:0"`

	CreatedAt time.Time

	UpdatedAt time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = uuid.New()
	return nil
}
