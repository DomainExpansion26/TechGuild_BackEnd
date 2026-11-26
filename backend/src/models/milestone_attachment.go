package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MilestoneAttachment is a deliverable file uploaded against a milestone.
// QuestID is denormalized so quest-scoped listings and counters stay single-scan.
type MilestoneAttachment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	MilestoneID uuid.UUID      `gorm:"type:uuid;not null;index" json:"milestoneId"`
	Milestone   QuestMilestone `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE" json:"-"`

	QuestID uuid.UUID `gorm:"type:uuid;not null;index" json:"questId"`

	UploadedByID uuid.UUID `gorm:"type:uuid;not null;index" json:"uploadedById"`

	FileURL       string `gorm:"type:text;not null" json:"fileUrl"`
	FileName      string `gorm:"type:varchar(255);not null" json:"fileName"`
	FileType      string `gorm:"type:varchar(50);not null" json:"fileType"`
	FileSizeBytes int64  `gorm:"not null" json:"fileSizeBytes"`

	CreatedAt time.Time `json:"createdAt"`
}

func (a *MilestoneAttachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
