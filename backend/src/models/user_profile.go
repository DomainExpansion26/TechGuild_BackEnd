package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

type UserProfile struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User              User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	FirstName         string    `gorm:"size:100;not null"`
	LastName          string    `gorm:"size:100"`
	DateOfBirth       *time.Time
	Gender            Gender `gorm:"size:20"`
	AvatarURL         string
	Bio               string `gorm:"type:text"`
	Country           string
	State             string
	City              string
	Headline          string
	PreferredLanguage string
	TimeZone          string
	CountryCode       string
	PublicUrlSlug     string `gorm:"type:varchar(255);uniqueIndex"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (up *UserProfile) BeforeCreate(tx *gorm.DB) error {
	up.ID = uuid.New()
	return nil
}
