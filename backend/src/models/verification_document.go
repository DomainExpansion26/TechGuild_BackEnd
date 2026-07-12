package models
// VerificationDocument stores encrypted document references
// uploaded during the verification process.
// Actual files are stored securely in Cloudflare.
import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type VerificationDocument struct {
    ID uuid.UUID `gorm:"type:uuid;primaryKey"`

    VerificationRecordID uuid.UUID `gorm:"type:uuid;index;not null"`

    DocumentType string `gorm:"size:50;not null"`

    FileURL string `gorm:"type:text;not null"`

    CreatedAt time.Time
}

func (v *VerificationDocument) BeforeCreate(tx *gorm.DB) error {
    v.ID = uuid.New()
    return nil
}