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

	// ============================================================
	// FREELANCER — Milestone lifecycle
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "create-quest-milestone",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones",
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Create a milestone",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateQuestMilestoneBody{
						Title:       "Research & Wireframes",
						Description: "User research and low-fidelity wireframes",
						Amount:      25000,
						DueDays:     7,
						Tasks: []dto.MilestoneTaskInput{
							{
								Title:       "Navbar Wireframe",
								Description: "Design responsive navigation component",
								AssigneeID:  ptr("a7f3c2e1-9b4d-4f6a-8e2c-1d5b7a9c3e0f"),
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
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Update a milestone (draft/revision)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateQuestMilestoneBody{
						Title:       ptr("Research & Wireframes"),
						Description: ptr("Updated scope after kickoff call"),
						Amount:      ptr(28000.0),
						DueDays:     ptr(10),
						Tasks: []dto.MilestoneTaskPatchInput{
							{
								ID:          ptr("c1e4a9d2-7f3b-4a8c-9d6e-2b5f8c1a4d70"),
								Title:       ptr("Navbar Wireframe v2"),
								Description: ptr("Add mobile breakpoints"),
								AssigneeID:  ptr("a7f3c2e1-9b4d-4f6a-8e2c-1d5b7a9c3e0f"),
								Status:      ptr("in_progress"),
							},
						},
						DeleteTaskIDs: []string{"f2b8d1c4-6a9e-4b3d-8c7f-0e5a2d9b6c13"},
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
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Delete a milestone (draft only)",
		Middlewares: authMw,
	}, c.DeleteMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "send-milestone-for-review",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/send-for-review",
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Send milestone for client review",
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
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Resubmit milestone after revision",
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
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Approve milestone",
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
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Request milestone revision",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RequestMilestoneRevisionBody{
						Note:          "Please adjust the color scheme to match brand guidelines",
						AttachmentURL: "https://cdn.techguild.in/reviews/milestone-1-revision-notes.pdf",
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
		Tags:        []string{"Common / Milestones Read"},
		Summary:     "[Common] Get milestone detail (client or freelancer)",
		Middlewares: authMw,
	}, c.GetMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-quest-milestones",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones",
		Tags:        []string{"Common / Milestones Read"},
		Summary:     "[Common] List all milestones for a quest",
		Middlewares: authMw,
	}, c.ListMilestonesHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-milestone-revisions",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/revisions",
		Tags:        []string{"Common / Milestones Read"},
		Summary:     "[Common] Get revision history for a milestone",
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
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Submit all draft milestones for review",
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
		Tags:        []string{"Client / Milestones & Escrow"},
		Summary:     "[Client] Approve the entire milestone roadmap",
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
