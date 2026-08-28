package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- CreateSubmission ----------

func CreateSubmissionHandler(ctx context.Context, input *dto.CreateSubmissionInput) (*dto.CreateSubmissionOutput, error) {
	freelancerID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	submissionService := services.NewSubmissionService()
	res, err := submissionService.CreateSubmission(freelancerID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateSubmissionOutput{Body: *res}, nil
}

// ---------- UpdateSubmission ----------

func UpdateSubmissionHandler(ctx context.Context, input *dto.UpdateSubmissionInput) (*dto.UpdateSubmissionOutput, error) {
	freelancerID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	submissionService := services.NewSubmissionService()
	if err := submissionService.UpdateSubmission(freelancerID, input.ID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateSubmissionOutput{
		Body: dto.UpdateSubmissionResponse{Message: "Submission updated successfully"},
	}, nil
}

// ---------- DeleteSubmission ----------

func DeleteSubmissionHandler(ctx context.Context, input *dto.DeleteSubmissionInput) (*dto.DeleteSubmissionOutput, error) {
	freelancerID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	submissionService := services.NewSubmissionService()
	if err := submissionService.DeleteSubmission(freelancerID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteSubmissionOutput{
		Body: dto.DeleteSubmissionResponse{Message: "Submission deleted successfully"},
	}, nil
}

// ---------- ApproveSubmission ----------

func ApproveSubmissionHandler(ctx context.Context, input *dto.ApproveSubmissionInput) (*dto.ApproveSubmissionOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	submissionService := services.NewSubmissionService()
	if err := submissionService.ApproveSubmission(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ApproveSubmissionOutput{
		Body: dto.ReviewSubmissionResponse{Message: "Submission approved successfully"},
	}, nil
}

// ---------- RejectSubmission ----------

func RejectSubmissionHandler(ctx context.Context, input *dto.RejectSubmissionInput) (*dto.RejectSubmissionOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	submissionService := services.NewSubmissionService()
	if err := submissionService.RejectSubmission(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectSubmissionOutput{
		Body: dto.ReviewSubmissionResponse{Message: "Submission rejected successfully"},
	}, nil
}

// ---------- GetSubmissionByID ----------

func GetSubmissionByIDHandler(ctx context.Context, input *dto.GetSubmissionByIDInput) (*dto.GetSubmissionByIDOutput, error) {
	submissionService := services.NewSubmissionService()
	res, err := submissionService.GetSubmissionByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetSubmissionByIDOutput{Body: *res}, nil
}

// ---------- GetMilestoneSubmissions ----------

func GetMilestoneSubmissionsHandler(ctx context.Context, input *dto.GetMilestoneSubmissionsInput) (*dto.GetMilestoneSubmissionsOutput, error) {
	submissionService := services.NewSubmissionService()
	res, err := submissionService.GetMilestoneSubmissions(input.MilestoneID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetMilestoneSubmissionsOutput{Body: *res}, nil
}
