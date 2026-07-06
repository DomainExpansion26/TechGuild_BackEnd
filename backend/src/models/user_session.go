package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSession struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null"`
	RefreshToken string         `gorm:"type:text;not null"`
	Device       string         `gorm:"size:255"`
	IPAddress    string         `gorm:"size:100"`
	IsRevoked    bool           `gorm:"default:false"`
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}