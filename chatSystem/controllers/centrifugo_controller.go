package controllers

import (
	"context"

	"techguild-chat/config"
	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
)

type CentrifugoController struct {
	cfg               *config.Config
	centrifugoService services.CentrifugoService
}

func NewCentrifugoController(cfg *config.Config, centrifugoService services.CentrifugoService) *CentrifugoController {
	return &CentrifugoController{
		cfg:               cfg,
		centrifugoService: centrifugoService,
	}
}

// GetCentrifugoTokenHandler handles GET /v1/communication/token
func (c *CentrifugoController) GetCentrifugoTokenHandler(ctx context.Context, input *dto.GetCentrifugoTokenInput) (*dto.GetCentrifugoTokenOutput, error) {
	userIDStr, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	token, exp, err := c.centrifugoService.GenerateConnectionToken(userIDStr)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate Centrifugo token: " + err.Error())
	}

	return &dto.GetCentrifugoTokenOutput{
		Body: dto.CentrifugoTokenResponse{
			Token:     token,
			ExpiresAt: exp,
			WSURL:     c.cfg.CentrifugoWSURL,
		},
	}, nil
}

// CentrifugoSubscribeProxyHandler handles POST /v1/communication/centrifugo/subscribe-proxy
func (c *CentrifugoController) CentrifugoSubscribeProxyHandler(ctx context.Context, input *dto.CentrifugoSubscribeProxyInput) (*dto.CentrifugoSubscribeProxyOutput, error) {
	req := input.Body
	if req.User == "" || req.Channel == "" {
		return &dto.CentrifugoSubscribeProxyOutput{
			Body: dto.CentrifugoSubscribeProxyResponse{
				Error: &dto.CentrifugoProxyError{
					Code:    1001,
					Message: "Bad request: missing user or channel",
				},
			},
		}, nil
	}

	res, err := c.centrifugoService.CheckSubscribePermission(ctx, req.User, req.Channel)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CentrifugoSubscribeProxyOutput{Body: *res}, nil
}

