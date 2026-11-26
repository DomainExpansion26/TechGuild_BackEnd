package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterQuestMilestoneRoutes(
	api huma.API,
	c *controllers.QuestMilestoneController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}
	tags := []string{"Quest Milestones"}

	// ============================================================
	// FREELANCER — Milestone lifecycle
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "create-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones",
		Tags:        tags,
		Summary:     "Create a milestone (freelancer only)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateQuestMilestoneBody{
						Title:       "Research & Wireframes",
						Description: "User research and low-fidelity wireframes",
						Amount:      10000,
						DueDays:     7,
						Tasks: []dto.MilestoneTaskInput{
							{
								Title:       "Navbar Wireframe",
								Description: "Design responsive navigation component",
								AssigneeID:  ptr("11111111-1111-1111-1111-111111111111"),
							},
						},
					},
				},
			},
		},
	}, c.CreateMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "update-quest-milestone",
		Method:      "PATCH",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}",
		Tags:        tags,
		Summary:     "Update a milestone (freelancer, draft/revision only)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateQuestMilestoneBody{
						Title:       ptr("Research & Wireframes"),
						Description: ptr("Updated scope after kickoff call"),
						Amount:      ptr(12000.0),
						DueDays:     ptr(10),
						Tasks: []dto.MilestoneTaskPatchInput{
							{
								ID:          ptr("22222222-2222-2222-2222-222222222222"),
								Title:       ptr("Navbar Wireframe v2"),
								Description: ptr("Add mobile breakpoints"),
								AssigneeID:  ptr("11111111-1111-1111-1111-111111111111"),
								Status:      ptr("in_progress"),
							},
						},
						DeleteTaskIDs: []string{"33333333-3333-3333-3333-333333333333"},
					},
				},
			},
		},
	}, c.UpdateMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "delete-quest-milestone",
		Method:      "DELETE",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}",
		Tags:        tags,
		Summary:     "Delete a milestone (freelancer, draft only)",
		Middlewares: authMw,
	}, c.DeleteMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "send-milestone-for-review",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/send-for-review",
		Tags:        tags,
		Summary:     "Send milestone for client review (freelancer)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SendForReviewBody{
						Note: "Please review the deliverables",
					},
				},
			},
		},
	}, c.SendForReviewHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "resubmit-milestone-after-revision",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/resubmit",
		Tags:        tags,
		Summary:     "Resubmit milestone after revision (freelancer)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SendForReviewBody{
						Note: "Revisions applied as requested. Ready for another review.",
					},
				},
			},
		},
	}, c.ResubmitHandler)

	// ============================================================
	// CLIENT — Review actions
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "approve-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/approve",
		Tags:        tags,
		Summary:     "Approve milestone (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveMilestoneBody{
						Feedback: "Looks great, approved!",
					},
				},
			},
		},
	}, c.ApproveMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-milestone-revision",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/request-revision",
		Tags:        tags,
		Summary:     "Request milestone revision (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RequestMilestoneRevisionBody{
						Note:          "Please adjust the color scheme to match brand guidelines",
						AttachmentURL: "https://example.com/attachments/color-scheme-notes.pdf",
					},
				},
			},
		},
	}, c.RequestRevisionHandler)

	// ============================================================
	// SHARED — Read
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-quest-milestone",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}",
		Tags:        tags,
		Summary:     "Get milestone detail (client or freelancer)",
		Middlewares: authMw,
	}, c.GetMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-quest-milestones",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones",
		Tags:        tags,
		Summary:     "List all milestones for a quest",
		Middlewares: authMw,
	}, c.ListMilestonesHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-milestone-revisions",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/revisions",
		Tags:        tags,
		Summary:     "Get revision history for a milestone",
		Middlewares: authMw,
	}, c.GetRevisionsHandler)

	// ============================================================
	// ROADMAP ACTIONS (Bulk)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "submit-milestone-roadmap",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/submit-roadmap",
		Tags:        tags,
		Summary:     "Submit all draft milestones for client review at once (freelancer)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SubmitRoadmapBody{
						Note: "Here is the complete project roadmap. Please review.",
					},
				},
			},
		},
	}, c.SubmitRoadmapHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "approve-milestone-roadmap",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/approve-roadmap",
		Tags:        tags,
		Summary:     "Approve the entire milestone roadmap (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveRoadmapBody{
						Feedback: "Roadmap looks perfect. Let's start with Milestone 1.",
					},
				},
			},
		},
	}, c.ApproveRoadmapHandler)
}
