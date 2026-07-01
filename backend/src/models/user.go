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
	Email        string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone        string `gorm:"type:varchar(20);uniqueIndex;not null"`
	PasswordHash string `gorm:"type:text;not null"`
	TwoFASecret string `gorm:"type:text"`

	FullName string `gorm:"type:varchar(255);not null"`
	AccountType AccountType `gorm:"type:varchar(30);not null"`
	Status      UserStatus  `gorm:"type:varchar(30);default:'pending_verification'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = uuid.New()
	return nil
}