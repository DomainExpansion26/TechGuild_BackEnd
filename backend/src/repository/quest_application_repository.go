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
	Update(app *models.QuestApplication) error
	AcceptApplicationAtomic(
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

// AcceptApplicationAtomic: accept app + reject others + update quest + chat + activity + audit, all in 1 tx.
func (r *questApplicationRepository) AcceptApplicationAtomic(
	acceptedApp *models.QuestApplication,
	quest *models.Quest,
	otherApps []models.QuestApplication,
	chat *models.QuestChatChannel,
	activity *models.QuestActivity,
	audit *models.AuditLog,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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
	})
}
