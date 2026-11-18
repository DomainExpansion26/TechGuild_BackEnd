package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// ErrRateLimited is the sentinel returned (via RateLimitError.Unwrap) when a
// reserve attempt exceeds a configured limit.
var ErrRateLimited = errors.New("rate limited")

// RateLimitError carries which scope triggered the limit and how long the
// caller should wait before retrying.
type RateLimitError struct {
	Scope      string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: %s (retry after %s)", e.Scope, e.RetryAfter)
}

func (e *RateLimitError) Unwrap() error { return ErrRateLimited }

// RateLimits is the set of limits enforced for a single reserve attempt.
type RateLimits struct {
	UserHourly int
	UserDaily  int
	IPHourly   int
	IPDaily    int
}

// ReserveAttemptRequest describes one attempt to reserve a rate-limit slot.
// IPHash may be empty, in which case IP-scoped limits are skipped.
type ReserveAttemptRequest struct {
	UserID         uuid.UUID
	VerificationID *uuid.UUID
	Action         models.VerificationAttemptAction
	IPHash         string // HMAC hex, empty = skip IP bucket
	UserAgentHash  string
	RequestID      string
	Limits         RateLimits
}

// AttemptsRepository reserves and compensates KYC rate-limit attempts.
type AttemptsRepository interface {
	// ReserveAttempt runs inside the caller's transaction (tx). The caller is
	// responsible for BEGIN/COMMIT and any advisory locks. On success it
	// increments all applicable counters and writes an outcome='allowed'
	// verification_attempts row. When a limit is exceeded it returns a
	// *RateLimitError (Unwrap -> ErrRateLimited) and leaves no attempt row.
	ReserveAttempt(ctx context.Context, tx *gorm.DB, req ReserveAttemptRequest) error

	// MarkFailed is the compensating update when the Redis publish fails after
	// ReserveAttempt succeeded. It flips the attempt to outcome='failed' and
	// releases the slots it had consumed so the row stops counting toward
	// limits.
	MarkFailed(ctx context.Context, requestID string) error
}

type attemptsRepository struct{}

// NewAttemptsRepository returns the default AttemptsRepository implementation.
func NewAttemptsRepository() AttemptsRepository {
	return &attemptsRepository{}
}

type attemptCheck struct {
	subjectType  string
	subjectHash  string
	windowKind   string
	windowStart  time.Time
	windowDur    time.Duration
	subjectCol   string
	subjectValue any
	scope        string
	limit        int
}

func (r *attemptsRepository) ReserveAttempt(ctx context.Context, tx *gorm.DB, req ReserveAttemptRequest) error {
	if tx == nil {
		return errors.New("attempts: nil transaction")
	}

	now := time.Now().UTC()
	hourStart := now.Truncate(time.Hour)
	dayStart := now.Truncate(24 * time.Hour)
	userHash := req.UserID.String()

	// User checks first so a user hitting their own limit gets a user-scoped
	// error rather than an ip-scoped one.
	checks := []attemptCheck{
		{
			subjectType: "user", subjectHash: userHash,
			windowKind: "hour", windowStart: hourStart, windowDur: time.Hour,
			subjectCol: "user_id", subjectValue: req.UserID,
			scope: "user_hourly", limit: req.Limits.UserHourly,
		},
		{
			subjectType: "user", subjectHash: userHash,
			windowKind: "day", windowStart: dayStart, windowDur: 24 * time.Hour,
			subjectCol: "user_id", subjectValue: req.UserID,
			scope: "user_daily", limit: req.Limits.UserDaily,
		},
	}

	if req.IPHash != "" {
		checks = append(checks,
			attemptCheck{
				subjectType: "ip", subjectHash: req.IPHash,
				windowKind: "hour", windowStart: hourStart, windowDur: time.Hour,
				subjectCol: "ip_hash", subjectValue: req.IPHash,
				scope: "ip_hourly", limit: req.Limits.IPHourly,
			},
			attemptCheck{
				subjectType: "ip", subjectHash: req.IPHash,
				windowKind: "day", windowStart: dayStart, windowDur: 24 * time.Hour,
				subjectCol: "ip_hash", subjectValue: req.IPHash,
				scope: "ip_daily", limit: req.Limits.IPDaily,
			},
		)
	}

	for _, check := range checks {
		// A successful upsert RETURNINGs the new count (>= 1). When the
		// WHERE guard fails, the statement is a no-op and returns no row, so
		// the scanned value stays 0. (Exec().RowsAffected is unreliable for
		// INSERT ... RETURNING across drivers, notably SQLite.)
		var newCount int64
		err := tx.WithContext(ctx).Raw(
			`INSERT INTO kyc_attempt_counters
				(subject_type, subject_hash, window_kind, window_start, count)
			 VALUES (?, ?, ?, ?, 1)
			 ON CONFLICT (subject_type, subject_hash, window_kind, window_start)
			 DO UPDATE SET count = kyc_attempt_counters.count + 1
			 WHERE kyc_attempt_counters.count < ?
			 RETURNING count`,
			check.subjectType, check.subjectHash, check.windowKind, check.windowStart, check.limit,
		).Scan(&newCount).Error
		if err != nil {
			return err
		}
		if newCount == 0 {
			return &RateLimitError{
				Scope:      check.scope,
				RetryAfter: r.retryAfter(tx, check),
			}
		}
	}

	attempt := models.VerificationAttempt{
		UserID:         req.UserID,
		VerificationID: req.VerificationID,
		Action:         req.Action,
		Outcome:        models.AttemptOutcomeAllowed,
		IPHash:         req.IPHash,
		UserAgentHash:  req.UserAgentHash,
		RequestID:      req.RequestID,
		CreatedAt:      now,
	}
	return tx.WithContext(ctx).Create(&attempt).Error
}

