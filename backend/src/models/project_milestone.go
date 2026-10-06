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

	// Amount — int64 minor units of Contract.Currency.
	// Example: INR 9500.00 => 950000 (paise).
	Amount int64 `gorm:"not null;default:0"`

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
