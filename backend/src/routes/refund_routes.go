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

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-milestone-refund",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/refund",
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Request refund for a milestone",
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
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Get refund details",
		Middlewares: authMw,
	}, c.GetRefundHandler)
}
