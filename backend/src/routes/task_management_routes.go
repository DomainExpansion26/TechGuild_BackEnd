package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

// RegisterTaskManagementRoutes registers all Client Task Management (TMD) routes.
func RegisterTaskManagementRoutes(api huma.API) {
	clientAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
		middleware.ClientMiddlewareHuma(api),
	}
	generalAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}

	tags := []string{"Client Task Management"}

	// 1. Get Metrics (Active Quests, Pending Reviews, Tasks in Progress, Upcoming Deadlines, Completed)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-metrics",
		Method:      "GET",
		Path:        "/v1/client/task-management/metrics",
		Tags:        tags,
		Summary:     "Get task management dashboard summary metrics",
		Description: "Returns the 5 primary KPI metric counts for the client. Can be scoped to a specific quest via query parameter.",
		Middlewares: clientAuthMw,
	}, controllers.GetTaskManagementMetricsHandler)

	// 2. Get Active Quests (Multi-Quest Banner & Dropdown Switcher)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-active-quests",
		Method:      "GET",
		Path:        "/v1/client/task-management/quests/active",
		Tags:        tags,
		Summary:     "List client's active quests for dropdown switcher",
		Description: "Returns all in-progress or completed quests for the authenticated client with progress percentages.",
		Middlewares: clientAuthMw,
	}, controllers.GetActiveQuestsHandler)

	// 3. Get Quest Dashboard Details & Donut Chart
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-quest-dashboard",
		Method:      "GET",
		Path:        "/v1/client/task-management/quests/{quest_id}/dashboard",
		Tags:        tags,
		Summary:     "Get overview and donut chart statistics for an active quest",
		Description: "Returns quest details, assigned freelancer information, donut chart milestone distribution, and recent activity preview.",
		Middlewares: clientAuthMw,
	}, controllers.GetQuestDashboardHandler)

	// 4. Get Milestones Table
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-quest-milestones",
		Method:      "GET",
		Path:        "/v1/client/task-management/quests/{quest_id}/milestones",
		Tags:        tags,
		Summary:     "Get milestones table for an active quest",
		Description: "Returns the table rows with task counts, individual progress, dates, amounts, and statuses.",
		Middlewares: clientAuthMw,
	}, controllers.GetQuestMilestonesTableHandler)

	// 5. Add Milestone (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "add-tmd-milestone",
		Method:      "POST",
		Path:        "/v1/client/task-management/quests/{quest_id}/milestones",
		Tags:        tags,
		Summary:     "DEPRECATED — use POST /v1/quests/{id}/milestones",
		Description: "DEPRECATED. Returns 410 Gone. Milestones are created by the assigned freelancer via POST /v1/quests/{id}/milestones and must go through the review/agreement/funding flow.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddMilestoneRequestBody{
						Title:       "Responsive Design & Mobile Optimization",
						Description: "Complete responsive layouts for tablet and mobile.",
						Amount:      10000,
						DueDays:     7,
					},
				},
			},
		},
	}, controllers.AddMilestoneHandler)

	// 6. Delete Milestone (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-tmd-milestone",
		Method:      "DELETE",
		Path:        "/v1/client/task-management/milestones/{milestone_id}",
		Tags:        tags,
		Summary:     "Delete a milestone and its associated tasks",
		Description: "Removes a milestone from an active quest. Only milestones in draft or revision_requested state can be deleted.",
		Middlewares: clientAuthMw,
	}, controllers.DeleteMilestoneTMDHandler)

	// 7. View Tasks (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-milestone-tasks",
		Method:      "GET",
		Path:        "/v1/client/task-management/milestones/{milestone_id}/tasks",
		Tags:        tags,
		Summary:     "View tasks under a milestone",
		Description: "Returns the list of granular tasks for the View Tasks modal.",
		Middlewares: clientAuthMw,
	}, controllers.GetMilestoneTasksHandler)

	// 8. Approve Milestone (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "approve-tmd-milestone",
		Method:      "POST",
		Path:        "/v1/client/task-management/milestones/{milestone_id}/approve",
		Tags:        tags,
		Summary:     "DEPRECATED — use POST /v1/quests/{id}/milestones/{milestone_id}/release",
		Description: "DEPRECATED. Returns 410 Gone. A milestone is marked released only by the escrow release flow (POST /v1/quests/{id}/milestones/{milestone_id}/release), which moves money into the freelancer wallet.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveMilestoneTMDRequestBody{
						Feedback: "Deliverables approved and tested",
					},
				},
			},
		},
	}, controllers.ApproveMilestoneTMDHandler)

	// 9. Cancel Milestone Approval (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "cancel-tmd-milestone-approval",
		Method:      "POST",
		Path:        "/v1/client/task-management/milestones/{milestone_id}/cancel-approval",
		Tags:        tags,
		Summary:     "DEPRECATED — no longer supported",
		Description: "DEPRECATED. Returns 410 Gone. Releasing a milestone moves money to the freelancer wallet and cannot be reverted through this endpoint.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CancelMilestoneApprovalTMDRequestBody{
						Reason: "Revisions required on mobile layouts",
					},
				},
			},
		},
	}, controllers.CancelMilestoneApprovalTMDHandler)

	// 10. Approve Quest (Modal)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "approve-tmd-quest",
		Method:      "POST",
		Path:        "/v1/client/task-management/quests/{quest_id}/approve",
		Tags:        tags,
		Summary:     "Final sign-off and quest completion",
		Description: "Marks the entire quest as completed. Requires every milestone to be released AND fully settled (MilestonePayment released + EscrowTransaction released), and collects rating and review.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveQuestTMDRequestBody{
						Rating:   5,
						Feedback: "Outstanding work delivered on time",
					},
				},
			},
		},
	}, controllers.ApproveQuestTMDHandler)

	// 11. Cancel Quest Approval (Dispute/Reopen)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "cancel-tmd-quest-approval",
		Method:      "POST",
		Path:        "/v1/client/task-management/quests/{quest_id}/cancel-approval",
		Tags:        tags,
		Summary:     "Cancel quest approval and reopen quest",
		Description: "Reverts a completed quest back to in_progress with a required justification reason.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CancelQuestApprovalTMDRequestBody{
						Reason: "Post-delivery integration defect requires patch",
					},
				},
			},
		},
	}, controllers.CancelQuestApprovalTMDHandler)

	// 12. Recent Activities Drawer
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-tmd-quest-activities",
		Method:      "GET",
		Path:        "/v1/client/task-management/quests/{quest_id}/activities",
		Tags:        tags,
		Summary:     "Get chronological activity audit log for a quest",
		Description: "Powers the Recent Activities timeline popup drawer.",
		Middlewares: clientAuthMw,
	}, controllers.GetQuestActivitiesTMDHandler)

	// 13. Create Task Under Milestone (Operational)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-tmd-milestone-task",
		Method:      "POST",
		Path:        "/v1/client/task-management/milestones/{milestone_id}/tasks",
		Tags:        tags,
		Summary:     "Create a task under a milestone",
		Description: "Creates an actionable task under a milestone. Caller must be the quest client or the assigned freelancer, and the milestone must be in draft, revision_in_progress, or work_in_progress.",
		Middlewares: generalAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateMilestoneTaskRequestBody{
						Title:       "Navbar Wireframe",
						Description: "Design responsive navigation component",
						AssigneeID:  ptr("11111111-1111-1111-1111-111111111111"),
						Status:      "todo",
					},
				},
			},
		},
	}, controllers.CreateMilestoneTaskHandler)

	// 14. Update Task Status (Operational)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-tmd-task-status",
		Method:      "PATCH",
		Path:        "/v1/client/task-management/tasks/{task_id}/status",
		Tags:        tags,
		Summary:     "Update status of a task",
		Description: "Updates task status to todo, in_progress, or completed. Caller must be the quest client or the assigned freelancer, and the parent milestone must be in draft, revision_in_progress, or work_in_progress.",
		Middlewares: generalAuthMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateTaskStatusRequestBody{
						Status: "in_progress",
					},
				},
			},
		},
	}, controllers.UpdateTaskStatusHandler)
}
