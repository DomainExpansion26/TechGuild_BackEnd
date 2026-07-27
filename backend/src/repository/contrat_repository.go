package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type ContractRepository struct {
}

func NewContractRepository() *ContractRepository {
	return &ContractRepository{}
}

// Create


func (r *ContractRepository) Create(contract *models.ProjectContract) error {

	return postgres.DB.Create(contract).Error
}

// Update
func (r *ContractRepository) Update(contract *models.ProjectContract) error {

	return postgres.DB.Save(contract).Error
}

// Delete
func (r *ContractRepository) Delete(contract *models.ProjectContract) error {

	return postgres.DB.Delete(contract).Error
}

//Finding contract by ID

func (r *ContractRepository) FindByID(
	contractID uuid.UUID,
) (*models.ProjectContract, error) {

	var contract models.ProjectContract

	err := postgres.DB.
		Preload("Project").
		Preload("Application").
		Preload("Client").
		Preload("Freelancer").
		Preload("Milestones").
		First(&contract, "id = ?", contractID).Error

	if err != nil {
		return nil, err
	}

	return &contract, nil
}

// Finding contract by UUID
func (r *ContractRepository) FindByUUID(
	contractID string,
) (*models.ProjectContract, error) {

	id, err := uuid.Parse(contractID)
	if err != nil {
		return nil, err
	}

	return r.FindByID(id)
}

// Find By Project
func (r *ContractRepository) FindByProject(
	projectID uuid.UUID,
) (*models.ProjectContract, error) {

	var contract models.ProjectContract

	err := postgres.DB.
		Preload("Project").
		Preload("Application").
		Preload("Client").
		Preload("Freelancer").
		Preload("Milestones").
		Where("project_id = ?", projectID).
		First(&contract).Error

	if err != nil {
		return nil, err
	}

	return &contract, nil
}

// Find By Application
func (r *ContractRepository) FindByApplication(
	applicationID uuid.UUID,
) (*models.ProjectContract, error) {

	var contract models.ProjectContract

	err := postgres.DB.
		Preload("Project").
		Preload("Application").
		Preload("Client").
		Preload("Freelancer").
		Preload("Milestones").
		Where("application_id = ?", applicationID).
		First(&contract).Error

	if err != nil {
		return nil, err
	}

	return &contract, nil
}

// Find By Client
func (r *ContractRepository) FindByClient(
	clientID uuid.UUID,
) ([]models.ProjectContract, error) {

	var contracts []models.ProjectContract

	err := postgres.DB.
		Where("client_id = ?", clientID).
		Preload("Project").
		Preload("Freelancer").
		Order("created_at DESC").
		Find(&contracts).Error

	return contracts, err
}

// Find By Freelancer
func (r *ContractRepository) FindByFreelancer(
	freelancerID uuid.UUID,
) ([]models.ProjectContract, error) {

	var contracts []models.ProjectContract

	err := postgres.DB.
		Where("freelancer_id = ?", freelancerID).
		Preload("Project").
		Preload("Client").
		Order("created_at DESC").
		Find(&contracts).Error

	return contracts, err
}

//complete the contract

func (r *ContractRepository) CompleteContract(
	contractID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectContract{}).
		Where("id = ?", contractID).
		Update("status", models.ContractCompleted).Error
}

// Cancel Contract
func (r *ContractRepository) CancelContract(
	contractID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectContract{}).
		Where("id = ?", contractID).
		Update("status", models.ContractCancelled).Error
}


//sign by client
func (r *ContractRepository) SignByClient(
	contractID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectContract{}).
		Where("id = ?", contractID).
		Update("signed_by_client", true).Error
}

// Sign By Freelancer
func (r *ContractRepository) SignByFreelancer(
	contractID uuid.UUID,
) error {

	return postgres.DB.
		Model(&models.ProjectContract{}).
		Where("id = ?", contractID).
		Update("signed_by_freelancer", true).Error
}

