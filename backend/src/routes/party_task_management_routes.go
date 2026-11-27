package routes

import (
	"time"

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

	// 1. Active quests (multi-quest banner + dropdown switcher)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-active-quests",
		Method:      "GET",
		Path:        "/v1/party/task-management/quests/active",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] List caller's active party quests",
		Description: "Returns in-progress quests where the caller is the team lead or an active party member.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDActiveQuestsHandler)

	// 2. Quest dashboard (banner stage, KPIs, milestone cards)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-dashboard",
		Method:      "GET",
		Path:        "/v1/party/task-management/dashboard",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get party quest dashboard",
		Description: "Returns banner stage, KPI counts, milestone cards with assignees, tasks, attachment counts, and recent activity.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDDashboardHandler)

	// 3. Members panel (member list + assignment status)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-tmd-members",
		Method:      "GET",
		Path:        "/v1/party/task-management/members",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] List party members and assignments",
		Description: "Returns active members, their rank/skills, and which milestones each is assigned to.",
		Middlewares: authMw,
	}, controllers.GetPartyTMDMembersHandler)

	// 4. Assign members to a milestone (team lead only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "assign-party-tmd-milestone",
		Method:      "POST",
		Path:        "/v1/party/task-management/milestones/{milestone_id}/assign",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Assign party members to milestone",
		Description: "Replaces the milestone's assignees. Caller must be the team lead and the milestone must be agreed or later.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AssignMilestoneBody{
						MemberIDs: []string{"1f2e3d4c-5b6a-4c7d-8e9f-0a1b2c3d4e5f", "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d"},
						StartDate: ptr(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)),
						EndDate:   ptr(time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC)),
						Note:      "Own the API integration and payment-webhook hardening.",
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
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] List milestone assignees",
		Middlewares: authMw,
	}, controllers.ListMilestoneAssignmentsHandler)
}

// RegisterMilestoneAttachmentRoutes registers deliverable upload/list/delete
// under a milestone.
func RegisterMilestoneAttachmentRoutes(api huma.API) {
	authMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}

	huma.Register(api, huma.Operation{
		Security:     []map[string][]string{{"BearerAuth": {}}},
		OperationID:  "upload-milestone-attachment",
		Method:       "POST",
		Path:         "/v1/quests/{id}/milestones/{milestone_id}/attachments",
		Tags:         []string{"Freelancer / Milestones & Deliverables"},
		Summary:      "[Freelancer] Upload deliverable attachment",
		Description:  "Upload the deliverable as a multipart/form-data request (form field `file`). Accepted types: PDF or image (JPEG, PNG, WebP); maximum size 2 MiB. Stores the file on Cloudinary and attaches it to the milestone. Caller must be the client, team lead, or an active party member.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw,
	}, controllers.UploadMilestoneAttachmentHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-milestone-attachments",
		Method:      "GET",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/attachments",
		Tags:        []string{"Common / Milestones Read"},
		Summary:     "[Common] List a milestone's attachments",
		Middlewares: authMw,
	}, controllers.ListMilestoneAttachmentsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-milestone-attachment",
		Method:      "DELETE",
		Path:        "/v1/quests/{id}/milestones/{milestone_id}/attachments/{attachment_id}",
		Tags:        []string{"Freelancer / Milestones & Deliverables"},
		Summary:     "[Freelancer] Delete milestone attachment",
		Description: "Allowed for the uploader, the quest client, or the team lead.",
		Middlewares: authMw,
	}, controllers.DeleteMilestoneAttachmentHandler)
}
