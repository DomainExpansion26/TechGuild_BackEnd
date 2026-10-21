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

type WalletWithdrawalController struct {
	svc *services.WalletWithdrawalService
}

func NewWalletWithdrawalController(cfg *config.Config, provider services.PaymentProvider) *WalletWithdrawalController {
	return &WalletWithdrawalController{
		svc: services.NewWalletWithdrawalService(cfg, provider),
	}
}

func (c *WalletWithdrawalController) RequestWithdrawalHandler(
	ctx context.Context,
	input *dto.RequestWithdrawalInput,
) (*dto.RequestWithdrawalOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	resp, err := c.svc.RequestWithdrawal(userID, input.Body)
	if err != nil {
		return nil, mapWithdrawalError(err)
	}
	return &dto.RequestWithdrawalOutput{Body: *resp}, nil
}

func (c *WalletWithdrawalController) ListWithdrawalsHandler(
	ctx context.Context,
	input *dto.ListWithdrawalsInput,
) (*dto.ListWithdrawalsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	resp, err := c.svc.ListWithdrawals(userID, input.Limit, input.Offset)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return &dto.ListWithdrawalsOutput{Body: *resp}, nil
}

func (c *WalletWithdrawalController) GetWithdrawalHandler(
	ctx context.Context,
	input *dto.GetWithdrawalInput,
) (*dto.GetWithdrawalOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	resp, err := c.svc.GetWithdrawal(userID, input.PayoutID)
	if err != nil {
		return nil, mapWithdrawalError(err)
	}
	return &dto.GetWithdrawalOutput{Body: *resp}, nil
}

func mapWithdrawalError(err error) error {
	switch {
	case errors.Is(err, services.ErrWDNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, services.ErrWDWalletNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, services.ErrWDInsufficient):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, services.ErrWDInvalidAmount):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, services.ErrWDPayoutMethodError):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, services.ErrWDProviderFailed):
		return huma.Error502BadGateway(err.Error())
	case errors.Is(err, services.ErrWDDuplicate):
		return huma.Error409Conflict(err.Error())
	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
