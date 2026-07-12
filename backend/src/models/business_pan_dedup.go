package models
// BusinessPANDedup ensures one business PAN cannot be registered with multiple organizations.
import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type BusinessPANDedup struct {
    ID uuid.UUID `gorm:"type:uuid;primaryKey"`

    PANHash string `gorm:"size:255;uniqueIndex;not null"`

    UserID uuid.UUID `gorm:"type:uuid;not null"`

    CreatedAt time.Time
}

func (b *BusinessPANDedup) BeforeCreate(tx *gorm.DB) error {
    b.ID = uuid.New()
    return nil
}