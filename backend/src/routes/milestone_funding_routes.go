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
	tags := []string{"Milestone Funding"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "fund-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/fund",
		Tags:        tags,
		Summary:     "Fund a milestone (client). Holds funds in escrow.",
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
		Tags:        tags,
		Summary:     "Confirm Razorpay checkout (signature verification)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ConfirmFundingBody{
						RazorpayPaymentID: "pay_xxx",
						RazorpayOrderID:   "order_xxx",
						RazorpaySignature: "<hmac>",
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
		Tags:        tags,
		Summary:     "Get payment details for a milestone",
		Middlewares: authMw,
	}, c.GetMilestonePaymentHandler)
}
