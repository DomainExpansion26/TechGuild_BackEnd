package models

import (
	"time"

	"github.com/google/uuid"
)

// VerificationAttemptAction is the user-facing action that consumed a rate
// limit slot.
type VerificationAttemptAction string

const (
	AttemptActionSubmit VerificationAttemptAction = "submit"
	AttemptActionUpload VerificationAttemptAction = "upload"
)

// VerificationAttemptOutcome records whether the attempt was allowed through
// or blocked (and, for compensating updates, whether a previously allowed
// attempt later failed).
type VerificationAttemptOutcome string

const (
	AttemptOutcomeAllowed     VerificationAttemptOutcome = "allowed"
	AttemptOutcomeRateLimited VerificationAttemptOutcome = "rate_limited"
	AttemptOutcomeFailed      VerificationAttemptOutcome = "failed"
)

// VerificationAttempt is the append-only audit trail of KYC submit/upload
// attempts. Only rows with outcome='allowed' count toward rate limits; a
// row flipped to 'failed' (e.g. Redis publish failure after reserving) stops
// counting.
type VerificationAttempt struct {
	ID             int64                      `gorm:"primaryKey;autoIncrement"`
	UserID         uuid.UUID                  `gorm:"type:uuid;not null;index"`
	VerificationID *uuid.UUID                 `gorm:"type:uuid"`
	Action         VerificationAttemptAction  `gorm:"size:32;not null"`
	Outcome        VerificationAttemptOutcome `gorm:"size:32;not null;index"`
	IPHash         string                     `gorm:"size:64;not null;index"`
	UserAgentHash  string                     `gorm:"size:64"`
	RequestID      string                     `gorm:"size:64"`
	CreatedAt      time.Time                  `gorm:"not null;index"`
}
