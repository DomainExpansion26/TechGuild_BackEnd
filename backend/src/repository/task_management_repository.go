package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
)

// ErrQuestNotFullySettled means at least one milestone is unreleased or its money
// has not actually moved. The service layer maps it onto its own public sentinel.
var ErrQuestNotFullySettled = errors.New("quest has milestones that are not fully settled")

type TaskManagementRepository interface {
	GetClientActiveQuests(clientID uuid.UUID) ([]models.Quest, error)
	GetQuestByIDAndClient(questID, clientID uuid.UUID) (*models.Quest, error)
	GetMilestoneByID(milestoneID uuid.UUID) (*models.QuestMilestone, error)
	GetMilestoneWithQuest(milestoneID uuid.UUID) (*models.QuestMilestone, *models.Quest, error)
	CreateMilestone(quest *models.Quest, milestone *models.QuestMilestone) error
	DeleteMilestone(milestoneID uuid.UUID, questID uuid.UUID) error
	UpdateMilestoneStatus(milestoneID uuid.UUID, status models.MilestoneStatus, approvedAt *time.Time) error
	LogMilestoneApproval(log *models.MilestoneApprovalLog) error
	UpdateQuestStatus(questID uuid.UUID, status models.QuestStatus) error
	CreateQuestReview(review *models.QuestReview) error
	CreateActivity(activity *models.QuestActivity) error
	GetActivitiesByQuestID(questID uuid.UUID, limit, offset int) ([]models.QuestActivity, int64, error)
	GetRecentActivities(questID uuid.UUID, limit int) ([]models.QuestActivity, error)
	GetMetrics(clientID uuid.UUID, questID *uuid.UUID) (dto.TaskManagementMetricsResponse, error)
	CreateTask(task *models.Task) error
	GetTaskByID(taskID uuid.UUID) (*models.Task, error)
	GetTaskByIDWithQuest(taskID uuid.UUID) (*models.Task, error)
	GetTasksByQuest(questID uuid.UUID) ([]models.Task, error)
	UpdateTaskStatus(taskID uuid.UUID, status models.TaskStatus) error
	GetQuestByID(questID uuid.UUID) (*models.Quest, error)
	VerifyQuestFullySettled(questID uuid.UUID) error
}

type taskManagementRepository struct {
	db *gorm.DB
}

func NewTaskManagementRepository() TaskManagementRepository {
	return &taskManagementRepository{db: postgres.DB}
}

func NewTaskManagementRepositoryWithDB(db *gorm.DB) TaskManagementRepository {
	return &taskManagementRepository{db: db}
}

func (r *taskManagementRepository) GetClientActiveQuests(clientID uuid.UUID) ([]models.Quest, error) {
	var quests []models.Quest
	err := r.db.
		Preload("Milestones.Tasks").
		Preload("AssignedFreelancer").
		Where("client_id = ? AND status IN (?, ?)", clientID, models.QuestStatusInProgress, models.QuestStatusCompleted).
		Order("updated_at DESC").
		Find(&quests).Error

	return quests, err
}

func (r *taskManagementRepository) GetQuestByIDAndClient(questID, clientID uuid.UUID) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Milestones.Tasks.Assignee").
		Preload("AssignedFreelancer").
		Where("id = ? AND client_id = ?", questID, clientID).
		First(&quest).Error

	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *taskManagementRepository) GetMilestoneByID(milestoneID uuid.UUID) (*models.QuestMilestone, error) {
	var milestone models.QuestMilestone
	err := r.db.
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Tasks.Assignee").
		Where("id = ?", milestoneID).
		First(&milestone).Error

	if err != nil {
		return nil, err
	}
	return &milestone, nil
}

func (r *taskManagementRepository) GetMilestoneWithQuest(milestoneID uuid.UUID) (*models.QuestMilestone, *models.Quest, error) {
	var milestone models.QuestMilestone
	if err := r.db.Where("id = ?", milestoneID).First(&milestone).Error; err != nil {
		return nil, nil, err
	}

	var quest models.Quest
	if err := r.db.Where("id = ?", milestone.QuestID).First(&quest).Error; err != nil {
		return &milestone, nil, err
	}

	return &milestone, &quest, nil
}

