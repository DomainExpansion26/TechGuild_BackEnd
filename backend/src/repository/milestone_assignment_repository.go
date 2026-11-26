package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// MilestoneAssignmentRepository stores which party members work on a milestone.
type MilestoneAssignmentRepository interface {
	// ReplaceForMilestone deletes the milestone's current assignments and
	// inserts the new set in one transaction (re-assignment semantics).
	ReplaceForMilestone(milestoneID uuid.UUID, rows []models.MilestoneAssignment) error
	DeleteByMilestone(milestoneID uuid.UUID) error
	ListByMilestone(milestoneID uuid.UUID) ([]models.MilestoneAssignment, error)
	ListByQuest(questID uuid.UUID) ([]models.MilestoneAssignment, error)
	ListByMemberInQuest(questID, memberID uuid.UUID) ([]models.MilestoneAssignment, error)
	ExistsForMilestoneAndMember(milestoneID, memberID uuid.UUID) (bool, error)
	// CountAssignedMilestonesInQuest counts distinct milestones that have at
	// least one assignee.
	CountAssignedMilestonesInQuest(questID uuid.UUID) (int64, error)
}

type milestoneAssignmentRepository struct {
	db *gorm.DB
}

func NewMilestoneAssignmentRepository() MilestoneAssignmentRepository {
	return &milestoneAssignmentRepository{db: postgres.DB}
}

func NewMilestoneAssignmentRepositoryWithDB(db *gorm.DB) MilestoneAssignmentRepository {
	return &milestoneAssignmentRepository{db: db}
}

func (r *milestoneAssignmentRepository) ReplaceForMilestone(milestoneID uuid.UUID, rows []models.MilestoneAssignment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("milestone_id = ?", milestoneID).
			Delete(&models.MilestoneAssignment{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			rows[i].MilestoneID = milestoneID
		}
		return tx.Create(&rows).Error
	})
}

func (r *milestoneAssignmentRepository) DeleteByMilestone(milestoneID uuid.UUID) error {
	return r.db.Where("milestone_id = ?", milestoneID).
		Delete(&models.MilestoneAssignment{}).Error
}

func (r *milestoneAssignmentRepository) ListByMilestone(milestoneID uuid.UUID) ([]models.MilestoneAssignment, error) {
	var rows []models.MilestoneAssignment
	err := r.db.Where("milestone_id = ?", milestoneID).
		Preload("Member").
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *milestoneAssignmentRepository) ListByQuest(questID uuid.UUID) ([]models.MilestoneAssignment, error) {
	var rows []models.MilestoneAssignment
	err := r.db.
		Joins("JOIN quest_milestones ON quest_milestones.id = milestone_assignments.milestone_id").
		Where("quest_milestones.quest_id = ?", questID).
		Preload("Member").
		Order("milestone_assignments.created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *milestoneAssignmentRepository) ListByMemberInQuest(questID, memberID uuid.UUID) ([]models.MilestoneAssignment, error) {
	var rows []models.MilestoneAssignment
	err := r.db.
		Joins("JOIN quest_milestones ON quest_milestones.id = milestone_assignments.milestone_id").
		Where("quest_milestones.quest_id = ? AND milestone_assignments.member_id = ?", questID, memberID).
		Order("milestone_assignments.created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *milestoneAssignmentRepository) ExistsForMilestoneAndMember(milestoneID, memberID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.MilestoneAssignment{}).
		Where("milestone_id = ? AND member_id = ?", milestoneID, memberID).
		Count(&count).Error
	return count > 0, err
}

func (r *milestoneAssignmentRepository) CountAssignedMilestonesInQuest(questID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.MilestoneAssignment{}).
		Joins("JOIN quest_milestones ON quest_milestones.id = milestone_assignments.milestone_id").
		Where("quest_milestones.quest_id = ?", questID).
		Distinct("milestone_assignments.milestone_id").
		Count(&count).Error
	return count, err
}
