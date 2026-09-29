// models/user_recovery_code.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRecoveryCode struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	CodeHash string `gorm:"type:text;not null"`
	IsUsed   bool   `gorm:"default:false;not null"`
	UsedAt   *time.Time

	CreatedAt time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (r *UserRecoveryCode) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
