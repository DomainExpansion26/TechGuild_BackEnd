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

type PayoutMethodController struct {
	svc *services.PayoutMethodService
}

func NewPayoutMethodController(svc *services.PayoutMethodService) *PayoutMethodController {
	return &PayoutMethodController{svc: svc}
}

// ---------- AddPayoutMethod ----------

func (c *PayoutMethodController) AddPayoutMethodHandler(ctx context.Context, input *dto.AddPayoutMethodInput) (*dto.AddPayoutMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.AddPayoutMethod(userID, input.Body)
	if err != nil {
		return nil, mapPayoutMethodError(err)
	}

	return &dto.AddPayoutMethodOutput{Body: *resp}, nil
}

// ---------- ListPayoutMethods ----------

func (c *PayoutMethodController) ListPayoutMethodsHandler(ctx context.Context, input *dto.ListPayoutMethodsInput) (*dto.ListPayoutMethodsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ListPayoutMethods(userID)
	if err != nil {
		return nil, mapPayoutMethodError(err)
	}

	return &dto.ListPayoutMethodsOutput{Body: *resp}, nil
}

// ---------- SetPrimaryPayoutMethod ----------

func (c *PayoutMethodController) SetPrimaryPayoutMethodHandler(ctx context.Context, input *dto.SetPrimaryPayoutMethodInput) (*dto.SetPrimaryPayoutMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.SetPrimaryPayoutMethod(userID, input.ID)
	if err != nil {
		return nil, mapPayoutMethodError(err)
	}

	return &dto.SetPrimaryPayoutMethodOutput{Body: *resp}, nil
}

// ---------- DeletePayoutMethod ----------

func (c *PayoutMethodController) DeletePayoutMethodHandler(ctx context.Context, input *dto.DeletePayoutMethodInput) (*dto.DeletePayoutMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.svc.DeletePayoutMethod(userID, input.ID); err != nil {
		return nil, mapPayoutMethodError(err)
	}

	return &dto.DeletePayoutMethodOutput{
		Body: dto.DeletePayoutMethodResponse{
			Message: "payout method deleted successfully",
		},
	}, nil
}

// ---------- shared error mapper ----------

func mapPayoutMethodError(err error) error {
	switch {
	case errors.Is(err, services.ErrPayoutMethodNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return huma.Error404NotFound("payout method not found")

	case errors.Is(err, services.ErrPayoutMethodDuplicate):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, services.ErrPayoutValidation):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrPayoutNotVerified):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrPaymentProviderUnsupported):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrPaymentProviderFailed):
		return huma.Error502BadGateway(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
