package services

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
	"techguild-backend/src/repository"
)

// setupEscalationTestDB gives the regression test a real (in-memory) database
// so the escalator's query and update are exercised end-to-end.
func setupEscalationTestDB(t *testing.T) *gorm.DB {
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
		&models.User{},
		&models.VerificationRecord{},
		&models.VerificationDocument{},
		&models.VerificationAuditLog{},
	); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })

	return db
}

// Regression: a 25h-old auto_eligible record must never be auto-approved. The
// stale-pending escalator may only move it to manual review while leaving the
// record pending.
func TestProcessStalePendingEscalations_25HourRecordBecomesManualReview(t *testing.T) {
	db := setupEscalationTestDB(t)

	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     fmt.Sprintf("escalation-%s@example.com", uuid.NewString()),
		Status:    models.StatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}

	mlCompletedAt := time.Now().Add(-25 * time.Hour)
	record := models.VerificationRecord{
		UserID:         user.ID,
		Type:           models.VerificationIndividual,
		Status:         models.VerificationPending,
		ReviewPriority: models.ReviewPriorityAutoEligible,
		MLStatus:       models.MLStatusCompleted,
		MLCompletedAt:  &mlCompletedAt,
	}
	if err := db.Omit(clause.Associations).Create(&record).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}

	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	svc.ProcessStalePendingEscalations()

	var got models.VerificationRecord
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload verification record: %v", err)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (must never be auto-approved)", got.Status, models.VerificationPending)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.AutoApprovedAt != nil {
		t.Errorf("auto_approved_at = %v, want nil", got.AutoApprovedAt)
	}
}

// Phase 3 closes Gap 1: a queued record submitted more than 24h ago is now
// caught by the dedicated stuck-in-ML pass. It must be failed into the manual
// review queue with the terminal system_escalated_at marker — never approved,
// and never left stuck without a human.
func TestProcessStalePendingEscalations_StuckQueuedRecordEscalatesToMLFailed(t *testing.T) {
	db := setupEscalationTestDB(t)

	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     fmt.Sprintf("queued-escalation-%s@example.com", uuid.NewString()),
		Status:    models.StatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}

	submittedAt := time.Now().Add(-30 * time.Hour)
	record := models.VerificationRecord{
		UserID:         user.ID,
		Type:           models.VerificationIndividual,
		Status:         models.VerificationPending,
		ReviewPriority: models.ReviewPriorityManual, // model default; deliberately not auto_eligible
		MLStatus:       models.MLStatusQueued,
		MLCompletedAt:  nil,
		SubmittedAt:    &submittedAt,
	}
	if err := db.Omit(clause.Associations).Create(&record).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}

	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	svc.ProcessStalePendingEscalations()

	var got models.VerificationRecord
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload verification record: %v", err)
	}

	if got.MLStatus != models.MLStatusFailed {
		t.Errorf("ml_status = %q, want %q (stuck record must be failed, not left queued)", got.MLStatus, models.MLStatusFailed)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.SystemEscalatedAt == nil {
		t.Error("system_escalated_at = nil, want set (terminal escalation marker)")
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q", got.RoutingState, models.RoutingRouted)
	}
	if got.DecisionSource != "system_stale" {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, "system_stale")
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (escalation never approves/rejects)", got.Status, models.VerificationPending)
	}

	var audit models.VerificationAuditLog
	if err := db.Where("verification_id = ? AND action_type = ?", record.ID, "SYSTEM_ESCALATED_STALE_24H").
		First(&audit).Error; err != nil {
		t.Fatalf("audit log with SYSTEM_ESCALATED_STALE_24H not written: %v", err)
	}
}
