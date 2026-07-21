package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)

type ProjectService struct {
	projectRepo *repository.ProjectRepository
}

func NewProjectService() *ProjectService {
	return &ProjectService{
		projectRepo: repository.NewProjectRepository(),
	}
}


// Helper Functions


func parseProjectUUID(id string) (uuid.UUID, error) {

	projectID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, errors.New("invalid project id")
	}

	return projectID, nil
}

func parseUserUUID(id string) (uuid.UUID, error) {

	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, errors.New("invalid user id")
	}

	return userID, nil
}


// Project Services


func (s *ProjectService) CreateProject(userID string, req dto.CreateProjectRequest) (*dto.CreateProjectResponse, error) {

	clientID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	project := &models.Project{
		ClientID: clientID,

		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,

		BudgetType: models.BudgetType(req.BudgetType),
		MinBudget:  req.MinBudget,
		MaxBudget:  req.MaxBudget,
		Currency:   req.Currency,

		ExperienceLevel: req.ExperienceLevel,
		ProjectType:     req.ProjectType,
		Duration:        req.Duration,

		Visibility: models.ProjectVisibility(req.Visibility),

		MaxApplications: req.MaxApplications,

		IsFeatured: req.IsFeatured,
		IsUrgent:   req.IsUrgent,

		Status: models.ProjectDraft,
	}

	// Default currency
	if project.Currency == "" {
		project.Currency = "INR"
	}

	// Default visibility
	if project.Visibility == "" {
		project.Visibility = models.VisibilityPublic
	}

	// Parse Dates
	if req.ApplicationDeadline != "" {
		t, err := time.Parse(time.RFC3339, req.ApplicationDeadline)
		if err != nil {
			return nil, errors.New("invalid application deadline")
		}
		project.ApplicationDeadline = &t
	}

	if req.EstimatedStartDate != "" {
		t, err := time.Parse(time.RFC3339, req.EstimatedStartDate)
		if err != nil {
			return nil, errors.New("invalid estimated start date")
		}
		project.EstimatedStartDate = &t
	}

	if req.EstimatedEndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EstimatedEndDate)
		if err != nil {
			return nil, errors.New("invalid estimated end date")
		}
		project.EstimatedEndDate = &t
	}

	// Save Project
	err = s.projectRepo.Create(project)
	if err != nil {
		return nil, err
	}

	// Save Skills
	if len(req.RequiredSkills) > 0 {

		var skills []models.ProjectSkill

		for _, skill := range req.RequiredSkills {

			skills = append(skills, models.ProjectSkill{
				ProjectID: project.ID,
				Skill:     skill,
			})
		}

		err = s.projectRepo.AddSkills(skills)
		if err != nil {
			return nil, err
		}
	}

	return &dto.CreateProjectResponse{
		Message:   "Project created successfully",
		ProjectID: project.ID.String(),
	}, nil
}
func (s *ProjectService) UpdateProject(
	userID string,
	projectID string,
	req dto.UpdateProjectRequest,
) (*dto.UpdateProjectResponse, error) {

	clientID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	pID, err := parseProjectUUID(projectID)
	if err != nil {
		return nil, err
	}

	project, err := s.projectRepo.FindByUUID(pID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	// Authorization
	if project.ClientID != clientID {
		return nil, errors.New("you are not authorized to update this project")
	}

	// ==========================
	// Update Fields
	// ==========================

	if req.Title != "" {
		project.Title = req.Title
	}

	if req.Description != "" {
		project.Description = req.Description
	}

	if req.Category != "" {
		project.Category = req.Category
	}

	if req.BudgetType != "" {
		project.BudgetType = models.BudgetType(req.BudgetType)
	}

	if req.MinBudget > 0 {
		project.MinBudget = req.MinBudget
	}

	if req.MaxBudget > 0 {
		project.MaxBudget = req.MaxBudget
	}

	if req.Currency != "" {
		project.Currency = req.Currency
	}

	if req.ExperienceLevel != "" {
		project.ExperienceLevel = req.ExperienceLevel
	}

	if req.ProjectType != "" {
		project.ProjectType = req.ProjectType
	}

	if req.Duration != "" {
		project.Duration = req.Duration
	}

	if req.Visibility != "" {
		project.Visibility = models.ProjectVisibility(req.Visibility)
	}

	if req.MaxApplications > 0 {
		project.MaxApplications = req.MaxApplications
	}

	project.IsFeatured = req.IsFeatured
	project.IsUrgent = req.IsUrgent

	// ==========================
	// Dates
	// ==========================

	if req.ApplicationDeadline != "" {

		t, err := time.Parse(time.RFC3339, req.ApplicationDeadline)
		if err != nil {
			return nil, errors.New("invalid application deadline")
		}

		project.ApplicationDeadline = &t
	}

	if req.EstimatedStartDate != "" {

		t, err := time.Parse(time.RFC3339, req.EstimatedStartDate)
		if err != nil {
			return nil, errors.New("invalid estimated start date")
		}

		project.EstimatedStartDate = &t
	}

	if req.EstimatedEndDate != "" {

		t, err := time.Parse(time.RFC3339, req.EstimatedEndDate)
		if err != nil {
			return nil, errors.New("invalid estimated end date")
		}

		project.EstimatedEndDate = &t
	}

	// ==========================
	// Save Project
	// ==========================

	err = s.projectRepo.Update(project)
	if err != nil {
		return nil, err
	}

	// ==========================
	// Replace Skills
	// ==========================

	if req.RequiredSkills != nil {

		err = s.projectRepo.DeleteSkills(project.ID)
		if err != nil {
			return nil, err
		}

		if len(req.RequiredSkills) > 0 {

			var skills []models.ProjectSkill

			for _, skill := range req.RequiredSkills {

				skills = append(skills, models.ProjectSkill{
					ProjectID: project.ID,
					Skill:     skill,
				})
			}

			err = s.projectRepo.AddSkills(skills)
			if err != nil {
				return nil, err
			}
		}
	}

	return &dto.UpdateProjectResponse{
		Message: "Project updated successfully",
	}, nil
}

// Delete Project
func (s *ProjectService) DeleteProject(clientID, projectID string) error {

	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("project not found")
	}

	if project.ClientID.String() != clientID {
		return errors.New("you are not allowed to delete this project")
	}

	if project.Status == models.ProjectPublished {
		return errors.New("cannot delete a published project")
	}

	return s.projectRepo.Delete(project.ID) 
}


