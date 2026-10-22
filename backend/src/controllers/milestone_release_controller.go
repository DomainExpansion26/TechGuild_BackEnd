package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

type MilestoneReleaseController struct {
	svc *services.MilestoneReleaseService
}

func NewMilestoneReleaseController(cfg *config.Config, provider services.PaymentProvider) *MilestoneReleaseController {
	return &MilestoneReleaseController{
		svc: services.NewMilestoneReleaseService(cfg, provider),
	}
}

// ============================================================
// Submit for Release (freelancer)
// ============================================================

func (c *MilestoneReleaseController) SubmitForReleaseHandler(
	ctx context.Context,
	input *dto.SubmitForReleaseInput,
) (*dto.SubmitForReleaseOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.SubmitForRelease(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapReleaseError(err)
	}
	return &dto.SubmitForReleaseOutput{Body: *resp}, nil
}

// ============================================================
// Release Milestone (client)
// ============================================================

func (c *MilestoneReleaseController) ReleaseMilestoneHandler(
	ctx context.Context,
	input *dto.ReleaseMilestoneInput,
) (*dto.ReleaseMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ReleaseMilestone(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapReleaseError(err)
	}
	return &dto.ReleaseMilestoneOutput{Body: *resp}, nil
}

// ============================================================
// Error mapper
// ============================================================

func mapReleaseError(err error) error {
	switch {
	case errors.Is(err, services.ErrRelQuestNotFound),
		errors.Is(err, services.ErrRelMilestoneNotFound),
		errors.Is(err, services.ErrRelPaymentNotFound):
		return huma.Error404NotFound(err.Error())

	case errors.Is(err, services.ErrRelUnauthorized),
		errors.Is(err, services.ErrRelNotAssignedFreelancer),
		errors.Is(err, services.ErrRelNotQuestClient):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, services.ErrRelInvalidState):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrRelAlreadyReleased):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, services.ErrRelProviderFailed),
		errors.Is(err, services.ErrProviderUnavailable):
		return huma.Error502BadGateway(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
