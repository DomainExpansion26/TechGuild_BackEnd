package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectSkill struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Relationship
	ProjectID uuid.UUID `gorm:"type:uuid;index;not null"`
	Project   Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`

	// Skill Name
	Skill string `gorm:"size:100;not null;index"`

	CreatedAt time.Time
}

func (ps *ProjectSkill) BeforeCreate(tx *gorm.DB) error {
	ps.ID = uuid.New()
	return nil
}