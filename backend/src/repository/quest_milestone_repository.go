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

type QuestMilestoneRepository interface {
	// Create
	Create(m *models.QuestMilestone) error
	CreateWithTasks(m *models.QuestMilestone, tasks []models.Task) error

	// Read
	FindByID(id uuid.UUID) (*models.QuestMilestone, error)
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
