package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type MilestoneRepository struct {
}

func NewMilestoneRepository() *MilestoneRepository {
	return &MilestoneRepository{}
}

// create milestone
func (r *MilestoneRepository) Create(
	milestone *models.ProjectMilestone,
) error {

	return postgres.DB.Create(milestone).Error
}

//update milestone
func (r *MilestoneRepository) Update(
	milestone *models.ProjectMilestone,
) error {

	return postgres.DB.Save(milestone).Error
}


//delete milestone
func (r *MilestoneRepository) Delete(
	milestone *models.ProjectMilestone,
) error {

	return postgres.DB.Delete(milestone).Error
}

//finding milestone by ID
func (r *MilestoneRepository) FindByID(
	milestoneID uuid.UUID,
) (*models.ProjectMilestone, error) {

	var milestone models.ProjectMilestone

	err := postgres.DB.
		Preload("Contract").
		Preload("Submissions").
		First(&milestone, "id = ?", milestoneID).Error

	if err != nil {
		return nil, err
	}

	return &milestone, nil
}

//finding by UUID
func (r *MilestoneRepository) FindByUUID(
	milestoneID string,
) (*models.ProjectMilestone, error) {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return nil, err
	}

	return r.FindByID(id)
}

//finding milestone by contract
func (r *MilestoneRepository) FindByContract(
	contractID uuid.UUID,
) ([]models.ProjectMilestone, error) {

	var milestones []models.ProjectMilestone

	err := postgres.DB.
		Where("contract_id = ?", contractID).
		Order("created_at ASC").
		Find(&milestones).Error

	return milestones, err
}

// Start Milestone
func (r *MilestoneRepository) StartMilestone(
	milestoneID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectMilestone{}).
		Where("id = ?", milestoneID).
		Update("status", models.MilestoneInProgress).Error
}

// Submit Milestone
func (r *MilestoneRepository) SubmitMilestone(
	milestoneID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectMilestone{}).
		Where("id = ?", milestoneID).
		Update("status", models.MilestoneSubmitted).Error
}

// Approve Milestone
func (r *MilestoneRepository) ApproveMilestone(
	milestoneID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneApproved,
			"completed_at": postgres.DB.NowFunc(),
		}).Error
}

// Reject Milestone
func (r *MilestoneRepository) RejectMilestone(
	milestoneID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectMilestone{}).
		Where("id = ?", milestoneID).
		Update("status", models.MilestoneRejected).Error
}

// Mark As Paid
func (r *MilestoneRepository) MarkAsPaid(
	milestoneID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectMilestone{}).
		Where("id = ?", milestoneID).
		Update("status", models.MilestonePaid).Error
}