func (r *taskManagementRepository) CreateMilestone(quest *models.Quest, milestone *models.QuestMilestone) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(milestone).Error; err != nil {
			return err
		}
		// Quest.BudgetMinor is a CLIENT-SET reference amount.
		// Freelancer milestones DO NOT modify it.
		return nil
	})
}

func (r *taskManagementRepository) DeleteMilestone(milestoneID uuid.UUID, questID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete associated tasks
		if err := tx.Where("milestone_id = ?", milestoneID).Delete(&models.Task{}).Error; err != nil {
			return err
		}
		// 2. Delete milestone
		if err := tx.Where("id = ?", milestoneID).Delete(&models.QuestMilestone{}).Error; err != nil {
			return err
		}
		// Quest.BudgetMinor stays unchanged — client's reference amount.
		return nil
	})
}

func (r *taskManagementRepository) UpdateMilestoneStatus(milestoneID uuid.UUID, status models.MilestoneStatus, approvedAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if approvedAt != nil {
		updates["approved_at"] = approvedAt
	} else {
		updates["approved_at"] = nil
	}
	return r.db.Model(&models.QuestMilestone{}).Where("id = ?", milestoneID).Updates(updates).Error
}

func (r *taskManagementRepository) LogMilestoneApproval(log *models.MilestoneApprovalLog) error {
	return r.db.Create(log).Error
}

func (r *taskManagementRepository) UpdateQuestStatus(questID uuid.UUID, status models.QuestStatus) error {
	return r.db.Model(&models.Quest{}).Where("id = ?", questID).Update("status", status).Error
}

func (r *taskManagementRepository) CreateQuestReview(review *models.QuestReview) error {
	return r.db.Create(review).Error
}

func (r *taskManagementRepository) CreateActivity(activity *models.QuestActivity) error {
	return r.db.Create(activity).Error
}

