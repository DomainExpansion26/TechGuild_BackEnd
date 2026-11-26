package repository

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func setupQuestMilestoneBudgetTestDB(t *testing.T) *gorm.DB {
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

	if err := db.AutoMigrate(&models.Quest{}, &models.QuestMilestone{}, &models.Task{}); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })

	return db
}

func budgetTestQuest(t *testing.T, db *gorm.DB, budgetMinor int64) *models.Quest {
	t.Helper()

	q := &models.Quest{
		ID:          uuid.New(),
		ClientID:    uuid.New(),
		Title:       "Budget Quest",
		Description: "budget ceiling test",
		Category:    "Engineering",
		BudgetMinor: budgetMinor,
		Status:      models.QuestStatusOpen,
	}
	if err := db.Create(q).Error; err != nil {
		t.Fatalf("create quest: %v", err)
	}
	return q
}

func TestCreateWithTasksBudgetCeiling(t *testing.T) {
	db := setupQuestMilestoneBudgetTestDB(t)
	repo := NewQuestMilestoneRepositoryTx(db)

	t.Run("under budget", func(t *testing.T) {
		q := budgetTestQuest(t, db, 1000)
		if err := repo.CreateWithTasks(&models.QuestMilestone{QuestID: q.ID, Title: "m1", AmountMinor: 400}, nil); err != nil {
			t.Fatalf("under-budget create: %v", err)
		}
	})

	t.Run("exact budget", func(t *testing.T) {
		q := budgetTestQuest(t, db, 1000)
		if err := repo.CreateWithTasks(&models.QuestMilestone{QuestID: q.ID, Title: "m1", AmountMinor: 400}, nil); err != nil {
			t.Fatalf("first create: %v", err)
		}
		if err := repo.CreateWithTasks(&models.QuestMilestone{QuestID: q.ID, Title: "m2", AmountMinor: 600}, nil); err != nil {
			t.Fatalf("exact-budget create: %v", err)
		}
	})

	t.Run("over budget", func(t *testing.T) {
		q := budgetTestQuest(t, db, 1000)
		if err := repo.CreateWithTasks(&models.QuestMilestone{QuestID: q.ID, Title: "m1", AmountMinor: 900}, nil); err != nil {
			t.Fatalf("first create: %v", err)
		}
		err := repo.CreateWithTasks(&models.QuestMilestone{QuestID: q.ID, Title: "m2", AmountMinor: 200}, nil)
		if err != ErrMilestoneBudgetExceeded {
			t.Fatalf("over-budget create: err = %v, want %v", err, ErrMilestoneBudgetExceeded)
		}
	})
}

func TestUpdateWithTasksBudgetCeiling(t *testing.T) {
	db := setupQuestMilestoneBudgetTestDB(t)
	repo := NewQuestMilestoneRepositoryTx(db)

	q := budgetTestQuest(t, db, 1000)
	m1 := &models.QuestMilestone{QuestID: q.ID, Title: "m1", AmountMinor: 400}
	m2 := &models.QuestMilestone{QuestID: q.ID, Title: "m2", AmountMinor: 300}
	if err := repo.CreateWithTasks(m1, nil); err != nil {
		t.Fatalf("create m1: %v", err)
	}
	if err := repo.CreateWithTasks(m2, nil); err != nil {
		t.Fatalf("create m2: %v", err)
	}

	// 400 + 700 = 1100 > 1000: rejected, and m2 must be left untouched.
	m2.AmountMinor = 700
	if err := repo.UpdateWithTasks(m2, TaskPatchSet{}); err != ErrMilestoneBudgetExceeded {
		t.Fatalf("over-budget update: err = %v, want %v", err, ErrMilestoneBudgetExceeded)
	}
	reloaded, err := repo.FindByID(m2.ID)
	if err != nil {
		t.Fatalf("reload m2: %v", err)
	}
	if reloaded.AmountMinor != 300 {
		t.Fatalf("m2 amount = %d, want 300 (update must not persist)", reloaded.AmountMinor)
	}

	// 400 + 600 = 1000 exactly: allowed.
	m2.AmountMinor = 600
	if err := repo.UpdateWithTasks(m2, TaskPatchSet{}); err != nil {
		t.Fatalf("exact-budget update: %v", err)
	}
}
