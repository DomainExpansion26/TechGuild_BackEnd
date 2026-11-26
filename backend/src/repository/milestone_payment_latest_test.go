package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func setupMilestonePaymentTestDB(t *testing.T) {
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

	if err := db.AutoMigrate(&models.MilestonePayment{}); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })
}

// The live-only lookup skips failed attempts; the latest-attempt lookup must
// still return the newest row so the detail endpoint can explain a failure.
func TestGetLatestByQuestMilestoneID(t *testing.T) {
	setupMilestonePaymentTestDB(t)
	repo := NewMilestonePaymentRepository()

	msID, questID := uuid.New(), uuid.New()
	seed := func(status models.MilestonePaymentStatus, createdAt time.Time) *models.MilestonePayment {
		mp := &models.MilestonePayment{
			QuestMilestoneID: msID,
			QuestID:          questID,
			PaymentID:        uuid.New(),
			Status:           status,
			CreatedAt:        createdAt,
		}
		if err := postgres.DB.Create(mp).Error; err != nil {
			t.Fatalf("seed attempt: %v", err)
		}
		return mp
	}

	seed(models.MilestonePaymentFailed, time.Now().Add(-time.Hour))
	newest := seed(models.MilestonePaymentFailed, time.Now())

	if _, err := repo.GetByQuestMilestoneID(msID); err == nil {
		t.Fatal("GetByQuestMilestoneID must not return a failed attempt")
	}

	got, err := repo.GetLatestByQuestMilestoneID(msID)
	if err != nil {
		t.Fatalf("GetLatestByQuestMilestoneID: %v", err)
	}
	if got.ID != newest.ID {
		t.Fatalf("got attempt %s, want newest %s", got.ID, newest.ID)
	}

	if _, err := repo.GetLatestByQuestMilestoneID(uuid.New()); err == nil {
		t.Fatal("expected an error for a milestone with no attempts")
	}
}
