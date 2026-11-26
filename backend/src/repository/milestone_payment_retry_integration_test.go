//go:build integration

package repository

import (
	"os"
	"testing"

	"github.com/google/uuid"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"techguild-backend/src/models"
)

// TestMilestonePaymentRetryIndex_Postgres proves the retry contract that the
// partial unique index exists for. SQLite cannot prove it: the whole point is
// PostgreSQL partial-index semantics, so this needs a real server.
//
// Requires POSTGRES_TEST_DSN; skips if empty. Not run by `go test ./...`.
func TestMilestonePaymentRetryIndex_Postgres(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DSN to run this integration test")
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	if err := db.Exec(`DROP TABLE IF EXISTS milestone_payments`).Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if err := db.AutoMigrate(&models.MilestonePayment{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	// Mirror migrateMilestonePaymentConstraints verbatim: the model's plain
	// index is dropped, then the live-only unique index replaces it.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_milestone_payments_quest_milestone_id;`).Error; err != nil {
		t.Fatalf("drop legacy unique index: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_mp_live_per_milestone
		ON milestone_payments (quest_milestone_id)
		WHERE status NOT IN ('failed','refunded');`).Error; err != nil {
		t.Fatalf("create partial unique index: %v", err)
	}

	msID, questID := uuid.New(), uuid.New()
	attempt := func(status models.MilestonePaymentStatus) *models.MilestonePayment {
		return &models.MilestonePayment{
			QuestMilestoneID: msID,
			QuestID:          questID,
			PaymentID:        uuid.New(),
			Status:           status,
		}
	}

	// A failed attempt is history and must be storable on its own.
	if err := db.Create(attempt(models.MilestonePaymentFailed)).Error; err != nil {
		t.Fatalf("failed attempt must be storable: %v", err)
	}

	// The retry the old full unique index made impossible.
	if err := db.Create(attempt(models.MilestonePaymentPending)).Error; err != nil {
		t.Fatalf("retry after failure must be allowed: %v", err)
	}

	// Only ONE live attempt may exist per milestone.
	if err := db.Create(attempt(models.MilestonePaymentFunded)).Error; err == nil {
		t.Fatal("two live attempts for one milestone must be rejected by uniq_mp_live_per_milestone")
	}

	// The lookup used by FundMilestone must see the live row, not the failure.
	live, err := NewMilestonePaymentRepositoryTx(db).GetByQuestMilestoneID(msID)
	if err != nil {
		t.Fatalf("GetByQuestMilestoneID: %v", err)
	}
	if live.Status != models.MilestonePaymentPending {
		t.Fatalf("live lookup returned %q, want %q", live.Status, models.MilestonePaymentPending)
	}

	// Refunded attempts are terminal history too, so a refund does not block
	// a later funding attempt on the same milestone.
	other := uuid.New()
	mk := func(status models.MilestonePaymentStatus, ms uuid.UUID) *models.MilestonePayment {
		return &models.MilestonePayment{QuestMilestoneID: ms, QuestID: questID, PaymentID: uuid.New(), Status: status}
	}
	if err := db.Create(mk(models.MilestonePaymentRefunded, other)).Error; err != nil {
		t.Fatalf("refunded attempt must be storable: %v", err)
	}
	if err := db.Create(mk(models.MilestonePaymentPending, other)).Error; err != nil {
		t.Fatalf("refunded must not block a new attempt: %v", err)
	}
}
