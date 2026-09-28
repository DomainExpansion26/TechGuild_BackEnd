package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PartyPortfolio struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Party Relationship
	PartyID uuid.UUID `gorm:"type:uuid;not null;index"`
	Party   Party      `gorm:"foreignKey:PartyID;constraint:OnDelete:CASCADE"`

	// Portfolio Details
	Title       string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`

	ImageURL string `gorm:"type:text"`

	ProjectURL string `gorm:"type:text"`

	GithubURL string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *PartyPortfolio) BeforeCreate(tx *gorm.DB) error {

	p.ID = uuid.New()

	return nil
}