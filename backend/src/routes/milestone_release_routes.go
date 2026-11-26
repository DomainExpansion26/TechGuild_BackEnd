package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterMilestoneReleaseRoutes(
	api huma.API,
	c *controllers.MilestoneReleaseController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}
	tags := []string{"Milestone Release"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "submit-milestone-for-release",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/submit-for-release",
		Tags:        tags,
		Summary:     "Freelancer submits completed milestone for client release",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SubmitForReleaseBody{
						Note: "All tasks completed and tested",
					},
				},
			},
		},
	}, c.SubmitForReleaseHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "release-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/release",
		Tags:        tags,
		Summary:     "Client releases escrow to freelancer wallet",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ReleaseMilestoneBody{
						IdempotencyKey: "release-ms-42-xyz",
						Note:           "Great work, releasing payment",
					},
				},
			},
		},
	}, c.ReleaseMilestoneHandler)
}
