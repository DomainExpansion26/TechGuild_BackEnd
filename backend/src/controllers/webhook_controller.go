package controllers

import (
	"context"
	"encoding/json"
	"errors"

	"techguild-backend/src/services"

	"github.com/danielgtaylor/huma/v2"
)

type WebhookController struct {
	svc *services.WebhookService
}

func NewWebhookController(svc *services.WebhookService) *WebhookController {
	return &WebhookController{svc: svc}
}

// ---------- Razorpay webhook ----------

func (c *WebhookController) RazorpayWebhookHandler(ctx context.Context, input *struct {
	Body      json.RawMessage `contentType:"application/json"`
	Signature string          `header:"X-Razorpay-Signature"`
}) (*struct{}, error) {
	payload := []byte(input.Body)

	if err := c.svc.ProcessWebhook("razorpay", payload, input.Signature); err != nil {
		switch {
		case errors.Is(err, services.ErrWebhookUnknownProvider):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrWebhookInvalidSig):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrWebhookMalformed):
			return nil, huma.Error400BadRequest(err.Error())
		default:
			return nil, huma.Error500InternalServerError(err.Error())
		}
	}
	return &struct{}{}, nil
}
