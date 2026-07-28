package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type SubmissionRepository struct {
}

func NewSubmissionRepository() *SubmissionRepository {
	return &SubmissionRepository{}
}

// Create

func (r *SubmissionRepository) Create(
	submission *models.ProjectSubmission,
) error {

	return postgres.DB.Create(submission).Error
}

// Update

func (r *SubmissionRepository) Update(
	submission *models.ProjectSubmission,
) error {

	return postgres.DB.Save(submission).Error
}

// Delete

func (r *SubmissionRepository) Delete(
	submission *models.ProjectSubmission,
) error {

	return postgres.DB.Delete(submission).Error
}

// Find By ID

func (r *SubmissionRepository) FindByID(
	submissionID uuid.UUID,
) (*models.ProjectSubmission, error) {

	var submission models.ProjectSubmission

	err := postgres.DB.
		Preload("Milestone").
		First(&submission, "id = ?", submissionID).Error

	if err != nil {
		return nil, err
	}

	return &submission, nil
}

// Find By UUID

func (r *SubmissionRepository) FindByUUID(
	submissionID string,
) (*models.ProjectSubmission, error) {

	id, err := uuid.Parse(submissionID)
	if err != nil {
		return nil, err
	}

	return r.FindByID(id)
}

// Find By Milestone
func (r *SubmissionRepository) FindByMilestone(
	milestoneID uuid.UUID,
) ([]models.ProjectSubmission, error) {

	var submissions []models.ProjectSubmission

	err := postgres.DB.
		Where("milestone_id = ?", milestoneID).
		Order("created_at DESC").
		Find(&submissions).Error

	return submissions, err
}

// Approve Submission
func (r *SubmissionRepository) ApproveSubmission(
	submissionID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectSubmission{}).
		Where("id = ?", submissionID).
		Updates(map[string]interface{}{
			"status":      models.SubmissionApproved,
			"reviewed_at": postgres.DB.NowFunc(),
		}).Error
}

// Reject Submission
func (r *SubmissionRepository) RejectSubmission(
	submissionID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectSubmission{}).
		Where("id = ?", submissionID).
		Updates(map[string]interface{}{
			"status":      models.SubmissionRejected,
			"reviewed_at": postgres.DB.NowFunc(),
		}).Error
}