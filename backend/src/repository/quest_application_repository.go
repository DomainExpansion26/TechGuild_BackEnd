package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type QuestApplicationRepository interface {
	Create(app *models.QuestApplication) error
	FindByID(id uuid.UUID) (*models.QuestApplication, error)
	FindByQuestAndFreelancer(questID, freelancerID uuid.UUID) (*models.QuestApplication, error)
	ListByQuestID(questID uuid.UUID, status string) ([]models.QuestApplication, error)
	// CountByStatusForFreelancer returns application counts grouped by status.
	CountByStatusForFreelancer(freelancerID uuid.UUID) (map[string]int64, error)
	// ListRecentByFreelancer returns the freelancer's latest applications with Quest preloaded.
	ListRecentByFreelancer(freelancerID uuid.UUID, limit int) ([]models.QuestApplication, error)
	Update(app *models.QuestApplication) error
	AcceptApplicationAtomic(
		tx *gorm.DB,
		acceptedApp *models.QuestApplication,
		quest *models.Quest,
		otherApps []models.QuestApplication,
		chat *models.QuestChatChannel,
		activity *models.QuestActivity,
		audit *models.AuditLog,
	) error
}

type questApplicationRepository struct {
	db *gorm.DB
}

func NewQuestApplicationRepository() QuestApplicationRepository {
	return &questApplicationRepository{db: postgres.DB}
}

func NewQuestApplicationRepositoryTx(tx *gorm.DB) QuestApplicationRepository {
	return &questApplicationRepository{db: tx}
}

func (r *questApplicationRepository) Create(app *models.QuestApplication) error {
	return r.db.Create(app).Error
}

func (r *questApplicationRepository) FindByID(id uuid.UUID) (*models.QuestApplication, error) {
	var app models.QuestApplication
	err := r.db.Preload("Freelancer").Preload("Party").First(&app, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *questApplicationRepository) FindByQuestAndFreelancer(questID, freelancerID uuid.UUID) (*models.QuestApplication, error) {
	var app models.QuestApplication
	err := r.db.Where("quest_id = ? AND freelancer_id = ?", questID, freelancerID).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *questApplicationRepository) ListByQuestID(questID uuid.UUID, status string) ([]models.QuestApplication, error) {
	q := r.db.Where("quest_id = ?", questID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var apps []models.QuestApplication
	err := q.Preload("Freelancer").Preload("Party").Order("created_at DESC").Find(&apps).Error
	return apps, err
}

func (r *questApplicationRepository) Update(app *models.QuestApplication) error {
	return r.db.Omit("Quest", "Freelancer", "Party").Save(app).Error
}

// AcceptApplicationAtomic: accept app + reject others + update quest + chat + activity + audit,
// executed on the CALLER's transaction so the quest row lock taken by the service stays held.
func (r *questApplicationRepository) AcceptApplicationAtomic(
	tx *gorm.DB,
	acceptedApp *models.QuestApplication,
	quest *models.Quest,
	otherApps []models.QuestApplication,
	chat *models.QuestChatChannel,
	activity *models.QuestActivity,
	audit *models.AuditLog,
) error {
	if err := tx.Omit("Quest", "Freelancer", "Party").Save(acceptedApp).Error; err != nil {
		return err
	}
	for i := range otherApps {
		if err := tx.Omit("Quest", "Freelancer", "Party").Save(&otherApps[i]).Error; err != nil {
			return err
		}
	}
	if err := tx.Omit("Client", "AssignedFreelancer", "AcceptedParty", "Milestones", "Skills").Save(quest).Error; err != nil {
		return err
	}
	if chat != nil {
		if err := tx.Create(chat).Error; err != nil {
			return err
		}
	}
	if activity != nil {
		if err := tx.Create(activity).Error; err != nil {
			return err
		}
	}
	if audit != nil {
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}
	return nil
}

// CountByStatusForFreelancer returns application counts grouped by status.
func (r *questApplicationRepository) CountByStatusForFreelancer(freelancerID uuid.UUID) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := r.db.Model(&models.QuestApplication{}).
		Select("status, COUNT(*) AS count").
		Where("freelancer_id = ?", freelancerID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Count
	}
	return out, nil
}

// ListRecentByFreelancer returns the freelancer's latest applications, newest first.
func (r *questApplicationRepository) ListRecentByFreelancer(freelancerID uuid.UUID, limit int) ([]models.QuestApplication, error) {
	if limit <= 0 {
		limit = 5
	}
	var apps []models.QuestApplication
	err := r.db.Preload("Quest").
		Where("freelancer_id = ?", freelancerID).
		Order("created_at DESC").
		Limit(limit).
		Find(&apps).Error
	return apps, err
}
