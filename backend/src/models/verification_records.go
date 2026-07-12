package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VerificationType string

const (
	VerificationIndividual VerificationType = "individual"
	VerificationBusiness   VerificationType = "business"
)

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
	VerificationReview   VerificationStatus = "under_review"
)

type VerificationRecord struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;index;not null"`

	User        User               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Type        VerificationType   `gorm:"size:30;not null"`
	Status      VerificationStatus `gorm:"size:30;default:'pending'"`
	Provider    string             `gorm:"size:100"`
	ReferenceID string             `gorm:"size:255"`
	VerifiedAt  *time.Time
	Remarks     string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (v *VerificationRecord) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}
