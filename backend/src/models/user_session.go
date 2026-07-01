package models
import (
	"time"
	"github.com/google/uuid"
)

type UserSession struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RefreshToken string `gorm:"type:text;not null"`
	IPAddress string
	UserAgent string
	Device    string
	ExpiresAt time.Time
	LastUsedAt time.Time
	CreatedAt time.Time
}