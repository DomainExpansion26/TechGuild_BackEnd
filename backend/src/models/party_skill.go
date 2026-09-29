package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PartySkill struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Party Relationship
	PartyID uuid.UUID `gorm:"type:uuid;not null;index"`
	Party   Party      `gorm:"foreignKey:PartyID;constraint:OnDelete:CASCADE"`

	SkillName string `gorm:"size:100;not null"`

	ExperienceLevel string `gorm:"size:50"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *PartySkill) BeforeCreate(tx *gorm.DB) error {

	s.ID = uuid.New()

	return nil
}