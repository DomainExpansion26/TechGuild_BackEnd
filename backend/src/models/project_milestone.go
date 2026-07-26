package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MilestoneStatus string

const (
	MilestonePending    MilestoneStatus = "pending"
	MilestoneInProgress MilestoneStatus = "in_progress"
	MilestoneSubmitted  MilestoneStatus = "submitted"
	MilestoneApproved   MilestoneStatus = "approved"
	MilestoneRejected   MilestoneStatus = "rejected"
	MilestonePaid       MilestoneStatus = "paid"
)

type ProjectMilestone struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContractID uuid.UUID       `gorm:"type:uuid;not null;index"`
	Contract   ProjectContract `gorm:"foreignKey:ContractID;constraint:OnDelete:CASCADE"`

	Title       string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`

	Amount float64

	DueDate *time.Time

	Status MilestoneStatus `gorm:"type:varchar(30);default:'pending'"`

	CompletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Submissions []ProjectSubmission `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE"`
}

func (m *ProjectMilestone) BeforeCreate(tx *gorm.DB) error {
	m.ID = uuid.New()
	return nil
}