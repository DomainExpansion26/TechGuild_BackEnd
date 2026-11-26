package repository

import (
	"errors"
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMilestoneNotFound = errors.New("milestone not found")
var ErrRevisionNotFound = errors.New("milestone revision not found")
var ErrTaskNotFound = errors.New("task not found")
var ErrMilestoneBudgetExceeded = errors.New("milestone amount exceeds remaining quest budget")

// lockQuestAndCheckBudget locks the parent quest row and rejects the write when
// adding newAmountMinor to the quest's other milestones would exceed the quest
// budget. excludeID is uuid.Nil on create and the milestone's own id on update,
// so an update is measured against the quest's other milestones only.
func lockQuestAndCheckBudget(tx *gorm.DB, questID uuid.UUID, newAmountMinor int64, excludeID uuid.UUID) error {
	query := tx.Where("id = ?", questID)
	if tx.Dialector.Name() != "sqlite" {
		// ponytail: sqlite has no SELECT ... FOR UPDATE and serializes writes
		// anyway, so tests skip the clause; Postgres gets the real row lock.
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var quest models.Quest
	if err := query.First(&quest).Error; err != nil {
		return err
	}

	sum, err := sumOtherMilestones(tx, questID, excludeID)
	if err != nil {
		return err
	}
	if sum+newAmountMinor > quest.BudgetMinor {
		return ErrMilestoneBudgetExceeded
	}
	return nil
}

func sumOtherMilestones(tx *gorm.DB, questID uuid.UUID, excludeID uuid.UUID) (int64, error) {
	query := tx.Model(&models.QuestMilestone{}).Where("quest_id = ?", questID)
	if excludeID != uuid.Nil {
		query = query.Where("id <> ?", excludeID)
	}
	var sum int64
	err := query.Select("COALESCE(SUM(amount_minor), 0)").Scan(&sum).Error
	return sum, err
}

// TaskPatch is a partial update for an existing milestone task. Nil fields are
// omitted from the UPDATE so a partial PATCH never zeroes stored values.
type TaskPatch struct {
	ID          uuid.UUID
	Title       *string
	Description *string
	StartDate   *time.Time
	EndDate     *time.Time
	AssigneeID  *uuid.UUID
	Status      *models.TaskStatus
}

// TaskPatchSet is the additive task modification applied with a milestone update.
type TaskPatchSet struct {
	Create    []models.Task
	Update    []TaskPatch
	DeleteIDs []uuid.UUID
}

// toUpdates builds the UPDATE map from non-nil patch fields only.
func (p TaskPatch) toUpdates() map[string]any {
	updates := make(map[string]any, 6)
	if p.Title != nil {
		updates["title"] = *p.Title
	}
	if p.Description != nil {
		updates["description"] = *p.Description
	}
	if p.StartDate != nil {
		updates["start_date"] = *p.StartDate
	}
	if p.EndDate != nil {
		updates["end_date"] = *p.EndDate
	}
	if p.AssigneeID != nil {
		updates["assignee_id"] = *p.AssigneeID
	}
	if p.Status != nil {
		updates["status"] = *p.Status
	}
	return updates
}

type QuestMilestoneRepository interface {
	// Create
	Create(m *models.QuestMilestone) error
	CreateWithTasks(m *models.QuestMilestone, tasks []models.Task) error
	// UpdateWithTasks saves milestone fields and applies an additive task patch
	// (creates, partial updates, deletes) in a single transaction.
	UpdateWithTasks(m *models.QuestMilestone, set TaskPatchSet) error

	// Read
	FindByID(id uuid.UUID) (*models.QuestMilestone, error)
	FindByIDForUpdate(id uuid.UUID) (*models.QuestMilestone, error)
	FindByIDWithRevisions(id uuid.UUID) (*models.QuestMilestone, error)
	FindByQuestID(questID uuid.UUID) ([]models.QuestMilestone, error)
	CountByQuestID(questID uuid.UUID) (int64, error)
	GetMaxOrderIndex(questID uuid.UUID) (int, error)

	// Update
	Update(m *models.QuestMilestone) error
	UpdateStatus(id uuid.UUID, status models.MilestoneStatus) error
	UpdateStatusWithTimestamp(id uuid.UUID, status models.MilestoneStatus, timestampField string, ts time.Time) error

	// Delete
	Delete(id uuid.UUID) error

	// Revision
	CreateRevision(r *models.MilestoneRevision) error
	GetRevisionsByMilestone(milestoneID uuid.UUID) ([]models.MilestoneRevision, error)
	GetMaxRevisionRound(milestoneID uuid.UUID) (int, error)

	// Tx support
	WithTransaction(fn func(txRepo QuestMilestoneRepository) error) error

	BulkUpdateMilestoneStatus(questID uuid.UUID, fromStatus models.MilestoneStatus, toStatus models.MilestoneStatus, timestampField string, ts *time.Time) (int64, error)
}

type questMilestoneRepository struct {
	db *gorm.DB
}

func NewQuestMilestoneRepository() QuestMilestoneRepository {
	return &questMilestoneRepository{db: postgres.DB}
}

func NewQuestMilestoneRepositoryTx(tx *gorm.DB) QuestMilestoneRepository {
	return &questMilestoneRepository{db: tx}
}

func (r *questMilestoneRepository) WithTransaction(fn func(txRepo QuestMilestoneRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewQuestMilestoneRepositoryTx(tx))
	})
}