// Publish Project


func (s *ProjectService) PublishProject(clientID, projectID string) error {

	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("project not found")
	}

	if project.ClientID.String() != clientID {
		return errors.New("you are not allowed to publish this project")
	}

	if project.Status != models.ProjectDraft {
		return errors.New("only draft projects can be published")
	}

	return s.projectRepo.Publish(project.ID)
}

// Close Project

func (s *ProjectService) CloseProject(clientID, projectID string) error {

	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("project not found")
	}

	if project.ClientID.String() != clientID {
		return errors.New("you are not allowed to close this project")
	}

	if project.Status != models.ProjectPublished {
		return errors.New("only published projects can be closed")
	}

	return s.projectRepo.Close(project.ID)
}

// Reopen Project

func (s *ProjectService) ReopenProject(clientID, projectID string) error {

	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("project not found")
	}

	if project.ClientID.String() != clientID {
		return errors.New("you are not allowed to reopen this project")
	}

	if project.Status != models.ProjectClosed {
		return errors.New("only closed projects can be reopened")
	}

	return s.projectRepo.Reopen(project.ID)
}

// Get Project By ID

func (s *ProjectService) GetProjectByID(projectID string) (*dto.ProjectResponse, error) {

	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	response := s.convertToProjectResponse(project)

	return &response, nil
}


// Get My Projects

func (s *ProjectService) GetMyProjects(clientID string) (*dto.ProjectListResponse, error) {

	projects, err := s.projectRepo.FindByClient(clientID)
	if err != nil {
		return nil, err
	}

	response := dto.ProjectListResponse{}

	for _, project := range projects {
		p := s.convertToProjectResponse(&project)
		response.Projects = append(response.Projects, p)
	}

	response.Total = len(response.Projects)

	return &response, nil
}
// Browse Published Projects


func (s *ProjectService) BrowseProjects() (*dto.ProjectListResponse, error) {

	projects, err := s.projectRepo.BrowseProjects()
	if err != nil {
		return nil, err
	}

	response := dto.ProjectListResponse{}

	for _, project := range projects {
		p := s.convertToProjectResponse(&project)
		response.Projects = append(response.Projects, p)
	}

	response.Total = len(response.Projects)

	return &response, nil
}
// Search Projects


func (s *ProjectService) SearchProjects(
	req dto.SearchProjectRequest,
) (*dto.ProjectListResponse, error) {

	projects, err := s.projectRepo.Search(
		req.Keyword,
		req.Category,
		req.ExperienceLevel,
		req.ProjectType,
		req.Page,
		req.Limit,
	)
	if err != nil {
		return nil, err
	}

	response := dto.ProjectListResponse{}

	for _, project := range projects {
		p := s.convertToProjectResponse(&project)
		response.Projects = append(response.Projects, p)
	}

	response.Total = len(response.Projects)

	return &response, nil
}

// =========================
// Helper
// =========================

func (s *ProjectService) convertToProjectResponse(
	project *models.Project,
) dto.ProjectResponse {

	var skills []string

	for _, skill := range project.Skills {
		skills = append(skills, skill.Skill)
	}

	response := dto.ProjectResponse{
		ID:                project.ID.String(),
		ClientID:          project.ClientID.String(),
		Title:             project.Title,
		Description:       project.Description,
		Category:          project.Category,
		BudgetType:        string(project.BudgetType),
		MinBudget:         project.MinBudget,
		MaxBudget:         project.MaxBudget,
		Currency:          project.Currency,
		ExperienceLevel:   project.ExperienceLevel,
		ProjectType:       project.ProjectType,
		Duration:          project.Duration,
		RequiredSkills:    skills,
		Visibility:        string(project.Visibility),
		Status:            string(project.Status),
		MaxApplications:   project.MaxApplications,
		IsFeatured:        project.IsFeatured,
		IsUrgent:          project.IsUrgent,
		CreatedAt:         project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         project.UpdatedAt.Format(time.RFC3339),
	}

	if project.ApplicationDeadline != nil {
		response.ApplicationDeadline = project.ApplicationDeadline.Format(time.RFC3339)
	}

	if project.EstimatedStartDate != nil {
		response.EstimatedStartDate = project.EstimatedStartDate.Format(time.RFC3339)
	}

	if project.EstimatedEndDate != nil {
		response.EstimatedEndDate = project.EstimatedEndDate.Format(time.RFC3339)
	}

	if project.PublishedAt != nil {
		response.PublishedAt = project.PublishedAt.Format(time.RFC3339)
	}

	return response
}