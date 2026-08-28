package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/services"

	"github.com/danielgtaylor/huma/v2"
)

type ContractController struct {
	service *services.ContractService
}

func NewContractController() *ContractController {
	return &ContractController{
		service: services.NewContractService(),
	}
}

var contractController = NewContractController()

// ---------- CreateContract ----------

func CreateContractHandler(ctx context.Context, input *dto.CreateContractInput) (*dto.CreateContractOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	res, err := contractController.service.CreateContract(clientID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateContractOutput{Body: *res}, nil
}

// ---------- SignContract ----------

func SignContractHandler(ctx context.Context, input *dto.SignContractInput) (*dto.SignContractOutput, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := contractController.service.SignContract(userID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SignContractOutput{
		Body: dto.SignContractResponse{Message: "Contract signed successfully"},
	}, nil
}

// ---------- CompleteContract ----------

func CompleteContractHandler(ctx context.Context, input *dto.CompleteContractInput) (*dto.CompleteContractOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := contractController.service.CompleteContract(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CompleteContractOutput{
		Body: dto.CompleteContractResponse{Message: "Contract completed successfully"},
	}, nil
}

// ---------- CancelContract ----------

func CancelContractHandler(ctx context.Context, input *dto.CancelContractInput) (*dto.CancelContractOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := contractController.service.CancelContract(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CancelContractOutput{
		Body: dto.CancelContractResponse{Message: "Contract cancelled successfully"},
	}, nil
}

// ---------- GetContractByID ----------

func GetContractByIDHandler(ctx context.Context, input *dto.GetContractByIDInput) (*dto.GetContractByIDOutput, error) {
	res, err := contractController.service.GetContractByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetContractByIDOutput{Body: *res}, nil
}

// ---------- GetClientContracts ----------

func GetClientContractsHandler(ctx context.Context, input *dto.GetClientContractsInput) (*dto.GetClientContractsOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	res, err := contractController.service.GetClientContracts(clientID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetClientContractsOutput{Body: *res}, nil
}

// ---------- GetFreelancerContracts ----------

func GetFreelancerContractsHandler(ctx context.Context, input *dto.GetFreelancerContractsInput) (*dto.GetFreelancerContractsOutput, error) {
	freelancerID, _ := ctx.Value(middleware.UserIDKey).(string)

	res, err := contractController.service.GetFreelancerContracts(freelancerID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetFreelancerContractsOutput{Body: *res}, nil
}
