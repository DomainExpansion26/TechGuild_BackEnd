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

type PaymentMethodController struct {
	svc *services.PaymentMethodService
}

func NewPaymentMethodController(svc *services.PaymentMethodService) *PaymentMethodController {
	return &PaymentMethodController{svc: svc}
}

// ---------- AddClientPaymentMethod ----------

func (c *PaymentMethodController) AddClientPaymentMethodHandler(ctx context.Context, input *dto.AddClientPaymentMethodInput) (*dto.AddClientPaymentMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.AddClientPaymentMethod(userID, input.Body)
	if err != nil {
		return nil, mapPaymentMethodError(err)
	}

	return &dto.AddClientPaymentMethodOutput{Body: *resp}, nil
}

// ---------- ListClientPaymentMethods ----------

func (c *PaymentMethodController) ListClientPaymentMethodsHandler(ctx context.Context, input *dto.ListClientPaymentMethodsInput) (*dto.ListClientPaymentMethodsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ListClientPaymentMethods(userID)
	if err != nil {
		return nil, mapPaymentMethodError(err)
	}

	return &dto.ListClientPaymentMethodsOutput{Body: *resp}, nil
}

// ---------- SetPrimaryClientPaymentMethod ----------

func (c *PaymentMethodController) SetPrimaryClientPaymentMethodHandler(ctx context.Context, input *dto.SetPrimaryClientPaymentMethodInput) (*dto.SetPrimaryClientPaymentMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.SetPrimaryClientPaymentMethod(userID, input.ID)
	if err != nil {
		return nil, mapPaymentMethodError(err)
	}

	return &dto.SetPrimaryClientPaymentMethodOutput{Body: *resp}, nil
}

// ---------- DeleteClientPaymentMethod ----------

func (c *PaymentMethodController) DeleteClientPaymentMethodHandler(ctx context.Context, input *dto.DeleteClientPaymentMethodInput) (*dto.DeleteClientPaymentMethodOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.svc.DeleteClientPaymentMethod(userID, input.ID); err != nil {
		return nil, mapPaymentMethodError(err)
	}

	return &dto.DeleteClientPaymentMethodOutput{
		Body: dto.DeleteClientPaymentMethodResponse{
			Message: "payment method deleted successfully",
		},
	}, nil
}

// ---------- shared error mapper ----------

func mapPaymentMethodError(err error) error {
	switch {
	case errors.Is(err, services.ErrPaymentMethodNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return huma.Error404NotFound("payment method not found")

	case errors.Is(err, services.ErrPaymentMethodDuplicate):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, services.ErrPaymentMethodForbidden):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, services.ErrPaymentProviderUnsupported):
		return huma.Error400BadRequest(err.Error())

	case errors.Is(err, services.ErrPaymentProviderFailed):
		return huma.Error502BadGateway(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
