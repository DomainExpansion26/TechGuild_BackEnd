package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// setupVerificationTestDB spins up an isolated in-memory SQLite database and
// swaps it into the package-level postgres.DB handle used by the repository.
// Each test gets its own database (unique DSN) so tests can't see each other's
// rows.
func setupVerificationTestDB(t *testing.T) *gorm.DB {
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
	// A single connection keeps the :memory: database alive and consistent.
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.User{},
		&models.VerificationRecord{},
		&models.VerificationDocument{},
	); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     fmt.Sprintf("test-%s@example.com", uuid.NewString()),
		Status:    models.StatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}

// seedVerificationRecord inserts a record with the auto-eligible baseline and
// lets the caller override individual fields.
func seedVerificationRecord(
	t *testing.T,
	db *gorm.DB,
	userID uuid.UUID,
	mutate func(*models.VerificationRecord),
) models.VerificationRecord {
	t.Helper()

	completed := time.Now().Add(-25 * time.Hour)
	record := models.VerificationRecord{
		UserID:         userID,
		Type:           models.VerificationIndividual,
		Status:         models.VerificationPending,
		MLStatus:       models.MLStatusCompleted,
		ReviewPriority: models.ReviewPriorityAutoEligible,
		MLCompletedAt:  &completed,
	}
	if mutate != nil {
		mutate(&record)
	}
	if err := db.Omit(clause.Associations).Create(&record).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}
	return record
}

func hoursAgo(hours float64) *time.Time {
	value := time.Now().Add(-time.Duration(hours * float64(time.Hour)))
	return &value
}

func TestGetAutoEligibleQueue_FiltersAndOrders(t *testing.T) {
	db := setupVerificationTestDB(t)
	user := createTestUser(t, db)

	oldest := seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
	})
	newer := seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(26)
	})

	// Every record below must be excluded by exactly one guard.
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(23) // hold window not elapsed
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.AutoApprovedAt = hoursAgo(1)
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.BulkApprovedAt = hoursAgo(1)
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.RejectedAt = hoursAgo(1)
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.BulkReviewRequestedAt = hoursAgo(1)
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.ReviewPriority = models.ReviewPriorityManual
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.Status = models.VerificationApproved
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.MLStatus = models.MLStatusQueued
	})
	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = nil // ML never completed
	})

	repo := NewVerificationRecordsRepository()

	records, total, err := repo.GetAutoEligibleQueue(10, 0, 24)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue returned an error: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(records) != 2 {
		t.Fatalf("returned %d record(s), want 2", len(records))
	}
	if records[0].ID != oldest.ID {
		t.Errorf("records[0] = %s, want oldest record %s", records[0].ID, oldest.ID)
	}
	if records[1].ID != newer.ID {
		t.Errorf("records[1] = %s, want newer record %s", records[1].ID, newer.ID)
	}

	// Pagination must keep the total (unfiltered by limit) and slice correctly.
	page, total, err := repo.GetAutoEligibleQueue(1, 0, 24)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue(limit=1) returned an error: %v", err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != oldest.ID {
		t.Errorf("first page = (total=%d, len=%d), want (2, 1) starting at %s", total, len(page), oldest.ID)
	}

	page, total, err = repo.GetAutoEligibleQueue(1, 1, 24)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue(offset=1) returned an error: %v", err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != newer.ID {
		t.Errorf("second page = (total=%d, len=%d), want (2, 1) starting at %s", total, len(page), newer.ID)
	}
}

func TestGetAutoEligibleQueue_RespectsConfiguredAge(t *testing.T) {
	db := setupVerificationTestDB(t)
	user := createTestUser(t, db)

	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(5)
	})

	repo := NewVerificationRecordsRepository()

	records, total, err := repo.GetAutoEligibleQueue(10, 0, 24)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue returned an error: %v", err)
	}
	if total != 0 || len(records) != 0 {
		t.Fatalf("24h window returned %d record(s) for a 5h-old record, want 0", len(records))
	}

	records, total, err = repo.GetAutoEligibleQueue(10, 0, 1)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue returned an error: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("1h window returned %d record(s) for a 5h-old record, want 1", len(records))
	}
}

func TestBulkApprove_ValidatesInput(t *testing.T) {
	setupVerificationTestDB(t)
	repo := NewVerificationRecordsRepository()
	adminID := uuid.New()

	if err := repo.BulkApprove(nil, adminID); err == nil {
		t.Fatal("BulkApprove accepted an empty ID list")
	}

	if err := repo.BulkApprove([]string{"not-a-uuid"}, adminID); err == nil {
		t.Fatal("BulkApprove accepted an invalid verification UUID")
	}

	valid := uuid.NewString()
	if err := repo.BulkApprove([]string{valid, "still-not-a-uuid"}, adminID); err == nil {
		t.Fatal("BulkApprove accepted a list containing an invalid verification UUID")
	}
}

func TestGetAutoEligibleQueue_ExcludesRecordsPulledForReview(t *testing.T) {
	db := setupVerificationTestDB(t)
	user := createTestUser(t, db)

	seedVerificationRecord(t, db, user.ID, func(r *models.VerificationRecord) {
		r.MLCompletedAt = hoursAgo(30)
		r.BulkReviewRequestedAt = hoursAgo(1)
	})

	repo := NewVerificationRecordsRepository()

	records, total, err := repo.GetAutoEligibleQueue(10, 0, 24)
	if err != nil {
		t.Fatalf("GetAutoEligibleQueue returned an error: %v", err)
	}
	if total != 0 || len(records) != 0 {
		t.Fatalf("record with bulk_review_requested_at was returned from the auto-eligible queue")
	}
}
