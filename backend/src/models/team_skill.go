package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamSkill struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Team Relationship
	TeamID uuid.UUID `gorm:"type:uuid;not null;index"`
	Team   Team      `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE"`

	SkillName string `gorm:"size:100;not null"`

	ExperienceLevel string `gorm:"size:50"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *TeamSkill) BeforeCreate(tx *gorm.DB) error {

	s.ID = uuid.New()

	return nil
}