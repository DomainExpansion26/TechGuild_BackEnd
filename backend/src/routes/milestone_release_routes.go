package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterMilestoneReleaseRoutes(
	api huma.API,
	c *controllers.MilestoneReleaseController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}
	tags := []string{"Milestone Release"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "submit-milestone-for-release",
		Method:      "POST",
		Path:        "/v1/quests/{quest_id}/milestones/{milestone_id}/submit-for-release",
		Tags:        tags,
		Summary:     "Freelancer submits completed milestone for client release",
		Middlewares: authMw,
	}, c.SubmitForReleaseHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "release-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{quest_id}/milestones/{milestone_id}/release",
		Tags:        tags,
		Summary:     "Client releases escrow to freelancer wallet",
		Middlewares: authMw,
	}, c.ReleaseMilestoneHandler)
}
