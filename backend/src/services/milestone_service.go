package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)

type MilestoneService struct {
	milestoneRepo *repository.MilestoneRepository
	contractRepo  *repository.ContractRepository
}

func NewMilestoneService() *MilestoneService {
	return &MilestoneService{
		milestoneRepo: repository.NewMilestoneRepository(),
		contractRepo:  repository.NewContractRepository(),
	}
}


//crete milestone
func (s *MilestoneService) CreateMilestone(
	clientID string,
	req dto.CreateMilestoneRequest,
) (*dto.CreateMilestoneResponse, error) {

	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		return nil, errors.New("invalid contract id")
	}

	contract, err := s.contractRepo.FindByID(contractID)
	if err != nil {
		return nil, errors.New("contract not found")
	}

	if contract.ClientID.String() != clientID {
		return nil, errors.New("unauthorized")
	}

	milestone := models.ProjectMilestone{
		ContractID: contract.ID,
		Title:      req.Title,
		Description:req.Description,
		Amount:     req.Amount,
		Status:     models.MilestonePending,
	}

	if req.DueDate != "" {
		t, _ := time.Parse(time.RFC3339, req.DueDate)
		milestone.DueDate = &t
	}

	if err := s.milestoneRepo.Create(&milestone); err != nil {
		return nil, err
	}

	return &dto.CreateMilestoneResponse{
		Message:     "Milestone created successfully",
		MilestoneID: milestone.ID.String(),
	}, nil
}


//updste milestone
func (s *MilestoneService) UpdateMilestone(
	clientID string,
	milestoneID string,
	req dto.UpdateMilestoneRequest,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if req.Title != "" {
		milestone.Title = req.Title
	}

	if req.Description != "" {
		milestone.Description = req.Description
	}

	if req.Amount > 0 {
		milestone.Amount = req.Amount
	}

	if req.DueDate != "" {
		t, _ := time.Parse(time.RFC3339, req.DueDate)
		milestone.DueDate = &t
	}

	return s.milestoneRepo.Update(milestone)
}


//delete milestone
func (s *MilestoneService) DeleteMilestone(
	clientID string,
	milestoneID string,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	return s.milestoneRepo.Delete(milestone)
}

//submit milestone
func (s *MilestoneService) SubmitMilestone(
	freelancerID string,
	milestoneID string,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.FreelancerID.String() != freelancerID {
		return errors.New("unauthorized")
	}

	if milestone.Status != models.MilestonePending {
		return errors.New("milestone cannot be submitted")
	}

	return s.milestoneRepo.SubmitMilestone(milestone.ID)
}


//approve milestone
func (s *MilestoneService) ApproveMilestone(
	clientID string,
	milestoneID string,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if milestone.Status != models.MilestoneSubmitted {
		return errors.New("only submitted milestones can be approved")
	}

	return s.milestoneRepo.ApproveMilestone(milestone.ID)
}

//reject milestone
func (s *MilestoneService) RejectMilestone(
	clientID string,
	milestoneID string,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if milestone.Status != models.MilestoneSubmitted {
		return errors.New("only submitted milestones can be rejected")
	}

	return s.milestoneRepo.RejectMilestone(milestone.ID)
}

func (s *MilestoneService) MarkMilestonePaid(
	clientID string,
	milestoneID string,
) error {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return errors.New("milestone not found")
	}

	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return err
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if milestone.Status != models.MilestoneApproved {
		return errors.New("only approved milestones can be marked as paid")
	}

	return s.milestoneRepo.MarkAsPaid(milestone.ID)
}

// Get Milestone By ID
func (s *MilestoneService) GetMilestoneByID(
	milestoneID string,
) (*dto.MilestoneResponse, error) {

	id, err := uuid.Parse(milestoneID)
	if err != nil {
		return nil, errors.New("invalid milestone id")
	}

	milestone, err := s.milestoneRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	response := s.convertToMilestoneResponse(milestone)

	return &response, nil
}

// Get Contract Milestones
func (s *MilestoneService) GetContractMilestones(
	contractID string,
) (*dto.MilestoneListResponse, error) {

	id, err := uuid.Parse(contractID)
	if err != nil {
		return nil, errors.New("invalid contract id")
	}

	milestones, err := s.milestoneRepo.FindByContract(id)
	if err != nil {
		return nil, err
	}

	response := dto.MilestoneListResponse{}

	for _, milestone := range milestones {
		response.Milestones = append(
			response.Milestones,
			s.convertToMilestoneResponse(&milestone),
		)
	}

	response.Total = len(response.Milestones)

	return &response, nil
}

// Helper to convert database model into dto response 
func (s *MilestoneService) convertToMilestoneResponse(
	milestone *models.ProjectMilestone,
) dto.MilestoneResponse {

	response := dto.MilestoneResponse{
		ID:          milestone.ID.String(),
		ContractID:  milestone.ContractID.String(),
		Title:       milestone.Title,
		Description: milestone.Description,
		Amount:      milestone.Amount,
		Status:      string(milestone.Status),
		CreatedAt:   milestone.CreatedAt.Format(time.RFC3339),
	}

	if milestone.DueDate != nil {
		response.DueDate = milestone.DueDate.Format(time.RFC3339)
	}

	if milestone.CompletedAt != nil {
		response.CompletedAt = milestone.CompletedAt.Format(time.RFC3339)
	}

	return response
}