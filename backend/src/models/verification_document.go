package models

// VerificationDocument stores uploaded verification documents.
// Only the document metadata is stored here.
// Actual files are stored securely in Cloudinary.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VerificationDocument struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	VerificationRecordID uuid.UUID `gorm:"type:uuid;index;not null"`

	VerificationRecord VerificationRecord `gorm:"foreignKey:VerificationRecordID;constraint:OnDelete:CASCADE"`

	// government_id, selfie, gst_certificate,
	// pan_card, authorized_representative_id
	DocumentType string `gorm:"size:50;not null"`

	// Cloudinary secure URL
	FileURL string `gorm:"type:text;not null"`

	// Cloudinary Public ID (used for delete/update)
	PublicID string `gorm:"size:255"`

	// image/jpeg, image/png, application/pdf
	ContentType string `gorm:"size:100"`

	CreatedAt time.Time
}

func (v *VerificationDocument) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}
