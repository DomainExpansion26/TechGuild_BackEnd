package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectRepository struct{}

func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}


// Project CRUD


func (r *ProjectRepository) Create(project *models.Project) error {
	return postgres.DB.Create(project).Error
}

func (r *ProjectRepository) Update(project *models.Project) error {
	return postgres.DB.Save(project).Error
}

func (r *ProjectRepository) Delete(projectID uuid.UUID) error {
	return postgres.DB.Delete(&models.Project{}, "id = ?", projectID).Error
}


// Get Project

func (r *ProjectRepository) FindByID(projectID string) (*models.Project, error) {

	var project models.Project

	err := postgres.DB.
		Preload("Client").
		Preload("Skills").
		Preload("Attachments").
		First(&project, "id = ?", projectID).Error

	if err != nil {
		return nil, err
	}

	return &project, nil
}


func (r *ProjectRepository) FindByClient(clientID string) ([]models.Project, error) {

	var projects []models.Project

	err := postgres.DB.
		Where("client_id = ?", clientID).
		Preload("Skills").
		Preload("Attachments").
		Order("created_at DESC").
		Find(&projects).Error

	return projects, err
}

func (r *ProjectRepository) BrowsePublished(limit, offset int) ([]models.Project, error) {

	var projects []models.Project

	err := postgres.DB.
		Where("status = ?", models.ProjectPublished).
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&projects).Error

	return projects, err
}


// Search


func (r *ProjectRepository) Search(
	keyword string,
	category string,
	experienceLevel string,
	projectType string,
	page int,
	limit int,
) ([]models.Project, error) {

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	db := postgres.DB.Model(&models.Project{}).
		Where("status = ?", models.ProjectPublished)

	if keyword != "" {
		db = db.Where(
			"title ILIKE ? OR description ILIKE ?",
			"%"+keyword+"%",
			"%"+keyword+"%",
		)
	}

	if category != "" {
		db = db.Where("category = ?", category)
	}

	if experienceLevel != "" {
		db = db.Where("experience_level = ?", experienceLevel)
	}

	if projectType != "" {
		db = db.Where("project_type = ?", projectType)
	}

	var projects []models.Project

	err := db.
		Preload("Client").
		Preload("Skills").
		Preload("Attachments").
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&projects).Error

	return projects, err
}


// Status


func (r *ProjectRepository) UpdateStatus(projectID uuid.UUID, status models.ProjectStatus) error {

	return postgres.DB.
		Model(&models.Project{}).
		Where("id = ?", projectID).
		Update("status", status).Error
}

func (r *ProjectRepository) Publish(projectID uuid.UUID) error {

	return postgres.DB.
		Model(&models.Project{}).
		Where("id = ?", projectID).
		Updates(map[string]interface{}{
			"status":       models.ProjectPublished,
			"published_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *ProjectRepository) Close(projectID uuid.UUID) error {
	return r.UpdateStatus(projectID, models.ProjectClosed)
}

func (r *ProjectRepository) Reopen(projectID uuid.UUID) error {
	return r.UpdateStatus(projectID, models.ProjectPublished)
}

// Skills


func (r *ProjectRepository) AddSkills(skills []models.ProjectSkill) error {
	return postgres.DB.Create(&skills).Error
}

func (r *ProjectRepository) DeleteSkills(projectID uuid.UUID) error {
	return postgres.DB.
		Where("project_id = ?", projectID).
		Delete(&models.ProjectSkill{}).Error
}

// Attachments


func (r *ProjectRepository) AddAttachment(file *models.ProjectAttachment) error {
	return postgres.DB.Create(file).Error
}

func (r *ProjectRepository) DeleteAttachment(id uuid.UUID) error {
	return postgres.DB.Delete(&models.ProjectAttachment{}, "id = ?", id).Error
}

func (r *ProjectRepository) GetAttachments(projectID uuid.UUID) ([]models.ProjectAttachment, error) {

	var attachments []models.ProjectAttachment

	err := postgres.DB.
		Where("project_id = ?", projectID).
		Find(&attachments).Error

	return attachments, err
}

// Utility


func (r *ProjectRepository) Exists(projectID uuid.UUID) (bool, error) {

	var count int64

	err := postgres.DB.
		Model(&models.Project{}).
		Where("id = ?", projectID).
		Count(&count).Error

	return count > 0, err
}

func (r *ProjectRepository) CountClientProjects(clientID uuid.UUID) (int64, error) {

	var count int64

	err := postgres.DB.
		Model(&models.Project{}).
		Where("client_id = ?", clientID).
		Count(&count).Error

	return count, err
}
func (r *ProjectRepository) FindByUUID(projectID uuid.UUID) (*models.Project, error) {

	var project models.Project

	err := postgres.DB.
		Preload("Client").
		Preload("Skills").
		Preload("Attachments").
		First(&project, "id = ?", projectID).Error

	if err != nil {
		return nil, err
	}

	return &project, nil
}
//to browse all published projects
func (r *ProjectRepository) BrowseProjects() ([]models.Project, error) {

	var projects []models.Project

	err := postgres.DB.
		Where("status = ?", models.ProjectPublished).
		Preload("Client").
		Preload("Skills").
		Preload("Attachments").
		Order("published_at DESC").
		Find(&projects).Error

	return projects, err
}