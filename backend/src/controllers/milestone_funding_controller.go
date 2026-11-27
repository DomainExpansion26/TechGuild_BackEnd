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

type MilestoneFundingController struct {
	svc *services.MilestoneFundingService
}

func NewMilestoneFundingController(cfg *config.Config, provider services.PaymentProvider) *MilestoneFundingController {
	return &MilestoneFundingController{
		svc: services.NewMilestoneFundingService(cfg, provider),
	}
}

func (c *MilestoneFundingController) FundMilestoneHandler(
	ctx context.Context,
	input *dto.FundMilestoneInput,
) (*dto.FundMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.FundMilestone(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapFundingError(err)
	}

	populateMinorTwins(resp)
	return &dto.FundMilestoneOutput{Body: *resp}, nil
}

func (c *MilestoneFundingController) ConfirmFundingHandler(
	ctx context.Context,
	input *dto.ConfirmFundingInput,
) (*dto.ConfirmFundingOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ConfirmFunding(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapFundingError(err)
	}

	populateMinorTwins(resp)
	return &dto.ConfirmFundingOutput{Body: *resp}, nil
}

func (c *MilestoneFundingController) GetMilestonePaymentHandler(
	ctx context.Context,
	input *dto.GetMilestonePaymentInput,
) (*dto.GetMilestonePaymentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.GetMilestonePayment(userID, input.QuestID, input.MilestoneID)
	if err != nil {
		return nil, mapFundingError(err)
	}

	return &dto.GetMilestonePaymentOutput{Body: *resp}, nil
}

// populateMinorTwins fills the minor-unit twins of the rupee amounts on
// FundMilestoneResponse. The funding service populates the rupee fields for
// every variant (mock and Razorpay), so the paise values are derived here with
// the same conversion the service uses in the opposite direction. The three
// minor fields that the service already sets (milestone amount, client fee,
// client charged) are left untouched.
func populateMinorTwins(resp *dto.FundMilestoneResponse) {
	resp.FreelancerFeeMinor = utils.RupeesToPaise(resp.FreelancerFee)
	resp.FreelancerNetMinor = utils.RupeesToPaise(resp.FreelancerNet)
	resp.PlatformTotalMinor = utils.RupeesToPaise(resp.PlatformTotal)
}

func mapFundingError(err error) error {
	switch {
	case errors.Is(err, services.ErrFundQuestNotFound),
		errors.Is(err, services.ErrFundMilestoneNotFound),
		errors.Is(err, services.ErrFundPaymentNotFound):
		return huma.Error404NotFound(err.Error())

	case errors.Is(err, services.ErrFundUnauthorized):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, services.ErrFundMilestoneNotAgreed),
		errors.Is(err, services.ErrFundQuestNotInProgress),
		errors.Is(err, services.ErrFundNoPaymentMethod),
		errors.Is(err, services.ErrFundNotPending),
		errors.Is(err, services.ErrFundPaymentMismatch),
		errors.Is(err, services.ErrFundOrderMismatch):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrFundInvalidSignature):
		return huma.Error401Unauthorized(err.Error())

	case errors.Is(err, services.ErrFundAlreadyFunded):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, services.ErrFundProviderFailed),
		errors.Is(err, services.ErrFundProviderVerifyFailed),
		errors.Is(err, services.ErrProviderUnavailable):
		return huma.Error502BadGateway(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}

func NewMilestoneFundingControllerFromService(svc *services.MilestoneFundingService) *MilestoneFundingController {
	return &MilestoneFundingController{svc: svc}
}
