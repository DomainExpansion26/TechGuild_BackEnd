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

type RefundController struct {
	svc *services.RefundService
}

func NewRefundController(cfg *config.Config, provider services.PaymentProvider) *RefundController {
	return &RefundController{
		svc: services.NewRefundService(cfg, provider),
	}
}

func (c *RefundController) RequestRefundHandler(
	ctx context.Context,
	input *dto.RequestRefundInput,
) (*dto.RequestRefundOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.RequestRefund(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapRefundError(err)
	}
	return &dto.RequestRefundOutput{Body: *resp}, nil
}

func (c *RefundController) GetRefundHandler(
	ctx context.Context,
	input *dto.GetRefundInput,
) (*dto.GetRefundOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.GetRefund(userID, input.QuestID, input.MilestoneID)
	if err != nil {
		return nil, mapRefundError(err)
	}
	return &dto.GetRefundOutput{Body: *resp}, nil
}

func mapRefundError(err error) error {
	switch {
	case errors.Is(err, services.ErrRefQuestNotFound),
		errors.Is(err, services.ErrRefMilestoneNotFound),
		errors.Is(err, services.ErrRefPaymentNotFound),
		errors.Is(err, services.ErrRefEscrowNotFound),
		errors.Is(err, services.ErrRefNotFound):
		return huma.Error404NotFound(err.Error())

	case errors.Is(err, services.ErrRefUnauthorized):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, services.ErrRefInvalidState),
		errors.Is(err, services.ErrRefReasonRequired):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrRefAlreadyRefunded):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, services.ErrRefProviderFailed),
		errors.Is(err, services.ErrProviderUnavailable):
		return huma.Error502BadGateway(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
