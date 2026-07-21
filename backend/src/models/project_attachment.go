package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectAttachment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Project Relationship
	ProjectID uuid.UUID `gorm:"type:uuid;index;not null"`
	Project   Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`

	// Cloudinary Details
	FileURL  string `gorm:"type:text;not null"`
	PublicID string `gorm:"size:255;not null"`

	// File Information
	FileName    string `gorm:"size:255;not null"`
	ContentType string `gorm:"size:100"`
	FileSize    int64

	CreatedAt time.Time
}

func (p *ProjectAttachment) BeforeCreate(tx *gorm.DB) error {
	p.ID = uuid.New()
	return nil
}