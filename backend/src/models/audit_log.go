package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuditLog struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID *uuid.UUID `gorm:"type:uuid;index"`

	Action string `gorm:"size:100;not null"`
	Entity string `gorm:"size:100"`
	EntityID string `gorm:"size:100"`
	IPAddress string `gorm:"size:45"`
	UserAgent string `gorm:"type:text"`
	Metadata datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
}
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	a.ID = uuid.New()
	return nil
}