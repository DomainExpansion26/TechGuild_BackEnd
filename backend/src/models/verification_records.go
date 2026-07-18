package models
// VerificationRecord stores the lifecycle of a user verification request.
// Each submission creates a new verification record.
import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
// VerificationType represents the type of verification submitted.
type VerificationType string

const (
	VerificationIndividual VerificationType = "individual"
	VerificationBusiness   VerificationType = "business"
)
// VerificationStatus represents the current state of a verification request.
type VerificationStatus string

const (
	VerificationPending   VerificationStatus = "pending"
	VerificationReview    VerificationStatus = "under_review"
	VerificationApproved  VerificationStatus = "approved"
	VerificationRejected  VerificationStatus = "rejected"
)
// VerificationRecord tracks every verification request submitted by a user.
type VerificationRecord struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	Type   VerificationType   `gorm:"size:30;not null"`
	Status VerificationStatus `gorm:"size:30;default:'pending'"`

	Vendor            string `gorm:"size:100"`
	VendorReferenceID string `gorm:"size:255"`
	GovtIDHash string `gorm:"size:64"`

	BusinessPANHash string `gorm:"size:64"`
	// Reason returned by admin if rejected
	RejectionReason string `gorm:"type:text"`
	// Previous verification record in case of resubmission
	PreviousRecordID *uuid.UUID `gorm:"type:uuid"`
	// Time at which verification was approved
	VerifiedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (v *VerificationRecord) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}