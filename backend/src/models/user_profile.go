package models

import (
	"time"
	"github.com/google/uuid"
)
type Gender string
const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)
type UserProfile struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	FirstName string `gorm:"size:100;not null"`
	Phone string `gorm:"type:varchar(20);uniqueIndex"`
	LastName  string `gorm:"size:100"`
	DateOfBirth *time.Time
	Gender Gender `gorm:"size:20"`
	AvatarURL string
	Bio string `gorm:"type:text"`
	Country string
	State   string
	City    string
	CreatedAt time.Time
	UpdatedAt time.Time
}