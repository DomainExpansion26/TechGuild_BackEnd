package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterMilestoneFundingRoutes(
	api huma.API,
	c *controllers.MilestoneFundingController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "fund-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/fund",
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Fund a milestone (holds funds in escrow)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.FundMilestoneBody{
						PaymentMethodID: "550e8400-e29b-41d4-a716-446655440000",
						IdempotencyKey:  "fund-ms-42-abc123",
					},
				},
			},
		},
	}, c.FundMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "confirm-milestone-funding",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/fund/confirm",
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Confirm Razorpay checkout",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ConfirmFundingBody{
						RazorpayPaymentID: "pay_NqX8kL2mZ4vB1c",
						RazorpayOrderID:   "order_NqX7tR5wY3uA9p",
						RazorpaySignature: "9f2c1a4b7e8d3f6a0b5c9e2d4f7a1b8c3e6d9f2a5b8c1e4d7f0a3b6c9e2d5f8a",
					},
				},
			},
		},
	}, c.ConfirmFundingHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-milestone-payment",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/payment",
		Tags:        []string{"Common / Milestones Read"},
		Summary:     "[Common] Get payment details for a milestone",
		Middlewares: authMw,
	}, c.GetMilestonePaymentHandler)
}
