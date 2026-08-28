package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- ApplyProject ----------

func ApplyProjectHandler(ctx context.Context, input *dto.ApplyProjectInput) (*dto.ApplyProjectOutput, error) {
	applicantID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectApplicationService := services.NewProjectApplicationService()
	res, err := projectApplicationService.ApplyProject(applicantID, input.ProjectID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ApplyProjectOutput{Body: *res}, nil
}

// ---------- WithdrawApplication ----------

func WithdrawApplicationHandler(ctx context.Context, input *dto.WithdrawApplicationInput) (*dto.WithdrawApplicationOutput, error) {
	applicantID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectApplicationService := services.NewProjectApplicationService()
	if err := projectApplicationService.WithdrawApplication(applicantID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.WithdrawApplicationOutput{
		Body: dto.WithdrawApplicationResponse{Message: "Application withdrawn successfully"},
	}, nil
}

// ---------- AcceptApplication ----------

func AcceptApplicationHandler(ctx context.Context, input *dto.AcceptApplicationInput) (*dto.AcceptApplicationOutput, error) {
	projectApplicationService := services.NewProjectApplicationService()
	if err := projectApplicationService.AcceptApplication(input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AcceptApplicationOutput{
		Body: dto.AcceptApplicationResponse{Message: "Application accepted successfully"},
	}, nil
}

// ---------- RejectApplication ----------

func RejectApplicationHandler(ctx context.Context, input *dto.RejectApplicationInput) (*dto.RejectApplicationOutput, error) {
	projectApplicationService := services.NewProjectApplicationService()
	if err := projectApplicationService.RejectApplication(input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectApplicationOutput{
		Body: dto.RejectApplicationResponse{Message: "Application rejected successfully"},
	}, nil
}

// ---------- ShortlistApplication ----------

func ShortlistApplicationHandler(ctx context.Context, input *dto.ShortlistApplicationInput) (*dto.ShortlistApplicationOutput, error) {
	projectApplicationService := services.NewProjectApplicationService()
	if err := projectApplicationService.ShortlistApplication(input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ShortlistApplicationOutput{
		Body: dto.ShortlistApplicationResponse{Message: "Application shortlisted successfully"},
	}, nil
}

// ---------- GetMyApplications ----------

func GetMyApplicationsHandler(ctx context.Context, input *dto.GetMyApplicationsInput) (*dto.GetMyApplicationsOutput, error) {
	applicantID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectApplicationService := services.NewProjectApplicationService()
	res, err := projectApplicationService.GetMyApplications(applicantID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetMyApplicationsOutput{Body: *res}, nil
}

// ---------- GetProjectApplications ----------

func GetProjectApplicationsHandler(ctx context.Context, input *dto.GetProjectApplicationsInput) (*dto.GetProjectApplicationsOutput, error) {
	projectApplicationService := services.NewProjectApplicationService()
	res, err := projectApplicationService.GetProjectApplications(input.ProjectID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetProjectApplicationsOutput{Body: *res}, nil
}
