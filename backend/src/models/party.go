package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// type PartyStatus string

// const (
// 	PartyPending  PartyStatus = "pending"
// 	PartyActive   PartyStatus = "active"
// 	PartySuspended PartyStatus = "suspended"
// 	PartyArchived PartyStatus = "archived"
// )

type Party struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Basic Information
	Name        string `gorm:"size:150;not null"`
	Slug        string `gorm:"size:180;uniqueIndex;not null"`
	Description string `gorm:"type:text"`

	LogoURL   string `gorm:"type:text"`
	BannerURL string `gorm:"type:text"`

	// Leader
	LeaderID uuid.UUID `gorm:"type:uuid;not null;index"`
	Leader   User      `gorm:"foreignKey:LeaderID;constraint:OnDelete:CASCADE"`

	// Party Settings
	IsHiring   bool `gorm:"default:false"`
	IsVerified bool `gorm:"default:false"`

	// Status PartyStatus `gorm:"type:varchar(30);default:'pending'"`

	// Relations
	Members     []PartyMember     `gorm:"foreignKey:PartyID"`
	Invitations []PartyInvitation `gorm:"foreignKey:PartyID"`
	Portfolio   []PartyPortfolio  `gorm:"foreignKey:PartyID"`
	Skills      []PartySkill      `gorm:"foreignKey:PartyID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *Party) BeforeCreate(tx *gorm.DB) error {
	t.ID = uuid.New()
	return nil
}