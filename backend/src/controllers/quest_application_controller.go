package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

func ApplyQuestHandler(ctx context.Context, input *dto.ApplyQuestInput) (*dto.ApplyQuestOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().ApplyToQuest(userID, input.ID, input.Body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestNotFound), errors.Is(err, services.ErrApplicationNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrFreelancerOnly),
			errors.Is(err, services.ErrCannotApplyOwnQuest),
			errors.Is(err, services.ErrFreelancerRankInsufficient),
			errors.Is(err, services.ErrFreelancerNotVerified):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrAlreadyApplied),
			errors.Is(err, services.ErrQuestNotOpenForApplications):
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.ApplyQuestOutput{Status: http.StatusCreated, Body: *res}, nil
}

func ListApplicationsHandler(ctx context.Context, input *dto.ListApplicationsInput) (*dto.ListApplicationsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().ListApplications(userID, input.ID, input.Status)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.ListApplicationsOutput{Body: *res}, nil
}

func AcceptApplicationHandler(ctx context.Context, input *dto.AcceptApplicationInput) (*dto.AcceptApplicationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().AcceptApplication(userID, input.ID, input.ApplicationID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestNotFound), errors.Is(err, services.ErrApplicationNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrOnlyClientCanReview):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrApplicationNotPending),
			errors.Is(err, services.ErrQuestNotOpenForApplications),
			errors.Is(err, services.ErrEscrowNotFunded):
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.AcceptApplicationOutput{Status: http.StatusOK, Body: *res}, nil
}

func RejectApplicationHandler(ctx context.Context, input *dto.RejectApplicationInput) (*dto.RejectApplicationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().RejectApplication(userID, input.ID, input.ApplicationID, input.Body.Reason)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestNotFound), errors.Is(err, services.ErrApplicationNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrOnlyClientCanReview):
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.RejectApplicationOutput{Body: *res}, nil
}

func WithdrawApplicationHandler(ctx context.Context, input *dto.WithdrawApplicationInput) (*dto.WithdrawApplicationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().WithdrawApplication(userID, input.ID, input.ApplicationID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrApplicationNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrNotApplicationOwner):
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.WithdrawApplicationOutput{Body: *res}, nil
}

func MyWorkHandler(ctx context.Context, input *dto.MyWorkInput) (*dto.MyWorkOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	res, err := services.NewQuestApplicationService().MyWork(userID, input.Status)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.MyWorkOutput{Body: *res}, nil
}
