package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/utils"
)

// RegisterPartyTaskManagementRoutes registers the freelancer/party-side task
// management (TMD) routes. Authorization is enforced in the service layer:
// clients, the team lead, and active party members each get different rights.
func RegisterPartyTaskManagementRoutes(api huma.API) {
	authMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}
	tags := []string{"Party Task Management"}

	// 1. Active quests (multi-quest banner + dropdown switcher)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-active-quests",
		Method:      "GET",
		Path:        "/v1/party/task-management/quests/active",
		Tags:        tags,
		Summary:     "List the caller's active party quests",
		Description: "Returns in-progress quests where the caller is the team lead or an active party member.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDActiveQuestsHandler)

	// 2. Quest dashboard (banner stage, KPIs, milestone cards)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-dashboard",
		Method:      "GET",
		Path:        "/v1/party/task-management/dashboard",
		Tags:        tags,
		Summary:     "Get the party quest dashboard",
		Description: "Returns banner stage, KPI counts, milestone cards with assignees, tasks, attachment counts, and recent activity.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDDashboardHandler)

	// 3. Members panel (member list + assignment status)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-members",
		Method:      "GET",
		Path:        "/v1/party/task-management/members",
		Tags:        tags,
		Summary:     "List party members and their milestone assignments",
		Description: "Returns active members, their rank/skills, and which milestones each is assigned to.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDMembersHandler)

	// 4. Assign members to a milestone (team lead only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "assign-party-tmd-milestone",
		Method:      "POST",
		Path:        "/v1/party/task-management/milestones/{milestone_id}/assign",
		Tags:        tags,
		Summary:     "Assign one or more party members to a milestone",
		Description: "Replaces the milestone's assignees. Caller must be the team lead and the milestone must be agreed or later.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AssignMilestoneBody{
						MemberIDs: []string{"11111111-1111-1111-1111-111111111111"},
						Note:      "Please review the API integration tasks",
					},
				},
			},
		},
	}, controllers.AssignMilestoneHandler)

	// 5. List a milestone's assignees
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-party-tmd-milestone-assignments",
		Method:      "GET",
		Path:        "/v1/party/task-management/milestones/{milestone_id}/assignments",
		Tags:        tags,
		Summary:     "List the assignees of a milestone",
		Middlewares: authMw,
	}, controllers.ListMilestoneAssignmentsHandler)
}

// RegisterMilestoneAttachmentRoutes registers deliverable upload/list/delete
// under a milestone.
func RegisterMilestoneAttachmentRoutes(api huma.API) {
	authMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}
	tags := []string{"Milestone Attachments"}

	huma.Register(api, huma.Operation{
		Security:     []map[string][]string{{"BearerAuth": {}}},
		OperationID:  "upload-milestone-attachment",
		Method:       "POST",
		Path:         "/v1/quests/{id}/milestones/{milestone_id}/attachments",
		Tags:         tags,
		Summary:      "Upload a deliverable attachment (PDF or image, max 2MB)",
		Description:  "Upload the deliverable as multipart form field `file`. Accepted types: PDF or image (JPEG, PNG, WebP); maximum size 2 MiB. Stores the file on Cloudinary and attaches it to the milestone. Caller must be the client, team lead, or an active party member.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw,
	}, controllers.UploadMilestoneAttachmentHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-milestone-attachments",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/attachments",
		Tags:        tags,
		Summary:     "List a milestone's attachments",
		Middlewares: authMw,
	}, controllers.ListMilestoneAttachmentsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-milestone-attachment",
		Method:      "DELETE",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/attachments/{attachment_id}",
		Tags:        tags,
		Summary:     "Delete a milestone attachment",
		Description: "Allowed for the uploader, the quest client, or the team lead.",
		Middlewares: authMw,
	}, controllers.DeleteMilestoneAttachmentHandler)
}
