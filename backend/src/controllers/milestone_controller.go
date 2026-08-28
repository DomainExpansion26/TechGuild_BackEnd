package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/services"

	"github.com/danielgtaylor/huma/v2"
)

type MilestoneController struct {
	service *services.MilestoneService
}

func NewMilestoneController() *MilestoneController {
	return &MilestoneController{
		service: services.NewMilestoneService(),
	}
}

var milestoneController = NewMilestoneController()

// ---------- CreateMilestone ----------

func CreateMilestoneHandler(ctx context.Context, input *dto.CreateMilestoneInput) (*dto.CreateMilestoneOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	res, err := milestoneController.service.CreateMilestone(clientID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateMilestoneOutput{Body: *res}, nil
}

// ---------- UpdateMilestone ----------

func UpdateMilestoneHandler(ctx context.Context, input *dto.UpdateMilestoneInput) (*dto.UpdateMilestoneOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.UpdateMilestone(clientID, input.ID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateMilestoneOutput{
		Body: dto.UpdateMilestoneResponse{Message: "Milestone updated successfully"},
	}, nil
}

// ---------- DeleteMilestone ----------

func DeleteMilestoneHandler(ctx context.Context, input *dto.DeleteMilestoneInput) (*dto.DeleteMilestoneOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.DeleteMilestone(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteMilestoneOutput{
		Body: dto.DeleteMilestoneResponse{Message: "Milestone deleted successfully"},
	}, nil
}

// ---------- SubmitMilestone ----------

func SubmitMilestoneHandler(ctx context.Context, input *dto.SubmitMilestoneInput) (*dto.SubmitMilestoneOutput, error) {
	freelancerID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.SubmitMilestone(freelancerID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SubmitMilestoneOutput{
		Body: dto.SubmitMilestoneResponse{Message: "Milestone submitted successfully"},
	}, nil
}

// ---------- ApproveMilestone ----------

func ApproveMilestoneHandler(ctx context.Context, input *dto.ApproveMilestoneInput) (*dto.ApproveMilestoneOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.ApproveMilestone(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ApproveMilestoneOutput{
		Body: dto.ApproveMilestoneResponse{Message: "Milestone approved successfully"},
	}, nil
}

// ---------- RejectMilestone ----------

func RejectMilestoneHandler(ctx context.Context, input *dto.RejectMilestoneInput) (*dto.RejectMilestoneOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.RejectMilestone(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectMilestoneOutput{
		Body: dto.RejectMilestoneResponse{Message: "Milestone rejected successfully"},
	}, nil
}

// ---------- MarkMilestonePaid ----------

func MarkMilestonePaidHandler(ctx context.Context, input *dto.MarkMilestonePaidInput) (*dto.MarkMilestonePaidOutput, error) {
	clientID, _ := ctx.Value(middleware.UserIDKey).(string)

	if err := milestoneController.service.MarkMilestonePaid(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.MarkMilestonePaidOutput{
		Body: dto.MarkMilestonePaidResponse{Message: "Milestone marked as paid successfully"},
	}, nil
}

// ---------- GetMilestoneByID ----------

func GetMilestoneByIDHandler(ctx context.Context, input *dto.GetMilestoneByIDInput) (*dto.GetMilestoneByIDOutput, error) {
	res, err := milestoneController.service.GetMilestoneByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetMilestoneByIDOutput{Body: *res}, nil
}

// ---------- GetContractMilestones ----------

func GetContractMilestonesHandler(ctx context.Context, input *dto.GetContractMilestonesInput) (*dto.GetContractMilestonesOutput, error) {
	res, err := milestoneController.service.GetContractMilestones(input.ContractID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetContractMilestonesOutput{Body: *res}, nil
}