// ============================================================
// CREATE
// ============================================================

func (r *questMilestoneRepository) Create(m *models.QuestMilestone) error {
	return r.db.Create(m).Error
}

// CreateWithTasks creates the milestone and its tasks atomically.
func (r *questMilestoneRepository) CreateWithTasks(
	m *models.QuestMilestone,
	tasks []models.Task,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockQuestAndCheckBudget(tx, m.QuestID, m.AmountMinor, uuid.Nil); err != nil {
			return err
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		if len(tasks) > 0 {
			for i := range tasks {
				tasks[i].MilestoneID = m.ID
				tasks[i].QuestID = m.QuestID
			}
			if err := tx.Create(&tasks).Error; err != nil {
				return err
			}
			m.Tasks = tasks
		}
		return nil
	})
}

// ============================================================
// READ
// ============================================================

func (r *questMilestoneRepository) FindByID(id uuid.UUID) (*models.QuestMilestone, error) {
	var m models.QuestMilestone
	err := r.db.
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Tasks.Assignee").
		Where("id = ?", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &m, nil
}

// FindByIDForUpdate is a bare locked read (SELECT ... FOR UPDATE) for use inside a
// caller-owned transaction; callers re-verify state after acquiring the lock.
func (r *questMilestoneRepository) FindByIDForUpdate(id uuid.UUID) (*models.QuestMilestone, error) {
	var m models.QuestMilestone
	err := r.db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *questMilestoneRepository) FindByIDWithRevisions(id uuid.UUID) (*models.QuestMilestone, error) {
	var m models.QuestMilestone
	err := r.db.
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Tasks.Assignee").
		Where("id = ?", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *questMilestoneRepository) FindByQuestID(questID uuid.UUID) ([]models.QuestMilestone, error) {
	var milestones []models.QuestMilestone
	err := r.db.
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Tasks.Assignee").
		Where("quest_id = ?", questID).
		Order("order_index ASC").
		Find(&milestones).Error
	return milestones, err
}

func (r *questMilestoneRepository) CountByQuestID(questID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.QuestMilestone{}).
		Where("quest_id = ?", questID).
		Count(&count).Error
	return count, err
}

func (r *questMilestoneRepository) GetMaxOrderIndex(questID uuid.UUID) (int, error) {
	var maxOrder *int
	err := r.db.Model(&models.QuestMilestone{}).
		Where("quest_id = ?", questID).
		Select("MAX(order_index)").
		Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder, nil
}

// ============================================================
// UPDATE
// ============================================================

func (r *questMilestoneRepository) Update(m *models.QuestMilestone) error {
	return r.db.Omit(clause.Associations).Save(m).Error
}

// UpdateWithTasks saves the milestone and applies the task patch atomically.
// Every task mutation is scoped to the milestone id, so a task id belonging to
// another milestone is reported as not found instead of being written.
func (r *questMilestoneRepository) UpdateWithTasks(m *models.QuestMilestone, set TaskPatchSet) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockQuestAndCheckBudget(tx, m.QuestID, m.AmountMinor, m.ID); err != nil {
			return err
		}
		if err := tx.Omit(clause.Associations).Save(m).Error; err != nil {
			return err
		}

		for i := range set.Create {
			t := set.Create[i]
			t.MilestoneID = m.ID
			t.QuestID = m.QuestID
			if err := tx.Create(&t).Error; err != nil {
				return err
			}
		}

		for _, patch := range set.Update {
			updates := patch.toUpdates()
			if len(updates) == 0 {
				continue // no fields supplied: nothing to change
			}
			res := tx.Model(&models.Task{}).
				Where("id = ? AND milestone_id = ?", patch.ID, m.ID).
				Updates(updates)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				// Postgres reports 0 rows for a no-op write, so distinguish
				// "missing task" from "same values" with an existence check.
				var count int64
				if err := tx.Model(&models.Task{}).
					Where("id = ? AND milestone_id = ?", patch.ID, m.ID).
					Count(&count).Error; err != nil {
					return err
				}
				if count == 0 {
					return ErrTaskNotFound
				}
			}
		}

		if len(set.DeleteIDs) > 0 {
			res := tx.Where("milestone_id = ? AND id IN ?", m.ID, set.DeleteIDs).
				Delete(&models.Task{})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != int64(len(set.DeleteIDs)) {
				return ErrTaskNotFound
			}
		}

		return nil
	})
}

