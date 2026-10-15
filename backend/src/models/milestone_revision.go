package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MilestoneRevision struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	MilestoneID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Milestone   QuestMilestone `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE"`

	RequestedByID uuid.UUID `gorm:"type:uuid;not null;index"`
	RequestedBy   User      `gorm:"foreignKey:RequestedByID"`

	TitleSnapshot       string `gorm:"type:varchar(255)"`
	DescriptionSnapshot string `gorm:"type:text"`
	AmountMinorSnapshot int64

	Note          string `gorm:"type:text"`
	AttachmentURL string `gorm:"type:text"`
	RoundNumber   int    `gorm:"not null;default:1"`

	CreatedAt time.Time
}

func (r *MilestoneRevision) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
