package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Wallet — one per user (freelancer only for now).
// Tracks available balance, lifetime earnings, and withdrawal history.
type Wallet struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	User   User      `gorm:"foreignKey:UserID"`

	// AvailableBalanceMinor — withdrawable amount
	AvailableBalanceMinor int64 `gorm:"not null;default:0"`

	// PendingBalanceMinor — held for future (T+2 settlement, dispute holds)
	PendingBalanceMinor int64 `gorm:"not null;default:0"`

	// Lifetime totals
	TotalEarnedMinor    int64 `gorm:"not null;default:0"`
	TotalWithdrawnMinor int64 `gorm:"not null;default:0"`

	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