func (r *questMilestoneRepository) UpdateStatus(id uuid.UUID, status models.MilestoneStatus) error {
	res := r.db.
		Model(&models.QuestMilestone{}).
		Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}

// UpdateStatusWithTimestamp updates status AND sets a specific timestamp field.
// Allowed timestamp fields: "agreed_at", "funded_at", "released_at", "completed_at", "approved_at".
func (r *questMilestoneRepository) UpdateStatusWithTimestamp(
	id uuid.UUID,
	status models.MilestoneStatus,
	timestampField string,
	ts time.Time,
) error {
	// Whitelist to prevent SQL injection via field name
	allowedFields := map[string]bool{
		"agreed_at":    true,
		"funded_at":    true,
		"released_at":  true,
		"completed_at": true,
		"approved_at":  true,
	}
	if !allowedFields[timestampField] {
		return errors.New("invalid timestamp field")
	}

	updates := map[string]interface{}{
		"status":       status,
		"updated_at":   time.Now(),
		timestampField: ts,
	}

	res := r.db.
		Model(&models.QuestMilestone{}).
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}

// ============================================================
// DELETE
// ============================================================

func (r *questMilestoneRepository) Delete(id uuid.UUID) error {
	res := r.db.
		Where("id = ?", id).
		Delete(&models.QuestMilestone{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}

// ============================================================
// REVISION
// ============================================================

func (r *questMilestoneRepository) CreateRevision(rev *models.MilestoneRevision) error {
	return r.db.Create(rev).Error
}

func (r *questMilestoneRepository) GetRevisionsByMilestone(milestoneID uuid.UUID) ([]models.MilestoneRevision, error) {
	var revs []models.MilestoneRevision
	err := r.db.
		Preload("RequestedBy").
		Where("milestone_id = ?", milestoneID).
		Order("round_number ASC, created_at ASC").
		Find(&revs).Error
	return revs, err
}

func (r *questMilestoneRepository) GetMaxRevisionRound(milestoneID uuid.UUID) (int, error) {
	var maxRound *int
	err := r.db.Model(&models.MilestoneRevision{}).
		Where("milestone_id = ?", milestoneID).
		Select("MAX(round_number)").
		Scan(&maxRound).Error
	if err != nil {
		return 0, err
	}
	if maxRound == nil {
		return 0, nil
	}
	return *maxRound, nil
}

func (r *questMilestoneRepository) BulkUpdateMilestoneStatus(questID uuid.UUID, fromStatus models.MilestoneStatus, toStatus models.MilestoneStatus, timestampField string, ts *time.Time) (int64, error) {
	updates := map[string]interface{}{
		"status":     toStatus,
		"updated_at": time.Now(),
	}
	if timestampField != "" && ts != nil {
		// Whitelist to prevent SQL injection via field name
		allowedFields := map[string]bool{
			"agreed_at":    true,
			"funded_at":    true,
			"released_at":  true,
			"completed_at": true,
			"approved_at":  true,
		}
		if !allowedFields[timestampField] {
			return 0, errors.New("invalid timestamp field")
		}
		updates[timestampField] = *ts
	}

	result := r.db.Model(&models.QuestMilestone{}).
		Where("quest_id = ? AND status = ?", questID, fromStatus).
		Updates(updates)

	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
