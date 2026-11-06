package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BulkReviewItem tracks which verification records were actioned together in a
// single admin bulk action. Each verification record must be unique within this
// table, and a row is inserted whenever an admin explicitly requests human
// review (ADMIN_REVIEW_REQUESTED), even for items that would otherwise be
// auto-eligible. This preserves a human-decision record for the 24-48h window.
type BulkReviewItem struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	VerificationID uuid.UUID `gorm:"uniqueIndex;not null"`
	UserEmail      string
	Type           string // individual | business
	ReviewedAt     *time.Time
	ReviewedBy     *uuid.UUID
	ReviewedAction string // approve | reject (recorded for bulk-approved items)
	CreatedAt      time.Time
}

func (b *BulkReviewItem) BeforeCreate(tx *gorm.DB) error {
	b.ID = uuid.New()
	return nil
}
