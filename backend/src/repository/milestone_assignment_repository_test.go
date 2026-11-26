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

func setupMilestoneAssignmentTestDB(t *testing.T) {
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

	if err := db.AutoMigrate(&models.MilestoneAssignment{}); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })
}

// Re-assignment replaces rows instead of appending, so the unique
// (milestone_id, member_id) index can never be violated.
func TestReplaceForMilestoneIsIdempotent(t *testing.T) {
	setupMilestoneAssignmentTestDB(t)
	repo := NewMilestoneAssignmentRepository()

	milestoneID := uuid.New()
	leadID := uuid.New()
	memberA := uuid.New()
	memberB := uuid.New()

	row := func(memberID uuid.UUID) models.MilestoneAssignment {
		return models.MilestoneAssignment{
			MilestoneID:  milestoneID,
			MemberID:     memberID,
			AssignedByID: leadID,
		}
	}

	if err := repo.ReplaceForMilestone(milestoneID, []models.MilestoneAssignment{row(memberA), row(memberB)}); err != nil {
		t.Fatalf("first assignment: %v", err)
	}
	rows, err := repo.ListByMilestone(milestoneID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("assignments = %d, want 2", len(rows))
	}

	if err := repo.ReplaceForMilestone(milestoneID, []models.MilestoneAssignment{row(memberA)}); err != nil {
		t.Fatalf("re-assignment: %v", err)
	}
	rows, err = repo.ListByMilestone(milestoneID)
	if err != nil {
		t.Fatalf("list after replace: %v", err)
	}
	if len(rows) != 1 || rows[0].MemberID != memberA {
		t.Fatalf("assignments after replace = %d, want 1 (%s)", len(rows), memberA)
	}

	assigned, err := repo.ExistsForMilestoneAndMember(milestoneID, memberA)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !assigned {
		t.Fatal("memberA should be assigned")
	}
	assigned, err = repo.ExistsForMilestoneAndMember(milestoneID, memberB)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if assigned {
		t.Fatal("memberB should no longer be assigned")
	}
}
