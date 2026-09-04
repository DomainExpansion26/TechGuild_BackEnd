package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type AgencyProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// Step 1: Agency Information
	AgencyName  string `gorm:"size:255;not null"`
	LogoURL     string `gorm:"type:varchar(255)"`
	Description string `gorm:"type:text"`
	WebsiteURL  string `gorm:"type:varchar(255)"`

	// Step 2: Services
	ServicesOffered pq.StringArray `gorm:"type:text[]"`
	Industries      pq.StringArray `gorm:"type:text[]"`
	TeamSize        string         `gorm:"type:varchar(50)"`

	// Optional Contact Info
	ContactName    string  `gorm:"size:100"`
	Phone          *string `gorm:"type:varchar(20);uniqueIndex"`
	RegistrationNo string  `gorm:"type:varchar(100)"`

	Country           string
	State             string
	City              string
	TimeZone          string
	CountryCode       string
	PublicUrlSlug     string `gorm:"type:varchar(255);uniqueIndex"`
	ProfileVisibility string `gorm:"type:varchar(50);default:'public'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ap *AgencyProfile) BeforeCreate(tx *gorm.DB) error {
	ap.ID = uuid.New()
	return nil
}
