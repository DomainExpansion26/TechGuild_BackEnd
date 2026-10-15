package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterMilestonePaymentRoutes(
	api huma.API,
	c *controllers.MilestonePaymentController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}

	// ============================================================
	// Fund Milestone — client funds a milestone (money IN)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "fund-milestone",
		Method:      "POST",
		Path:        "/v1/projects/{project_id}/milestones/{milestone_id}/fund",
		Tags:        []string{"Milestone Payments"},
		Summary:     "Fund a milestone (money held in escrow)",
		Middlewares: authMw,
	}, c.FundMilestoneHandler)

	// ============================================================
	// Release Milestone — client approves + escrow releases
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "release-milestone-payment",
		Method:      "POST",
		Path:        "/v1/projects/{project_id}/milestones/{milestone_id}/release",
		Tags:        []string{"Milestone Payments"},
		Summary:     "Release escrow for an approved milestone (initiates payout)",
		Middlewares: authMw,
	}, c.ReleaseMilestonePaymentHandler)

	// ============================================================
	// Read Milestone Payment
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-milestone-payment",
		Method:      "GET",
		Path:        "/v1/projects/{project_id}/milestones/{milestone_id}/payment",
		Tags:        []string{"Milestone Payments"},
		Summary:     "Fetch the payment details for a milestone",
		Middlewares: authMw,
	}, c.GetMilestonePaymentHandler)
}
