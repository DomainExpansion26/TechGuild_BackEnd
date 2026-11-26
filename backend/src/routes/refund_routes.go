package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterRefundRoutes(api huma.API, c *controllers.RefundController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}
	tags := []string{"Refunds"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-milestone-refund",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/refund",
		Tags:        tags,
		Summary:     "Request refund for a milestone (client only)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RequestRefundBody{
						Reason:         "Freelancer did not deliver as agreed",
						IdempotencyKey: "refund-ms-f4aa-001",
					},
				},
			},
		},
	}, c.RequestRefundHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-milestone-refund",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/refund",
		Tags:        tags,
		Summary:     "Get refund details for a milestone",
		Middlewares: authMw,
	}, c.GetRefundHandler)
}
