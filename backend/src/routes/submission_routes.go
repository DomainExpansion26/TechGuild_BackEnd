package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterSubmissionRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Submission CRUD
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-submission",
		Method:      "POST",
		Path:        "/v1/submissions",
		Tags:        []string{"Submissions"},
		Summary:     "Create a submission",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateSubmissionRequest{
						MilestoneID:   "550e8400-e29b-41d4-a716-446655440000",
						Message:       "Work completed as per requirements",
						SubmissionURL: "https://github.com/user/repo/pull/123",
						AttachmentURL: "https://storage.example.com/files/demo.zip",
					},
				},
			},
		},
	}, controllers.CreateSubmissionHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-submission-by-id", Method: "GET", Path: "/v1/submissions/{id}", Tags: []string{"Submissions"}, Summary: "Get submission by ID", Middlewares: authMw}, controllers.GetSubmissionByIDHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-submission",
		Method:      "PUT",
		Path:        "/v1/submissions/{id}",
		Tags:        []string{"Submissions"},
		Summary:     "Update a submission",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateSubmissionRequest{
						Message:       "Updated submission with fixes",
						SubmissionURL: "https://github.com/user/repo/pull/124",
						AttachmentURL: "https://storage.example.com/files/demo-v2.zip",
					},
				},
			},
		},
	}, controllers.UpdateSubmissionHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-submission", Method: "DELETE", Path: "/v1/submissions/{id}", Tags: []string{"Submissions"}, Summary: "Delete a submission", Middlewares: authMw}, controllers.DeleteSubmissionHandler)

	// Milestone submissions
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-milestone-submissions", Method: "GET", Path: "/v1/submissions/milestone/{milestone_id}", Tags: []string{"Submissions"}, Summary: "Get submissions for a milestone", Middlewares: authMw}, controllers.GetMilestoneSubmissionsHandler)

	// Client actions
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "approve-submission",
		Method:      "POST",
		Path:        "/v1/submissions/{id}/approve",
		Tags:        []string{"Submissions"},
		Summary:     "Approve a submission",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveSubmissionRequest{
						Message: "Looks good, approved!",
					},
				},
			},
		},
	}, controllers.ApproveSubmissionHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-submission",
		Method:      "POST",
		Path:        "/v1/submissions/{id}/reject",
		Tags:        []string{"Submissions"},
		Summary:     "Reject a submission",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectSubmissionRequest{
						Reason: "Work does not meet the requirements",
					},
				},
			},
		},
	}, controllers.RejectSubmissionHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func SubmissionRoutes(router *gin.Engine) {
}