func (r *taskManagementRepository) GetActivitiesByQuestID(questID uuid.UUID, limit, offset int) ([]models.QuestActivity, int64, error) {
	var activities []models.QuestActivity
	var total int64

	db := r.db.Model(&models.QuestActivity{}).Where("quest_id = ?", questID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Actor").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&activities).Error

	return activities, total, err
}

func (r *taskManagementRepository) GetRecentActivities(questID uuid.UUID, limit int) ([]models.QuestActivity, error) {
	var activities []models.QuestActivity
	err := r.db.
		Where("quest_id = ?", questID).
		Order("created_at DESC").
		Limit(limit).
		Find(&activities).Error
	return activities, err
}

func (r *taskManagementRepository) GetMetrics(clientID uuid.UUID, questID *uuid.UUID) (dto.TaskManagementMetricsResponse, error) {
	var metrics dto.TaskManagementMetricsResponse

	// 1. Active Quests
	var activeQuestsCount int64
	if err := r.db.Model(&models.Quest{}).
		Where("client_id = ? AND status = ?", clientID, models.QuestStatusInProgress).
		Count(&activeQuestsCount).Error; err != nil {
		return metrics, err
	}
	metrics.ActiveQuests = int(activeQuestsCount)

	// 2. Completed Quests
	var completedQuestsCount int64
	if err := r.db.Model(&models.Quest{}).
		Where("client_id = ? AND status = ?", clientID, models.QuestStatusCompleted).
		Count(&completedQuestsCount).Error; err != nil {
		return metrics, err
	}
	metrics.CompletedQuests = int(completedQuestsCount)

	milestoneQuery := r.db.Model(&models.QuestMilestone{}).
		Joins("JOIN quests ON quests.id = quest_milestones.quest_id").
		Where("quests.client_id = ? AND quests.status IN (?, ?)", clientID, models.QuestStatusInProgress, models.QuestStatusCompleted)

	taskQuery := r.db.Model(&models.Task{}).
		Joins("JOIN quests ON quests.id = tasks.quest_id").
		Where("quests.client_id = ? AND quests.status IN (?, ?)", clientID, models.QuestStatusInProgress, models.QuestStatusCompleted)

	if questID != nil && *questID != uuid.Nil {
		milestoneQuery = milestoneQuery.Where("quests.id = ?", *questID)
		taskQuery = taskQuery.Where("quests.id = ?", *questID)
	}

	// 3. Pending Reviews (submitted for release milestones)
	var pendingReviewsCount int64
	if err := milestoneQuery.Session(&gorm.Session{}).
		Where("quest_milestones.status = ?", models.MilestoneStatusSubmittedForRelease).
		Count(&pendingReviewsCount).Error; err != nil {
		return metrics, err
	}
	metrics.PendingReviews = int(pendingReviewsCount)

	// 4. Tasks In Progress
	var tasksInProgressCount int64
	if err := taskQuery.Session(&gorm.Session{}).
		Where("tasks.status = ?", models.TaskStatusInProgress).
		Count(&tasksInProgressCount).Error; err != nil {
		return metrics, err
	}
	metrics.TasksInProgress = int(tasksInProgressCount)

	// 5. Upcoming Deadlines
	var upcomingDeadlinesCount int64
	now := time.Now()
	if err := milestoneQuery.Session(&gorm.Session{}).
		Where("quest_milestones.due_date >= ? AND quest_milestones.status != ?", now, models.MilestoneStatusReleased).
		Count(&upcomingDeadlinesCount).Error; err != nil {
		return metrics, err
	}
	metrics.UpcomingDeadlines = int(upcomingDeadlinesCount)

	return metrics, nil
}

func (r *taskManagementRepository) CreateTask(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *taskManagementRepository) GetTaskByID(taskID uuid.UUID) (*models.Task, error) {
	var task models.Task
	err := r.db.Preload("Assignee").Preload("Milestone").Where("id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskManagementRepository) UpdateTaskStatus(taskID uuid.UUID, status models.TaskStatus) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *taskManagementRepository) GetQuestByID(questID uuid.UUID) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Where("id = ?", questID).
		First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

// VerifyQuestFullySettled fails unless every milestone of the quest is released AND
// its money has actually moved (MilestonePayment released + EscrowTransaction released).
// COALESCE makes a missing payment/escrow row count as unsettled instead of NULL-comparing
// to false. Kept as one query here so the TMD service needs no money-repository wiring.
func (r *taskManagementRepository) VerifyQuestFullySettled(questID uuid.UUID) error {
	var unsettled int64
	err := r.db.Model(&models.QuestMilestone{}).
		Joins("LEFT JOIN milestone_payments mp ON mp.quest_milestone_id = quest_milestones.id").
		Joins("LEFT JOIN escrow_transactions e ON e.quest_milestone_id = quest_milestones.id").
		Where("quest_milestones.quest_id = ?", questID).
		Where("quest_milestones.status <> ? OR COALESCE(mp.status, '') <> ? OR COALESCE(e.status, '') <> ?",
			models.MilestoneStatusReleased,
			models.MilestonePaymentReleased,
			models.EscrowStatusReleased,
		).
		Count(&unsettled).Error
	if err != nil {
		return err
	}
	if unsettled > 0 {
		return ErrQuestNotFullySettled
	}
	return nil
}

// GetTaskByIDWithQuest loads a task together with its milestone and quest so
// callers can authorize against quest ownership and the accepted party.
func (r *taskManagementRepository) GetTaskByIDWithQuest(taskID uuid.UUID) (*models.Task, error) {
	var task models.Task
	err := r.db.
		Preload("Assignee").
		Preload("Milestone").
		Preload("Milestone.Quest").
		Where("id = ?", taskID).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTasksByQuest returns every task of a quest in one query (dashboard grouping).
func (r *taskManagementRepository) GetTasksByQuest(questID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("quest_id = ?", questID).
		Preload("Assignee").
		Order("created_at ASC").
		Find(&tasks).Error
	return tasks, err
}
