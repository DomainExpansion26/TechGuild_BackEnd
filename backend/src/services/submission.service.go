package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)

type SubmissionService struct {
	submissionRepo *repository.SubmissionRepository
	milestoneRepo  *repository.MilestoneRepository
	contractRepo   *repository.ContractRepository
}

func NewSubmissionService() *SubmissionService {
	return &SubmissionService{
		submissionRepo: repository.NewSubmissionRepository(),
		milestoneRepo:  repository.NewMilestoneRepository(),
		contractRepo:   repository.NewContractRepository(),
	}
}
//create submission
func (s *SubmissionService) CreateSubmission(
	freelancerID string,
	req dto.CreateSubmissionRequest,
) (*dto.CreateSubmissionResponse, error) {

	milestoneID, err := uuid.Parse(req.MilestoneID)
	if err != nil {
		return nil, errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return nil, err
	}

	if contract.FreelancerID.String() != freelancerID {
		return nil, errors.New("unauthorized")
	}

	submission := models.ProjectSubmission{
	MilestoneID:  milestone.ID,
	Message:      req.Message,
	SubmissionURL: req.SubmissionURL,
	AttachmentURL: req.AttachmentURL,
	Status:       models.SubmissionPending,
	}

	if err := s.submissionRepo.Create(&submission); err != nil {
		return nil, err
	}

	return &dto.CreateSubmissionResponse{
		Message:      "Submission created successfully",
		SubmissionID: submission.ID.String(),
	}, nil
}

//update submission 
func (s *SubmissionService) UpdateSubmission(
	freelancerID string,
	submissionID string,
	req dto.UpdateSubmissionRequest,
) error {

	id, err := uuid.Parse(submissionID)
	if err != nil {
		return errors.New("invalid submission id")
	}

	submission, err := s.submissionRepo.FindByID(id)
	if err != nil {
		return errors.New("submission not found")
	}

	milestone, err := s.milestoneRepo.FindByID(submission.MilestoneID)
	if err != nil {
		return err
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.FreelancerID.String() != freelancerID {
		return errors.New("unauthorized")
	}

	if req.Message != "" {
		submission.Message = req.Message
	}

	if req.SubmissionURL != "" {
		submission.SubmissionURL = req.SubmissionURL
	}

	if req.AttachmentURL != "" {
		submission.AttachmentURL = req.AttachmentURL
	}
	return s.submissionRepo.Update(submission)
}

//delete submission
func (s *SubmissionService) DeleteSubmission(
	freelancerID string,
	submissionID string,
) error {

	id, err := uuid.Parse(submissionID)
	if err != nil {
		return errors.New("invalid submission id")
	}

	submission, err := s.submissionRepo.FindByID(id)
	if err != nil {
		return errors.New("submission not found")
	}

	milestone, err := s.milestoneRepo.FindByID(submission.MilestoneID)
	if err != nil {
		return err
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.FreelancerID.String() != freelancerID {
		return errors.New("unauthorized")
	}

	return s.submissionRepo.Delete(submission)
}

//approve submission 
func (s *SubmissionService) ApproveSubmission(
	clientID string,
	submissionID string,
) error {

	id, err := uuid.Parse(submissionID)
	if err != nil {
		return errors.New("invalid submission id")
	}

	submission, err := s.submissionRepo.FindByID(id)
	if err != nil {
		return errors.New("submission not found")
	}

	milestone, err := s.milestoneRepo.FindByID(submission.MilestoneID)
	if err != nil {
		return err
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if submission.Status != models.SubmissionPending {
		return errors.New("submission already reviewed")
	}

	return s.submissionRepo.ApproveSubmission(submission.ID)
}

//reject submission
func (s *SubmissionService) RejectSubmission(
	clientID string,
	submissionID string,
) error {

	id, err := uuid.Parse(submissionID)
	if err != nil {
		return errors.New("invalid submission id")
	}

	submission, err := s.submissionRepo.FindByID(id)
	if err != nil {
		return errors.New("submission not found")
	}

	milestone, err := s.milestoneRepo.FindByID(submission.MilestoneID)
	if err != nil {
		return err
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if submission.Status != models.SubmissionPending {
		return errors.New("submission already reviewed")
	}

	return s.submissionRepo.RejectSubmission(submission.ID)
}

//get submiision by id 
func (s *SubmissionService) GetSubmissionByID(
	submissionID string,
) (*dto.SubmissionResponse, error) {

	submission, err := s.submissionRepo.FindByUUID(submissionID)
	if err != nil {
		return nil, errors.New("submission not found")
	}

	response := s.convertToSubmissionResponse(submission)

	return &response, nil
}

//get submission by milestone 
func (s *SubmissionService) GetMilestoneSubmissions(
	milestoneID string,
) (*dto.SubmissionListResponse, error) {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return nil, errors.New("invalid milestone id")
	}

	submissions, err := s.submissionRepo.FindByMilestone(id)
	if err != nil {
		return nil, err
	}

	response := dto.SubmissionListResponse{}

	for _, submission := range submissions {
		response.Submissions = append(
			response.Submissions,
			s.convertToSubmissionResponse(&submission),
		)
	}

	response.Total = len(response.Submissions)

	return &response, nil
}

//helper function to convert model to dto 
func (s *SubmissionService) convertToSubmissionResponse(
	submission *models.ProjectSubmission,
) dto.SubmissionResponse {

	response := dto.SubmissionResponse{
		ID:            submission.ID.String(),
		MilestoneID:   submission.MilestoneID.String(),
		Message:       submission.Message,
		SubmissionURL: submission.SubmissionURL,
		AttachmentURL: submission.AttachmentURL,
		Status:        string(submission.Status),
		SubmittedAt:   submission.SubmittedAt.Format(time.RFC3339),
		CreatedAt:     submission.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     submission.UpdatedAt.Format(time.RFC3339),
	}

	if submission.ReviewedAt != nil {
		response.ReviewedAt = submission.ReviewedAt.Format(time.RFC3339)
	}

	return response
}