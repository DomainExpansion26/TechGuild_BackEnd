package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)

type ContractService struct {
	contractRepo    *repository.ContractRepository
	projectRepo     *repository.ProjectRepository
	applicationRepo *repository.ProjectApplicationRepository
}

func NewContractService() *ContractService {
	return &ContractService{
		contractRepo:    repository.NewContractRepository(),
		projectRepo:     repository.NewProjectRepository(),
		applicationRepo: repository.NewProjectApplicationRepository(),
	}
}

// Create Contract
func (s *ContractService) CreateContract(
	clientID string,
	req dto.CreateContractRequest,
) (*dto.CreateContractResponse, error) {

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, errors.New("invalid project id")
	}

	project, err := s.projectRepo.FindByUUID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	if project.ClientID.String() != clientID {
		return nil, errors.New("unauthorized")
	}

	applicationID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		return nil, errors.New("invalid application id")
	}

	application, err := s.applicationRepo.FindByUUID(applicationID)
	if err != nil {
		return nil, errors.New("application not found")
	}

	contract := models.ProjectContract{
		ProjectID:      project.ID,
		ApplicationID:  application.ID,
		ClientID:       project.ClientID,
		FreelancerID: application.ApplicantID,
		ContractAmount: req.ContractAmount,
		Currency:       req.Currency,
		Status:         models.ContractPending,
	}

	if req.StartDate != "" {
		t, _ := time.Parse(time.RFC3339, req.StartDate)
		contract.StartDate = &t
	}

	if req.ExpectedEndDate != "" {
		t, _ := time.Parse(time.RFC3339, req.ExpectedEndDate)
		contract.ExpectedEndDate = &t
	}

	if err := s.contractRepo.Create(&contract); err != nil {
		return nil, err
	}

	return &dto.CreateContractResponse{
		Message:    "Contract created successfully",
		ContractID: contract.ID.String(),
	}, nil
}

// Sign Contract
func (s *ContractService) SignContract(
	userID string,
	contractID string,
) error {

	id, err := uuid.Parse(contractID)
	if err != nil {
		return errors.New("invalid contract id")
	}

	contract, err := s.contractRepo.FindByID(id)
	if err != nil {
		return errors.New("contract not found")
	}

	if contract.ClientID.String() == userID {

		if err := s.contractRepo.SignByClient(contract.ID); err != nil {
			return err
		}

	} else if contract.FreelancerID.String() == userID {

		if err := s.contractRepo.SignByFreelancer(contract.ID); err != nil {
			return err
		}

	} else {

		return errors.New("unauthorized")
	}

	contract, err = s.contractRepo.FindByID(contract.ID)
	if err != nil {
		return err
	}

	if contract.SignedByClient && contract.SignedByFreelancer {

		if err := s.contractRepo.ActivateContract(contract.ID); err != nil {
			return err
		}
	}

	return nil
}

// Complete Contract
func (s *ContractService) CompleteContract(
	clientID string,
	contractID string,
) error {

	id, err := uuid.Parse(contractID)
	if err != nil {
		return errors.New("invalid contract id")
	}

	contract, err := s.contractRepo.FindByID(id)
	if err != nil {
		return errors.New("contract not found")
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if contract.Status != models.ContractActive {
		return errors.New("contract is not active")
	}

	return s.contractRepo.CompleteContract(contract.ID)
}

// Get Contract By ID
func (s *ContractService) GetContractByID(
	contractID string,
) (*dto.ContractResponse, error) {

	contract, err := s.contractRepo.FindByUUID(contractID)
	if err != nil {
		return nil, errors.New("contract not found")
	}

	response := s.convertToContractResponse(contract)

	return &response, nil
}

// Get Client Contracts
func (s *ContractService) GetClientContracts(
	clientID string,
) (*dto.ContractListResponse, error) {

	id, err := uuid.Parse(clientID)
	if err != nil {
		return nil, errors.New("invalid client id")
	}

	contracts, err := s.contractRepo.FindByClient(id)
	if err != nil {
		return nil, err
	}

	response := dto.ContractListResponse{}

	for _, contract := range contracts {
		response.Contracts = append(
			response.Contracts,
			s.convertToContractResponse(&contract),
		)
	}

	response.Total = len(response.Contracts)

	return &response, nil
}

// Get Freelancer Contracts
func (s *ContractService) GetFreelancerContracts(
	freelancerID string,
) (*dto.ContractListResponse, error) {

	id, err := uuid.Parse(freelancerID)
	if err != nil {
		return nil, errors.New("invalid freelancer id")
	}

	contracts, err := s.contractRepo.FindByFreelancer(id)
	if err != nil {
		return nil, err
	}

	response := dto.ContractListResponse{}

	for _, contract := range contracts {
		response.Contracts = append(
			response.Contracts,
			s.convertToContractResponse(&contract),
		)
	}

	response.Total = len(response.Contracts)

	return &response, nil
}


// Helper to convert 
func (s *ContractService) convertToContractResponse(
	contract *models.ProjectContract,
) dto.ContractResponse {

	response := dto.ContractResponse{
		ID:                  contract.ID.String(),
		ProjectID:           contract.ProjectID.String(),
		ApplicationID:       contract.ApplicationID.String(),
		ClientID:            contract.ClientID.String(),
		FreelancerID:        contract.FreelancerID.String(),
		ContractAmount:      contract.ContractAmount,
		Currency:            contract.Currency,
		Status:              string(contract.Status),
		SignedByClient:      contract.SignedByClient,
		SignedByFreelancer:  contract.SignedByFreelancer,
		CreatedAt:           contract.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           contract.UpdatedAt.Format(time.RFC3339),
	}

	if contract.StartDate != nil {
		response.StartDate = contract.StartDate.Format(time.RFC3339)
	}

	if contract.ExpectedEndDate != nil {
		response.ExpectedEndDate = contract.ExpectedEndDate.Format(time.RFC3339)
	}

	if contract.CompletedAt != nil {
		response.CompletedAt = contract.CompletedAt.Format(time.RFC3339)
	}

	return response
}

//cancel the contract 
func (s *ContractService) CancelContract(
	clientID string,
	contractID string,
) error {

	id, err := uuid.Parse(contractID)
	if err != nil {
		return errors.New("invalid contract id")
	}

	contract, err := s.contractRepo.FindByID(id)
	if err != nil {
		return errors.New("contract not found")
	}

	if contract.ClientID.String() != clientID {
		return errors.New("unauthorized")
	}

	if contract.Status == models.ContractCompleted {
		return errors.New("completed contract cannot be cancelled")
	}

	return s.contractRepo.CancelContract(contract.ID)
}