func (r *attemptsRepository) retryAfter(tx *gorm.DB, check attemptCheck) time.Duration {
	// ORDER BY + LIMIT rather than MIN() so the timestamp is scanned with the
	// column's native type across drivers (SQLite returns MIN(text) as text).
	var oldest struct {
		CreatedAt time.Time
	}
	tx.WithContext(context.Background()).Raw(
		`SELECT created_at FROM verification_attempts
		  WHERE outcome = ? AND `+check.subjectCol+` = ? AND created_at >= ?
		  ORDER BY created_at ASC
		  LIMIT 1`,
		string(models.AttemptOutcomeAllowed), check.subjectValue, check.windowStart,
	).Scan(&oldest)

	if oldest.CreatedAt.IsZero() {
		return check.windowDur
	}

	retry := check.windowStart.Add(check.windowDur).Sub(oldest.CreatedAt.UTC())
	if retry < time.Second {
		retry = time.Second
	}
	return retry
}

func (r *attemptsRepository) MarkFailed(ctx context.Context, requestID string) error {
	db := postgres.DB.WithContext(ctx)

	var attempt models.VerificationAttempt
	err := db.Where("request_id = ? AND outcome = ?", requestID, models.AttemptOutcomeAllowed).
		First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.VerificationAttempt{}).
			Where("request_id = ? AND outcome = ?", requestID, models.AttemptOutcomeAllowed).
			Update("outcome", models.AttemptOutcomeFailed)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		created := attempt.CreatedAt.UTC()
		if err := r.releaseCounter(tx, "user", attempt.UserID.String(), "hour", created.Truncate(time.Hour)); err != nil {
			return err
		}
		if err := r.releaseCounter(tx, "user", attempt.UserID.String(), "day", created.Truncate(24*time.Hour)); err != nil {
			return err
		}
		if attempt.IPHash != "" {
			if err := r.releaseCounter(tx, "ip", attempt.IPHash, "hour", created.Truncate(time.Hour)); err != nil {
				return err
			}
			if err := r.releaseCounter(tx, "ip", attempt.IPHash, "day", created.Truncate(24*time.Hour)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *attemptsRepository) releaseCounter(tx *gorm.DB, subjectType, subjectHash, windowKind string, windowStart time.Time) error {
	return tx.Exec(
		`UPDATE kyc_attempt_counters
		    SET count = CASE WHEN count > 0 THEN count - 1 ELSE 0 END
		  WHERE subject_type = ? AND subject_hash = ? AND window_kind = ? AND window_start = ?`,
		subjectType, subjectHash, windowKind, windowStart,
	).Error
}
