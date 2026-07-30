package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamStatus string

const (
	TeamPending  TeamStatus = "pending"
	TeamActive   TeamStatus = "active"
	TeamSuspended TeamStatus = "suspended"
	TeamArchived TeamStatus = "archived"
)

type Team struct {
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

	// Team Settings
	IsHiring   bool `gorm:"default:false"`
	IsVerified bool `gorm:"default:false"`

	Status TeamStatus `gorm:"type:varchar(30);default:'pending'"`

	// Relations
	Members     []TeamMember     `gorm:"foreignKey:TeamID"`
	Invitations []TeamInvitation `gorm:"foreignKey:TeamID"`
	Portfolio   []TeamPortfolio  `gorm:"foreignKey:TeamID"`
	Skills      []TeamSkill      `gorm:"foreignKey:TeamID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *Team) BeforeCreate(tx *gorm.DB) error {
	t.ID = uuid.New()
	return nil
}