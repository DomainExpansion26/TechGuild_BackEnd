package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type QuestFilter struct {
	AllowedRanks []string
	ClientID     *uuid.UUID
	Status       string
	StatusIn     []string
	Category     string
	Complexity   string
	MinBudget    *float64
	MaxBudget    *float64
	Cursor       string
	Limit        int
}

type QuestRepository interface {
	CreateQuestWithMilestonesAndSkills(quest *models.Quest, milestones []models.QuestMilestone, skills []models.QuestSkill) error
	FindByID(questID uuid.UUID) (*models.Quest, error)
	UpdateQuest(quest *models.Quest) error
	ListQuests(filter QuestFilter) ([]models.Quest, string, bool, int64, error)
	CountPartyActiveQuests(partyID uuid.UUID) (int64, error)
	HasActiveQuestWithClient(partyID uuid.UUID, clientID uuid.UUID) (bool, error)
	AcceptQuestAtomic(quest *models.Quest, chatChannel *models.QuestChatChannel, tasks []models.Task, activity *models.QuestActivity, auditLog *models.AuditLog) error
	CreateAuditLog(auditLog *models.AuditLog) error
	GetClientProfile(userID uuid.UUID) (*models.ClientProfile, error)
}

type questRepository struct {
	db *gorm.DB
}

func NewQuestRepository() QuestRepository {
	return &questRepository{db: postgres.DB}
}

func NewQuestRepositoryTx(tx *gorm.DB) QuestRepository {
	return &questRepository{db: tx}
}

// CreateQuestWithMilestonesAndSkills saves the quest, milestones, and skills atomically in a single transaction.
func (r *questRepository) CreateQuestWithMilestonesAndSkills(
	quest *models.Quest,
	milestones []models.QuestMilestone,
	skills []models.QuestSkill,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quest).Error; err != nil {
			return err
		}

		if len(milestones) > 0 {
			for i := range milestones {
				milestones[i].QuestID = quest.ID
			}
			if err := tx.Create(&milestones).Error; err != nil {
				return err
			}
			quest.Milestones = milestones
		}

		if len(skills) > 0 {
			for i := range skills {
				skills[i].QuestID = quest.ID
			}
			if err := tx.Create(&skills).Error; err != nil {
				return err
			}
			quest.Skills = skills
		}

		return nil
	})
}

// FindByID retrieves a quest by its UUID, preloading Client, Milestones, Skills, and AcceptedParty.
func (r *questRepository) FindByID(questID uuid.UUID) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.
		Preload("Client").
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Skills").
		Preload("AcceptedParty").
		First(&quest, "id = ?", questID).Error

	if err != nil {
		return nil, err
	}
	return &quest, nil
}

// UpdateQuest saves updates to an existing quest.
func (r *questRepository) UpdateQuest(quest *models.Quest) error {
	return r.db.Save(quest).Error
}

// ListQuests queries quests based on asymmetric tier visibility, client filter, and cursor pagination.
func (r *questRepository) ListQuests(filter QuestFilter) ([]models.Quest, string, bool, int64, error) {
	query := r.db.Model(&models.Quest{})

	if filter.ClientID != nil {
		query = query.Where("client_id = ?", *filter.ClientID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else if len(filter.StatusIn) > 0 {
		query = query.Where("status IN (?)", filter.StatusIn)
	}

	// Asymmetric tier visibility: filter by allowed ranks
	if len(filter.AllowedRanks) > 0 {
		query = query.Where("required_min_rank IN (?)", filter.AllowedRanks)
	}

	if filter.Category != "" {
		query = query.Where("category ILIKE ?", "%"+filter.Category+"%")
	}

	if filter.Complexity != "" {
		query = query.Where("complexity = ?", filter.Complexity)
	}

	if filter.MinBudget != nil {
		query = query.Where("budget_inr >= ?", *filter.MinBudget)
	}

	if filter.MaxBudget != nil {
		query = query.Where("budget_inr <= ?", *filter.MaxBudget)
	}

	// Count total matching records before cursor pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, "", false, 0, err
	}

	// Apply cursor pagination: order by created_at DESC, id DESC
	if filter.Cursor != "" {
		if cursorTime, err := time.Parse(time.RFC3339Nano, filter.Cursor); err == nil {
			query = query.Where("created_at < ?", cursorTime)
		} else if cursorTime, err := time.Parse(time.RFC3339, filter.Cursor); err == nil {
			query = query.Where("created_at < ?", cursorTime)
		}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	var quests []models.Quest
	err := query.
		Preload("Client").
		Preload("Skills").
		Preload("Milestones").
		Order("created_at DESC, id DESC").
		Limit(limit + 1).
		Find(&quests).Error

	if err != nil {
		return nil, "", false, 0, err
	}

	hasMore := false
	nextCursor := ""

	if len(quests) > limit {
		hasMore = true
		quests = quests[:limit]
		nextCursor = quests[len(quests)-1].CreatedAt.Format(time.RFC3339Nano)
	} else if len(quests) > 0 {
		nextCursor = quests[len(quests)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	return quests, nextCursor, hasMore, total, nil
}

// CountPartyActiveQuests returns count of quests currently in_progress for a given party.
func (r *questRepository) CountPartyActiveQuests(partyID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Quest{}).
		Where("accepted_party_id = ? AND status = ?", partyID, models.QuestStatusInProgress).
		Count(&count).Error
	return count, err
}

// HasActiveQuestWithClient checks if party has any in_progress quest with the specified client.
func (r *questRepository) HasActiveQuestWithClient(partyID uuid.UUID, clientID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Quest{}).
		Where("accepted_party_id = ? AND client_id = ? AND status = ?", partyID, clientID, models.QuestStatusInProgress).
		Count(&count).Error
	return count > 0, err
}

// AcceptQuestAtomic atomically executes quest acceptance: status update, chat channel, task template, activity, and audit log.
func (r *questRepository) AcceptQuestAtomic(
	quest *models.Quest,
	chatChannel *models.QuestChatChannel,
	tasks []models.Task,
	activity *models.QuestActivity,
	auditLog *models.AuditLog,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update quest status and assigned entities
		if err := tx.Save(quest).Error; err != nil {
			return err
		}

		// 2. Auto-create Quest Chat Channel
		if chatChannel != nil {
			if err := tx.Create(chatChannel).Error; err != nil {
				return err
			}
		}

		// 3. Auto-create Task Template (initial tasks for milestones)
		if len(tasks) > 0 {
			if err := tx.Create(&tasks).Error; err != nil {
				return err
			}
		}

		// 4. Record Quest Activity
		if activity != nil {
			if err := tx.Create(activity).Error; err != nil {
				return err
			}
		}

		// 5. Audit Log
		if auditLog != nil {
			if err := tx.Create(auditLog).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// CreateAuditLog records an audit log entry.
func (r *questRepository) CreateAuditLog(auditLog *models.AuditLog) error {
	return r.db.Create(auditLog).Error
}

// GetClientProfile retrieves the client profile for a user ID.
func (r *questRepository) GetClientProfile(userID uuid.UUID) (*models.ClientProfile, error) {
	var profile models.ClientProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
