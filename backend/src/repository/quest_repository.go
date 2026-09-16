package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type QuestRepository interface {
	CreateQuestWithMilestonesAndSkills(quest *models.Quest, milestones []models.QuestMilestone, skills []models.QuestSkill) error
	FindByID(questID uuid.UUID) (*models.Quest, error)
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

// FindByID retrieves a quest by its UUID, preloading Client, Milestones, and Skills.
func (r *questRepository) FindByID(questID uuid.UUID) (*models.Quest, error) {
	var quest models.Quest
	err := r.db.
		Preload("Client").
		Preload("Milestones").
		Preload("Skills").
		First(&quest, "id = ?", questID).Error

	if err != nil {
		return nil, err
	}
	return &quest, nil
}
