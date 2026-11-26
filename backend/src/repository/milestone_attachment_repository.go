package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// MilestoneAttachmentRepository stores uploaded milestone deliverables.
type MilestoneAttachmentRepository interface {
	Create(a *models.MilestoneAttachment) error
	ListByMilestone(milestoneID uuid.UUID) ([]models.MilestoneAttachment, error)
	FindByIDAndMilestone(id, milestoneID uuid.UUID) (*models.MilestoneAttachment, error)
	Delete(id uuid.UUID) error
	CountByQuestGrouped(questID uuid.UUID) (map[uuid.UUID]int64, error)
}

type milestoneAttachmentRepository struct {
	db *gorm.DB
}

func NewMilestoneAttachmentRepository() MilestoneAttachmentRepository {
	return &milestoneAttachmentRepository{db: postgres.DB}
}

func NewMilestoneAttachmentRepositoryWithDB(db *gorm.DB) MilestoneAttachmentRepository {
	return &milestoneAttachmentRepository{db: db}
}

func (r *milestoneAttachmentRepository) Create(a *models.MilestoneAttachment) error {
	return r.db.Create(a).Error
}

func (r *milestoneAttachmentRepository) ListByMilestone(milestoneID uuid.UUID) ([]models.MilestoneAttachment, error) {
	var rows []models.MilestoneAttachment
	err := r.db.Where("milestone_id = ?", milestoneID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *milestoneAttachmentRepository) FindByIDAndMilestone(id, milestoneID uuid.UUID) (*models.MilestoneAttachment, error) {
	var row models.MilestoneAttachment
	err := r.db.Where("id = ? AND milestone_id = ?", id, milestoneID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *milestoneAttachmentRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.MilestoneAttachment{}).Error
}

// CountByQuestGrouped returns attachment counts per milestone for one quest.
func (r *milestoneAttachmentRepository) CountByQuestGrouped(questID uuid.UUID) (map[uuid.UUID]int64, error) {
	type row struct {
		MilestoneID uuid.UUID
		Total       int64
	}
	var rows []row
	err := r.db.Model(&models.MilestoneAttachment{}).
		Select("milestone_id, COUNT(*) AS total").
		Where("quest_id = ?", questID).
		Group("milestone_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[uuid.UUID]int64, len(rows))
	for _, r := range rows {
		counts[r.MilestoneID] = r.Total
	}
	return counts, nil
}
