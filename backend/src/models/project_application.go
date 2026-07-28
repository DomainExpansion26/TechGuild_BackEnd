package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	ApplicationPending     ApplicationStatus = "pending"
	ApplicationShortlisted ApplicationStatus = "shortlisted"
	ApplicationAccepted    ApplicationStatus = "accepted"
	ApplicationRejected    ApplicationStatus = "rejected"
	ApplicationWithdrawn   ApplicationStatus = "withdrawn"
)

type ProjectApplication struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// project and applicant relationship

	ProjectID uuid.UUID `gorm:"type:uuid;not null;index"`
	Project   Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`

	ApplicantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Applicant   User       `gorm:"foreignKey:ApplicantID;constraint:OnDelete:CASCADE"`

	//proposal details

	CoverLetter string `gorm:"type:text;not null"`

	ProposedBudget float64 `gorm:"not null"`

	Currency string `gorm:"size:10;default:'INR'"`

	EstimatedDuration string `gorm:"size:100"`

	//project contract relationship
	Contract *ProjectContract `gorm:"foreignKey:ApplicationID"`
	// Application Status

	Status ApplicationStatus `gorm:"type:varchar(30);default:'pending'"`

	ClientMessage string `gorm:"type:text"`

	// dates of application

	AppliedAt time.Time

	ReviewedAt *time.Time

	CreatedAt time.Time

	UpdatedAt time.Time
}

func (p *ProjectApplication) BeforeCreate(tx *gorm.DB) error {

	p.ID = uuid.New()

	if p.AppliedAt.IsZero() {
		p.AppliedAt = time.Now()
	}

	return nil
}