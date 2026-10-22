package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterQuestMilestoneRoutes(
	api huma.API,
	c *controllers.QuestMilestoneController,
) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}
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
	}, c.CreateMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "update-quest-milestone",
		Method:      "PATCH",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}",
		Tags:        tags,
		Summary:     "Update a milestone (freelancer, draft/revision only)",
		Middlewares: authMw,
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
	}, c.SendForReviewHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "resubmit-milestone-after-revision",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/resubmit",
		Tags:        tags,
		Summary:     "Resubmit milestone after revision (freelancer)",
		Middlewares: authMw,
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
	}, c.ApproveMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-milestone-revision",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/request-revision",
		Tags:        tags,
		Summary:     "Request milestone revision (client)",
		Middlewares: authMw,
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
	}, c.SubmitRoadmapHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "approve-milestone-roadmap",
		Method:      "POST",
		Path:        "/v1/quests/{id}/milestones/approve-roadmap",
		Tags:        tags,
		Summary:     "Approve the entire milestone roadmap (client)",
		Middlewares: authMw,
	}, c.ApproveRoadmapHandler)
}
