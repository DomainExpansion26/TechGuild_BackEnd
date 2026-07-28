package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubmissionStatus string

const (
	SubmissionPending  SubmissionStatus = "pending"
	SubmissionApproved SubmissionStatus = "approved"
	SubmissionRejected SubmissionStatus = "rejected"
)

type ProjectSubmission struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	MilestoneID uuid.UUID        `gorm:"type:uuid;not null;index"`
	Milestone   ProjectMilestone `gorm:"foreignKey:MilestoneID;constraint:OnDelete:CASCADE"`

	Message string `gorm:"type:text"`

	SubmissionURL string `gorm:"type:text"`

	AttachmentURL string `gorm:"type:text"`

	Status SubmissionStatus `gorm:"type:varchar(30);default:'pending'"`

	SubmittedAt time.Time

	ReviewedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *ProjectSubmission) BeforeCreate(tx *gorm.DB) error {
	s.ID = uuid.New()

	if s.SubmittedAt.IsZero() {
		s.SubmittedAt = time.Now()
	}

	return nil
}