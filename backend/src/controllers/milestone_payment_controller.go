package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

type MilestonePaymentController struct {
	svc *services.MilestonePaymentService
}

func NewMilestonePaymentController(svc *services.MilestonePaymentService) *MilestonePaymentController {
	return &MilestonePaymentController{svc: svc}
}

// ============================================================
// Fund Milestone
// ============================================================

func (c *MilestonePaymentController) FundMilestoneHandler(
	ctx context.Context,
	input *dto.FundMilestoneInput,
) (*dto.FundMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.FundMilestone(
		userID,
		input.ProjectID,
		input.MilestoneID,
		input.Body,
	)
	if err != nil {
		return nil, mapMilestonePaymentError(err)
	}

	return &dto.FundMilestoneOutput{Body: *resp}, nil
}

// ============================================================
// Release Milestone
// ============================================================

func (c *MilestonePaymentController) ReleaseMilestonePaymentHandler(
	ctx context.Context,
	input *dto.ReleaseMilestonePaymentInput,
) (*dto.ReleaseMilestonePaymentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ReleaseMilestonePayment(
		userID,
		input.ProjectID,
		input.MilestoneID,
		input.Body,
	)
	if err != nil {
		return nil, mapMilestonePaymentError(err)
	}

	return &dto.ReleaseMilestonePaymentOutput{Body: *resp}, nil
}

// ============================================================
// Get Milestone Payment
// ============================================================

func (c *MilestonePaymentController) GetMilestonePaymentHandler(
	ctx context.Context,
	input *dto.GetMilestonePaymentInput,
) (*dto.GetMilestonePaymentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.GetMilestonePayment(
		userID,
		input.ProjectID,
		input.MilestoneID,
	)
	if err != nil {
		return nil, mapMilestonePaymentError(err)
	}

	return &dto.GetMilestonePaymentOutput{Body: *resp}, nil
}

// ============================================================
// Error mapper
// ============================================================

func mapMilestonePaymentError(err error) error {
	switch {
	case errors.Is(err, services.ErrMilestoneNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return huma.Error404NotFound("milestone not found")

	case errors.Is(err, services.ErrMilestoneNotFunded):
		return huma.Error400BadRequest("milestone is not funded")

	case errors.Is(err, services.ErrMilestoneAlreadyFunded):
		return huma.Error409Conflict("milestone already funded")

	case errors.Is(err, services.ErrMilestoneAlreadyReleased):
		return huma.Error409Conflict("milestone already released")

	case errors.Is(err, services.ErrUnauthorizedMilestone):
		return huma.Error403Forbidden("not authorized for this milestone")

	case errors.Is(err, services.ErrContractInactive):
		return huma.Error400BadRequest("contract is not active")

	case errors.Is(err, services.ErrFundingFailed):
		return huma.Error502BadGateway("milestone funding failed at provider")

	case errors.Is(err, services.ErrProviderUnsupported):
		return huma.Error400BadRequest("operation not supported")

	case errors.Is(err, services.ErrProviderUnavailable):
		return huma.Error502BadGateway("provider unavailable")

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
