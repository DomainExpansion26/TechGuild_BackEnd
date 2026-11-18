package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// VerificationAuditLog records every state change in the KYC AI pipeline.
// Used for admin audit trails, bulk-action accountability, and post-incident
// forensics. Never stores raw ID numbers or PII beyond what's already in
// verification_records.
type VerificationAuditLog struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	VerificationID uuid.UUID `gorm:"index"`
	UserID         uuid.UUID `gorm:"index"`
	ActionType     string    `gorm:"size:50"` // ML_JOB_SUBMITTED, ML_JOB_COMPLETED, AI_AUTO_APPROVED,
	//                 BULK_APPROVE, MANUAL_APPROVE, MANUAL_REJECT, AI_AUTO_REJECT,
	//                 ADMIN_REVIEW_REQUESTED, TIMEOUT_REQUEUED, DLQ_ESCALATED
	AdminID   *uuid.UUID   `gorm:"type:uuid"`
	Details   datatypes.JSON `gorm:"type:jsonb"` // {job_id, face_score, liveness_score, reason_code, count, verification_ids}
	IPHash    string `gorm:"size:64"`
	UserAgentHash string `gorm:"size:64"`
	CreatedAt time.Time
}

func (v *VerificationAuditLog) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}
