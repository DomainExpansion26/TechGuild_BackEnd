package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamPortfolio struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Team Relationship
	TeamID uuid.UUID `gorm:"type:uuid;not null;index"`
	Team   Team      `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE"`

	// Portfolio Details
	Title       string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`

	ImageURL string `gorm:"type:text"`

	ProjectURL string `gorm:"type:text"`

	GithubURL string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *TeamPortfolio) BeforeCreate(tx *gorm.DB) error {

	p.ID = uuid.New()

	return nil
}