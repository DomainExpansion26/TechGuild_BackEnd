package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

type IndividualProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	Phone             *string `gorm:"type:varchar(20);uniqueIndex"`
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

	ExperienceLevel string `gorm:"type:varchar(50)"`
	Availability    string `gorm:"type:varchar(50)"`

	Skills            pq.StringArray `gorm:"type:text[]"`
	ToolsTechnologies pq.StringArray `gorm:"type:text[]"`
	ServiceCategories pq.StringArray `gorm:"type:text[]"`

	PortfolioURL string `gorm:"type:varchar(255)"`
	GithubURL    string `gorm:"type:varchar(255)"`
	LinkedinURL  string `gorm:"type:varchar(255)"`
	ResumeURL    string `gorm:"type:varchar(255)"`

	TermsConfirmed    bool   `gorm:"default:false"`
	ProfileVisibility string `gorm:"type:varchar(50);default:'public'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ip *IndividualProfile) BeforeCreate(tx *gorm.DB) error {
	ip.ID = uuid.New()
	return nil
}
