package models

import "time"

// KYCAttemptCounter is a per-window counter used to enforce KYC rate limits.
// A single row is keyed by (subject_type, subject_hash, window_kind,
// window_start): e.g. ("user", <uuid>, "hour", <hour-truncated timestamp>).
type KYCAttemptCounter struct {
	SubjectType string    `gorm:"primaryKey;size:8"`  // "user" | "ip"
	SubjectHash string    `gorm:"primaryKey;size:64"` // user UUID string or IP hash
	WindowKind  string    `gorm:"primaryKey;size:8"`  // "hour" | "day"
	WindowStart time.Time `gorm:"primaryKey;not null"`
	Count       int64     `gorm:"not null"`
}
