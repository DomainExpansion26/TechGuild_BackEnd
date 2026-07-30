package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

type ProjectApplicationService struct {
	applicationRepo *repository.ProjectApplicationRepository
	projectRepo     *repository.ProjectRepository
}

func NewProjectApplicationService() *ProjectApplicationService {
	return &ProjectApplicationService{
		applicationRepo: repository.NewProjectApplicationRepository(),
		projectRepo:     repository.NewProjectRepository(),
	}
}
//apply for a project

func (s *ProjectApplicationService) ApplyProject(
	applicantID string,
	projectID string,
	req dto.ApplyProjectRequest,
) (*dto.ApplyProjectResponse, error) {

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, errors.New("invalid project id")
	}

	applicantUUID, err := uuid.Parse(applicantID)
	if err != nil {
		return nil, errors.New("invalid applicant id")
	}

	project, err := s.projectRepo.FindByUUID(projectUUID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	if project.Status != models.ProjectPublished {
		return nil, errors.New("project is not accepting applications")
	}

	_, err = s.applicationRepo.FindExisting(projectUUID, applicantUUID)
	if err == nil {
		return nil, errors.New("you have already applied for this project")
	}

	application := models.ProjectApplication{
		ProjectID:         projectUUID,
		ApplicantID:       applicantUUID,
		CoverLetter:       req.CoverLetter,
		ProposedBudget:    req.ProposedBudget,
		Currency:          req.Currency,
		EstimatedDuration: req.EstimatedDuration,
		Status:            models.ApplicationPending,
		AppliedAt:         time.Now(),
	}

	if err := s.applicationRepo.Create(&application); err != nil {
		return nil, err
	}

	return &dto.ApplyProjectResponse{
		Message:       "Application submitted successfully",
		ApplicationID: application.ID.String(),
	}, nil
}
//withdraw application

func (s *ProjectApplicationService) WithdrawApplication(
	applicantID string,
	applicationID string,
) error {

	application, err := s.applicationRepo.FindByID(applicationID)
	if err != nil {
		return errors.New("application not found")
	}

	if application.ApplicantID.String() != applicantID {
		return errors.New("unauthorized")
	}

	if application.Status != models.ApplicationPending {
		return errors.New("only pending applications can be withdrawn")
	}

	application.Status = models.ApplicationWithdrawn

	return s.applicationRepo.Update(application)
}
//accept application

func (s *ProjectApplicationService) AcceptApplication(
	applicationID string,
) error {

	application, err := s.applicationRepo.FindByID(applicationID)
	if err != nil {
		return errors.New("application not found")
	}

	if application.Status != models.ApplicationPending &&
		application.Status != models.ApplicationShortlisted {
		return errors.New("application cannot be accepted")
	}

	now := time.Now()

	application.Status = models.ApplicationAccepted
	application.ReviewedAt = &now

	return s.applicationRepo.Update(application)
}

//reject application

func (s *ProjectApplicationService) RejectApplication(
	applicationID string,
) error {

	application, err := s.applicationRepo.FindByID(applicationID)
	if err != nil {
		return errors.New("application not found")
	}

	if application.Status == models.ApplicationAccepted {
		return errors.New("accepted application cannot be rejected")
	}

	now := time.Now()

	application.Status = models.ApplicationRejected
	application.ReviewedAt = &now

	return s.applicationRepo.Update(application)
}

//shortlist application

func (s *ProjectApplicationService) ShortlistApplication(
	applicationID string,
) error {

	application, err := s.applicationRepo.FindByID(applicationID)
	if err != nil {
		return errors.New("application not found")
	}

	if application.Status != models.ApplicationPending {
		return errors.New("only pending applications can be shortlisted")
	}

	now := time.Now()

	application.Status = models.ApplicationShortlisted
	application.ReviewedAt = &now

	return s.applicationRepo.Update(application)
}
//get my applications

func (s *ProjectApplicationService) GetMyApplications(
	applicantID string,
) (*dto.ProjectApplicationListResponse, error) {

	applicantUUID, err := uuid.Parse(applicantID)
	if err != nil {
		return nil, errors.New("invalid applicant id")
	}

	applications, err := s.applicationRepo.FindByApplicant(applicantUUID)
	if err != nil {
		return nil, err
	}

	response := dto.ProjectApplicationListResponse{}

	for _, application := range applications {
		response.Applications = append(
			response.Applications,
			s.convertToApplicationResponse(&application),
		)
	}

	response.Total = len(response.Applications)

	return &response, nil
}
//get project applications by project id

func (s *ProjectApplicationService) GetProjectApplications(
	projectID string,
) (*dto.ProjectApplicationListResponse, error) {

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, errors.New("invalid project id")
	}

	applications, err := s.applicationRepo.FindByProject(projectUUID)
	if err != nil {
		return nil, err
	}

	response := dto.ProjectApplicationListResponse{}

	for _, application := range applications {
		response.Applications = append(
			response.Applications,
			s.convertToApplicationResponse(&application),
		)
	}

	response.Total = len(response.Applications)

	return &response, nil
}
//helper function to convert ProjectApplication model to ProjectApplicationResponse DTO

func (s *ProjectApplicationService) convertToApplicationResponse(
	application *models.ProjectApplication,
) dto.ProjectApplicationResponse {

	response := dto.ProjectApplicationResponse{
		ID:                application.ID.String(),
		ProjectID:         application.ProjectID.String(),
		ApplicantID:       application.ApplicantID.String(),
		CoverLetter:       application.CoverLetter,
		ProposedBudget:    application.ProposedBudget,
		Currency:          application.Currency,
		EstimatedDuration: application.EstimatedDuration,
		Status:            string(application.Status),
		ClientMessage:     application.ClientMessage,
		AppliedAt:         application.AppliedAt.Format(time.RFC3339),
		CreatedAt:         application.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         application.UpdatedAt.Format(time.RFC3339),
	}

	if application.ReviewedAt != nil {
		response.ReviewedAt = application.ReviewedAt.Format(time.RFC3339)
	}

	return response
}