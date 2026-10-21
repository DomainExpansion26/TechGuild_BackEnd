package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterRefundRoutes(api huma.API, c *controllers.RefundController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}
	tags := []string{"Refunds"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-milestone-refund",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/refund",
		Tags:        tags,
		Summary:     "Request refund for a milestone (client only)",
		Middlewares: authMw,
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
