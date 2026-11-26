package repository

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"
)

type QuestFilter struct {
	AllowedRanks []string
	ClientID     *uuid.UUID
	Status       string
	StatusIn     []string
	Search       string
	Category     string
	Complexity   string
	QuestType    string
	Skills       []string
	TimelineMin  *int
	TimelineMax  *int
	ClientRank   string
	MinBudget    *float64
	MaxBudget    *float64
	Cursor       string
	Sort         string
	Limit        int
}

type QuestRepository interface {
	CreateQuestWithSkills(quest *models.Quest, skills []models.QuestSkill) error
	FindByID(questID uuid.UUID) (*models.Quest, error)
	FindByIDForUpdate(questID uuid.UUID) (*models.Quest, error)
	UpdateQuest(quest *models.Quest) error
	// UpdateDraftQuest saves quest fields and, when replaceSkills is true,
	// swaps the quest's skill rows inside the same transaction.
	UpdateDraftQuest(quest *models.Quest, skills []models.QuestSkill, replaceSkills bool) error
	ListQuests(filter QuestFilter) ([]models.Quest, string, bool, int64, error)
	CountPartyActiveQuests(partyID uuid.UUID) (int64, error)
	HasActiveQuestWithClient(partyID uuid.UUID, clientID uuid.UUID) (bool, error)
	AcceptQuestAtomic(quest *models.Quest, chatChannel *models.QuestChatChannel, tasks []models.Task, activity *models.QuestActivity, auditLog *models.AuditLog) error
	CreateAuditLog(auditLog *models.AuditLog) error
	GetClientProfile(userID uuid.UUID) (*models.ClientProfile, error)
	ListAssignedQuests(userID uuid.UUID, status string) ([]models.Quest, error)
	// ListQuestsForPartyViewer returns quests where the user is the assigned
	// freelancer (team lead) or an active member of the accepted party.
	ListQuestsForPartyViewer(userID uuid.UUID, status string) ([]models.Quest, error)
	// ToggleSavedQuest bookmarks/unbookmarks a quest and reports the new state.
	ToggleSavedQuest(userID, questID uuid.UUID) (bool, error)
	SavedQuestIDs(userID uuid.UUID, questIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	ListSavedQuests(userID uuid.UUID) ([]models.Quest, error)
	IncrementQuestViews(questID uuid.UUID) error
	CountApplicantsByQuestIDs(questIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	// GetUserSkillNames returns the caller's profile skills, falling back to the
	// skills of the parties they are an active member of.
	GetUserSkillNames(userID uuid.UUID) ([]string, error)
	// CreateQuestActivity appends a single quest activity feed entry.
	CreateQuestActivity(activity *models.QuestActivity) error
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
func (r *questRepository) CreateQuestWithSkills(
	quest *models.Quest,
	skills []models.QuestSkill,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quest).Error; err != nil {
			return err
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

// FindByIDForUpdate is FindByID with a row-level lock (SELECT ... FOR UPDATE).
// Must be called inside a caller-owned transaction; it exists so state checks and
// the writes that depend on them cannot interleave with a concurrent caller.
func (r *questRepository) FindByIDForUpdate(questID uuid.UUID) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
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

// UpdateQuest saves updates to an existing quest (ignoring associations).
func (r *questRepository) UpdateQuest(quest *models.Quest) error {
	// FIX: Foreign Key constraint error ko rokne ke liye Omit lagaya hai
	return r.db.Omit("Client", "AssignedFreelancer", "AcceptedParty", "Milestones", "Skills").Save(quest).Error
}

// ListQuests queries quests based on asymmetric tier visibility, filters, sort and pagination.
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

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("(title ILIKE ? OR description ILIKE ?)", like, like)
	}

	if filter.Category != "" {
		query = query.Where("category ILIKE ?", "%"+filter.Category+"%")
	}

	if filter.Complexity != "" {
		query = query.Where("complexity = ?", filter.Complexity)
	}

	if filter.QuestType != "" {
		query = query.Where("quest_type = ?", filter.QuestType)
	}

	if len(filter.Skills) > 0 {
		names := make([]string, len(filter.Skills))
		for i, s := range filter.Skills {
			names[i] = strings.ToLower(strings.TrimSpace(s))
		}
		sub := r.db.Model(&models.QuestSkill{}).Select("quest_id").Where("LOWER(name) IN ?", names)
		query = query.Where("id IN (?)", sub)
	}

	if filter.TimelineMin != nil {
		query = query.Where("timeline_days >= ?", *filter.TimelineMin)
	}
	if filter.TimelineMax != nil {
		query = query.Where("timeline_days <= ?", *filter.TimelineMax)
	}

	if filter.ClientRank != "" {
		sub := r.db.Model(&models.User{}).Select("id").Where("rank = ?", filter.ClientRank)
		query = query.Where("client_id IN (?)", sub)
	}

	// FIX: budget_inr -> budget_minor
	if filter.MinBudget != nil {
		query = query.Where("budget_minor >= ?", utils.RupeesToPaise(*filter.MinBudget))
	}

	if filter.MaxBudget != nil {
		query = query.Where("budget_minor <= ?", utils.RupeesToPaise(*filter.MaxBudget))
	}

	// Count total matching records before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, "", false, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	// Non-default sorts page by numeric offset; `newest` keeps the timestamp
	// cursor so existing clients continue to work unchanged.
	offsetMode := filter.Sort != "" && filter.Sort != "newest"
	offset := 0
	if offsetMode {
		if n, err := strconv.Atoi(filter.Cursor); err == nil && n > 0 {
			offset = n
		}
	} else if filter.Cursor != "" {
		if cursorTime, err := time.Parse(time.RFC3339Nano, filter.Cursor); err == nil {
			query = query.Where("created_at < ?", cursorTime)
		} else if cursorTime, err := time.Parse(time.RFC3339, filter.Cursor); err == nil {
			query = query.Where("created_at < ?", cursorTime)
		}
	}

	var quests []models.Quest
	err := query.
		Preload("Client").
		Preload("Skills").
		Preload("Milestones").
		Order(questOrderClause(filter.Sort)).
		Offset(offset).
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
	}

	if offsetMode {
		if hasMore {
			nextCursor = strconv.Itoa(offset + limit)
		}
	} else if len(quests) > 0 {
		nextCursor = quests[len(quests)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	return quests, nextCursor, hasMore, total, nil
}

// questOrderClause maps an API sort key to a deterministic ORDER BY clause.
func questOrderClause(sort string) string {
	switch sort {
	case "budget_desc":
		return "budget_minor DESC, id DESC"
	case "budget_asc":
		return "budget_minor ASC, id DESC"
	case "emergency_first":
		return "CASE WHEN quest_type = 'emergency' THEN 0 ELSE 1 END ASC, created_at DESC, id DESC"
	default:
		return "created_at DESC, id DESC"
	}
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
		// FIX: Foreign Key constraint error ko rokne ke liye Omit lagaya hai
		if err := tx.Omit("Client", "AssignedFreelancer", "AcceptedParty", "Milestones", "Skills").Save(quest).Error; err != nil {
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

// ListAssignedQuests returns quests assigned to a freelancer directly OR
// through an active party membership. Kept as the status-required alias.
func (r *questRepository) ListAssignedQuests(userID uuid.UUID, status string) ([]models.Quest, error) {
	return r.ListQuestsForPartyViewer(userID, status)
}

// ListQuestsForPartyViewer returns quests where the user is the assigned
// freelancer (team lead) or an active member of the accepted party. An empty
// status returns every status, so callers can list all of a member's work.
func (r *questRepository) ListQuestsForPartyViewer(userID uuid.UUID, status string) ([]models.Quest, error) {
	// Subquery: active party IDs the user belongs to
	partyIDsQuery := r.db.Model(&models.PartyMember{}).
		Select("party_id").
		Where("user_id = ? AND status = ?", userID, models.MemberActive)

	query := r.db.Model(&models.Quest{}).
		Where("(assigned_freelancer_id = ? OR accepted_party_id IN (?))", userID, partyIDsQuery)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var quests []models.Quest
	err := query.
		Preload("Client").
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Skills").
		Order("updated_at DESC").
		Find(&quests).Error

	return quests, err
}

// UpdateDraftQuest saves editable quest fields and optionally replaces skills.
func (r *questRepository) UpdateDraftQuest(quest *models.Quest, skills []models.QuestSkill, replaceSkills bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Client", "AssignedFreelancer", "AcceptedParty", "Milestones", "Skills").Save(quest).Error; err != nil {
			return err
		}
		if !replaceSkills {
			return nil
		}
		if err := tx.Where("quest_id = ?", quest.ID).Delete(&models.QuestSkill{}).Error; err != nil {
			return err
		}
		if len(skills) > 0 {
			for i := range skills {
				skills[i].QuestID = quest.ID
			}
			if err := tx.Create(&skills).Error; err != nil {
				return err
			}
		}
		quest.Skills = skills
		return nil
	})
}

// ToggleSavedQuest removes an existing bookmark or creates one, returning the
// resulting state (true = saved).
func (r *questRepository) ToggleSavedQuest(userID, questID uuid.UUID) (bool, error) {
	res := r.db.Where("user_id = ? AND quest_id = ?", userID, questID).Delete(&models.SavedQuest{})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return false, nil
	}
	if err := r.db.Create(&models.SavedQuest{UserID: userID, QuestID: questID}).Error; err != nil {
		// A concurrent request may have just inserted the bookmark.
		var count int64
		if e := r.db.Model(&models.SavedQuest{}).
			Where("user_id = ? AND quest_id = ?", userID, questID).
			Count(&count).Error; e == nil && count > 0 {
			return true, nil
		}
		return false, err
	}
	return true, nil
}

// SavedQuestIDs returns the subset of questIDs bookmarked by the user.
func (r *questRepository) SavedQuestIDs(userID uuid.UUID, questIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(questIDs))
	if len(questIDs) == 0 {
		return out, nil
	}
	var ids []uuid.UUID
	err := r.db.Model(&models.SavedQuest{}).
		Where("user_id = ? AND quest_id IN ?", userID, questIDs).
		Pluck("quest_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// ListSavedQuests returns every quest the user bookmarked, newest bookmark last-write wins.
func (r *questRepository) ListSavedQuests(userID uuid.UUID) ([]models.Quest, error) {
	var quests []models.Quest
	err := r.db.Model(&models.Quest{}).
		Where("id IN (?)", r.db.Model(&models.SavedQuest{}).Select("quest_id").Where("user_id = ?", userID)).
		Preload("Client").
		Preload("Skills").
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Order("created_at DESC").
		Find(&quests).Error
	return quests, err
}

// IncrementQuestViews bumps the counter without touching updated_at.
func (r *questRepository) IncrementQuestViews(questID uuid.UUID) error {
	return r.db.Model(&models.Quest{}).
		Where("id = ?", questID).
		UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error
}

// CreateQuestActivity appends a single quest activity feed entry.
func (r *questRepository) CreateQuestActivity(activity *models.QuestActivity) error {
	return r.db.Create(activity).Error
}

// CountApplicantsByQuestIDs returns application counts keyed by quest ID.
func (r *questRepository) CountApplicantsByQuestIDs(questIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	out := make(map[uuid.UUID]int64, len(questIDs))
	if len(questIDs) == 0 {
		return out, nil
	}
	type row struct {
		QuestID uuid.UUID
		Count   int64
	}
	var rows []row
	err := r.db.Model(&models.QuestApplication{}).
		Select("quest_id, COUNT(*) AS count").
		Where("quest_id IN ?", questIDs).
		Group("quest_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.QuestID] = row.Count
	}
	return out, nil
}

// GetUserSkillNames returns the caller's individual-profile skills, falling back
// to the skills of the parties they belong to as an active member.
func (r *questRepository) GetUserSkillNames(userID uuid.UUID) ([]string, error) {
	var profile models.IndividualProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err == nil && len(profile.Skills) > 0 {
		return []string(profile.Skills), nil
	}

	var names []string
	err := r.db.Model(&models.PartySkill{}).
		Joins("JOIN party_members ON party_members.party_id = party_skills.party_id").
		Where("party_members.user_id = ? AND party_members.status = ?", userID, models.MemberActive).
		Pluck("party_skills.skill_name", &names).Error
	if err != nil {
		return nil, err
	}
	return names, nil
}
