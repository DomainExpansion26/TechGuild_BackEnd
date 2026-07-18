package models
// GovtIDDedup prevents multiple accounts from using the same government ID. .
import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type GovtIDDedup struct {
    ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	// Salted SHA-256 hash of Aadhaar/PAN/Passport/Driving License number
    GovtIDHash string `gorm:"size:255;uniqueIndex;not null"`

    UserID uuid.UUID `gorm:"type:uuid;not null"`

    CreatedAt time.Time
}

func (g *GovtIDDedup) BeforeCreate(tx *gorm.DB) error {
    g.ID = uuid.New()
    return nil
}