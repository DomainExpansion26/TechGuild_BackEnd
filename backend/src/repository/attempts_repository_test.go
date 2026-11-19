package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// setupAttemptsTestDB spins up an isolated in-memory SQLite database and swaps
// it into postgres.DB for the duration of the test.
func setupAttemptsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.VerificationAttempt{},
		&models.KYCAttemptCounter{},
	); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })

	return db
}

func reserveOnce(t *testing.T, repo AttemptsRepository, req ReserveAttemptRequest) error {
	t.Helper()
	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		return repo.ReserveAttempt(context.Background(), tx, req)
	})
}

func seedCounter(t *testing.T, subjectType, subjectHash, windowKind string, windowStart time.Time, count int64) {
	t.Helper()
	if err := postgres.DB.Create(&models.KYCAttemptCounter{
		SubjectType: subjectType,
		SubjectHash: subjectHash,
		WindowKind:  windowKind,
		WindowStart: windowStart,
		Count:       count,
	}).Error; err != nil {
		t.Fatalf("seed counter: %v", err)
	}
}

func seedAllowedAttempt(t *testing.T, userID uuid.UUID, ipHash string, createdAt time.Time) {
	t.Helper()
	if err := postgres.DB.Create(&models.VerificationAttempt{
		UserID:    userID,
		Action:    models.AttemptActionSubmit,
		Outcome:   models.AttemptOutcomeAllowed,
		IPHash:    ipHash,
		RequestID: "seed-" + uuid.NewString(),
		CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("seed attempt: %v", err)
	}
}

func countCounters(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := postgres.DB.Raw("SELECT COUNT(*) FROM kyc_attempt_counters").Scan(&n).Error; err != nil {
		t.Fatalf("count counters: %v", err)
	}
	return n
}

func TestReserve_AllowsUnderLimit(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()

	req := ReserveAttemptRequest{
		UserID:    uuid.New(),
		Action:    models.AttemptActionSubmit,
		RequestID: "r-allow",
		Limits:    RateLimits{UserHourly: 3, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	if err := reserveOnce(t, repo, req); err != nil {
		t.Fatalf("first ReserveAttempt should succeed, got %v", err)
	}
}

func TestReserve_BlocksWhenHourlyExceeded(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()

	now := time.Now().UTC()
	userID := uuid.New()
	seedCounter(t, "user", userID.String(), "hour", now.Truncate(time.Hour), 3)
	seedAllowedAttempt(t, userID, "", now)

	req := ReserveAttemptRequest{
		UserID:    userID,
		Action:    models.AttemptActionSubmit,
		RequestID: "r-hour",
		Limits:    RateLimits{UserHourly: 3, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	err := reserveOnce(t, repo, req)
	if err == nil {
		t.Fatal("expected RateLimitError, got nil")
	}

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if !strings.Contains(rle.Scope, "user_hourly") {
		t.Fatalf("expected scope containing user_hourly, got %q", rle.Scope)
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatal("expected error to unwrap to ErrRateLimited")
	}
}

func TestReserve_BlocksWhenDailyExceeded(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()

	now := time.Now().UTC()
	userID := uuid.New()
	seedCounter(t, "user", userID.String(), "day", now.Truncate(24*time.Hour), 10)
	seedAllowedAttempt(t, userID, "", now)

	req := ReserveAttemptRequest{
		UserID:    userID,
		Action:    models.AttemptActionSubmit,
		RequestID: "r-day",
		Limits:    RateLimits{UserHourly: 1000, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	err := reserveOnce(t, repo, req)
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if !strings.Contains(rle.Scope, "user_daily") {
		t.Fatalf("expected scope containing user_daily, got %q", rle.Scope)
	}
}

func TestReserve_RetryAfterPositive(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()

	now := time.Now().UTC()
	userID := uuid.New()
	seedCounter(t, "user", userID.String(), "hour", now.Truncate(time.Hour), 1)
	seedAllowedAttempt(t, userID, "", now)

	req := ReserveAttemptRequest{
		UserID:    userID,
		Action:    models.AttemptActionSubmit,
		RequestID: "r-retry",
		Limits:    RateLimits{UserHourly: 1, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	err := reserveOnce(t, repo, req)
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rle.RetryAfter <= 0 {
		t.Fatalf("expected positive RetryAfter, got %s", rle.RetryAfter)
	}
}

func TestReserve_SkipsIPWhenHashEmpty(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()

	req := ReserveAttemptRequest{
		UserID:    uuid.New(),
		Action:    models.AttemptActionSubmit,
		IPHash:    "",
		RequestID: "r-noip",
		// IP limits of zero would block instantly if IP checks were enforced.
		Limits: RateLimits{UserHourly: 5, UserDaily: 100, IPHourly: 0, IPDaily: 0},
	}

	if err := reserveOnce(t, repo, req); err != nil {
		t.Fatalf("ReserveAttempt with empty IPHash should skip IP checks, got %v", err)
	}

	var n int64
	if err := postgres.DB.Raw(
		"SELECT COUNT(*) FROM kyc_attempt_counters WHERE subject_type = 'ip'",
	).Scan(&n).Error; err != nil {
		t.Fatalf("count ip counters: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected no ip counter rows, got %d", n)
	}
}

func TestMarkFailed_RemovesFromCounting(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()
	ctx := context.Background()

	req := ReserveAttemptRequest{
		UserID:    uuid.New(),
		Action:    models.AttemptActionSubmit,
		RequestID: "r-mark",
		Limits:    RateLimits{UserHourly: 1, UserDaily: 1, IPHourly: 10, IPDaily: 30},
	}

	if err := reserveOnce(t, repo, req); err != nil {
		t.Fatalf("first ReserveAttempt should succeed, got %v", err)
	}
	if err := repo.MarkFailed(ctx, "r-mark"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	req.RequestID = "r-mark-2"
	if err := reserveOnce(t, repo, req); err != nil {
		t.Fatalf("ReserveAttempt after MarkFailed should succeed, got %v", err)
	}

	var outcome models.VerificationAttemptOutcome
	if err := postgres.DB.Raw(
		"SELECT outcome FROM verification_attempts WHERE request_id = ?", "r-mark",
	).Scan(&outcome).Error; err != nil {
		t.Fatalf("read failed outcome: %v", err)
	}
	if outcome != models.AttemptOutcomeFailed {
		t.Fatalf("expected outcome failed, got %q", outcome)
	}
}

func TestReserve_TransactionScoped(t *testing.T) {
	setupAttemptsTestDB(t)
	repo := NewAttemptsRepository()
	ctx := context.Background()

	req := ReserveAttemptRequest{
		UserID:    uuid.New(),
		Action:    models.AttemptActionSubmit,
		RequestID: "r-rollback",
		Limits:    RateLimits{UserHourly: 5, UserDaily: 10, IPHourly: 10, IPDaily: 30},
	}

	sentinel := errors.New("force rollback")
	err := postgres.DB.Transaction(func(tx *gorm.DB) error {
		if err := repo.ReserveAttempt(ctx, tx, req); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel rollback error, got %v", err)
	}

	if n := countCounters(t); n != 0 {
		t.Fatalf("expected counters rolled back to 0 rows, got %d", n)
	}

	var attempts int64
	if err := postgres.DB.Raw("SELECT COUNT(*) FROM verification_attempts").Scan(&attempts).Error; err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if attempts != 0 {
		t.Fatalf("expected attempts rolled back to 0 rows, got %d", attempts)
	}
}
