package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type ProjectApplicationRepository struct {
}

func NewProjectApplicationRepository() *ProjectApplicationRepository {
	return &ProjectApplicationRepository{}
}

//create application

func (r *ProjectApplicationRepository) Create(application *models.ProjectApplication) error {

	return postgres.DB.Create(application).Error
}

//update application

func (r *ProjectApplicationRepository) Update(application *models.ProjectApplication) error {

	return postgres.DB.Save(application).Error
}

//delete application

func (r *ProjectApplicationRepository) Delete(application *models.ProjectApplication) error {

	return postgres.DB.Delete(application).Error
}

//find by uuid

func (r *ProjectApplicationRepository) FindByUUID(applicationID uuid.UUID) (*models.ProjectApplication, error) {

	var application models.ProjectApplication

	err := postgres.DB.
		Preload("Project").
		Preload("Applicant").
		First(&application, "id = ?", applicationID).Error

	if err != nil {
		return nil, err
	}

	return &application, nil
}

//find by application id string

func (r *ProjectApplicationRepository) FindByID(applicationID string) (*models.ProjectApplication, error) {

	id, err := uuid.Parse(applicationID)
	if err != nil {
		return nil, err
	}

	return r.FindByUUID(id)
}

//find project applications by project id

func (r *ProjectApplicationRepository) FindByProject(projectID uuid.UUID) ([]models.ProjectApplication, error) {

	var applications []models.ProjectApplication

	err := postgres.DB.
		Where("project_id = ?", projectID).
		Preload("Applicant").
		Order("created_at DESC").
		Find(&applications).Error

	return applications, err
}

//find applications by applicant

func (r *ProjectApplicationRepository) FindByApplicant(applicantID uuid.UUID) ([]models.ProjectApplication, error) {

	var applications []models.ProjectApplication

	err := postgres.DB.
		Where("applicant_id = ?", applicantID).
		Preload("Project").
		Order("created_at DESC").
		Find(&applications).Error

	return applications, err
}

//find existing application by project and applicant

func (r *ProjectApplicationRepository) FindExisting(projectID, applicantID uuid.UUID) (*models.ProjectApplication, error) {

	var application models.ProjectApplication

	err := postgres.DB.
		Where("project_id = ? AND applicant_id = ?", projectID, applicantID).
		First(&application).Error

	if err != nil {
		return nil, err
	}

	return &application, nil
}

//update application status

func (r *ProjectApplicationRepository) UpdateStatus(
	applicationID uuid.UUID,
	status models.ApplicationStatus,
) error {

	return postgres.DB.
		Model(&models.ProjectApplication{}).
		Where("id = ?", applicationID).
		Update("status", status).Error
}