//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"techguild-backend/src/models"
)

// TestReserveAttempt_Postgres verifies ON CONFLICT ... WHERE ... RETURNING
// works on real Postgres (SQLite cannot prove the dialect semantics).
// Requires POSTGRES_TEST_DSN; skips if empty. Not run by `go test ./...`.
func TestReserveAttempt_Postgres(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DSN to run this integration test")
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	// Mirror migrateVerificationAttempts (migration package is unexported to
	// this package, so the production DDL is repeated verbatim).
	stmts := []string{
		`DROP TABLE IF EXISTS verification_attempts`,
		`DROP TABLE IF EXISTS kyc_attempt_counters`,
		`CREATE TABLE verification_attempts (
			id BIGSERIAL PRIMARY KEY,
			user_id UUID NOT NULL,
			verification_id UUID,
			action VARCHAR(32) NOT NULL,
			outcome VARCHAR(32) NOT NULL,
			ip_hash VARCHAR(64) NOT NULL,
			user_agent_hash VARCHAR(64),
			request_id VARCHAR(64),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE kyc_attempt_counters (
			subject_type VARCHAR(8) NOT NULL,
			subject_hash VARCHAR(64) NOT NULL,
			window_kind VARCHAR(8) NOT NULL,
			window_start TIMESTAMPTZ NOT NULL,
			count BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (subject_type, subject_hash, window_kind, window_start)
		)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare schema: %v", err)
		}
	}

	repo := NewAttemptsRepository()
	ctx := context.Background()
	userID := uuid.New()
	req := ReserveAttemptRequest{
		UserID:    userID,
		Action:    models.AttemptActionSubmit,
		RequestID: "pg-" + uuid.NewString(),
		Limits:    RateLimits{UserHourly: 3, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	reserve := func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			return repo.ReserveAttempt(ctx, tx, req)
		})
	}

	for i := 0; i < 3; i++ {
		req.RequestID = fmt.Sprintf("pg-%s-%d", userID, i)
		if err := reserve(); err != nil {
			t.Fatalf("reserve #%d: %v", i+1, err)
		}
	}

	req.RequestID = fmt.Sprintf("pg-%s-blocked", userID)
	err = reserve()
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("4th reserve error = %T %v, want *RateLimitError", err, err)
	}
	if rlErr.Scope != "user_hourly" {
		t.Errorf("scope = %q, want user_hourly", rlErr.Scope)
	}

	var hourly int64
	if err := db.Raw(
		`SELECT count FROM kyc_attempt_counters
		  WHERE subject_type = 'user' AND subject_hash = ? AND window_kind = 'hour'`,
		userID.String(),
	).Scan(&hourly).Error; err != nil {
		t.Fatalf("read hourly counter: %v", err)
	}
	if hourly != 3 {
		t.Errorf("hourly counter = %d, want 3", hourly)
	}

	var allowed int64
	if err := db.Raw(
		`SELECT COUNT(*) FROM verification_attempts WHERE outcome = 'allowed'`,
	).Scan(&allowed).Error; err != nil {
		t.Fatalf("count allowed attempts: %v", err)
	}
	if allowed != 3 {
		t.Errorf("allowed attempt rows = %d, want 3", allowed)
	}
}