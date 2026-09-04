package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ClientProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// Step 1: Company Information
	CompanyName string `gorm:"size:255;not null"`
	LogoURL     string `gorm:"type:varchar(255)"`
	Industry    string `gorm:"type:varchar(100)"`
	WebsiteURL  string `gorm:"type:varchar(255)"`

	// Step 2: Hiring Preferences
	ProjectTypes pq.StringArray `gorm:"type:text[]"`
	BudgetRange  string         `gorm:"type:varchar(50)"`
	TeamSize     string         `gorm:"type:varchar(50)"`

	// Optional Contact Info
	ContactName string  `gorm:"size:100"`
	Phone       *string `gorm:"type:varchar(20);uniqueIndex"`

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

func (cp *ClientProfile) BeforeCreate(tx *gorm.DB) error {
	cp.ID = uuid.New()
	return nil
}
