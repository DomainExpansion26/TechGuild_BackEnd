package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterProjectApplicationRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// All routes are protected

	// Freelancer / Agency
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-my-applications", Method: "GET", Path: "/v1/applications/my", Tags: []string{"Applications"}, Summary: "Get my applications", Middlewares: authMw}, controllers.GetMyApplicationsHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "withdraw-application", Method: "DELETE", Path: "/v1/applications/{application_id}", Tags: []string{"Applications"}, Summary: "Withdraw an application", Middlewares: authMw}, controllers.WithdrawApplicationHandler)

	// Client
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "accept-application",
		Method:      "POST",
		Path:        "/v1/applications/{application_id}/accept",
		Tags:        []string{"Applications"},
		Summary:     "Accept an application",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AcceptApplicationRequest{
						Message: "Great profile, let's discuss further.",
					},
				},
			},
		},
	}, controllers.AcceptApplicationHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-application",
		Method:      "POST",
		Path:        "/v1/applications/{application_id}/reject",
		Tags:        []string{"Applications"},
		Summary:     "Reject an application",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectApplicationRequest{
						Reason: "Profile does not match project requirements",
					},
				},
			},
		},
	}, controllers.RejectApplicationHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "shortlist-application",
		Method:      "POST",
		Path:        "/v1/applications/{application_id}/shortlist",
		Tags:        []string{"Applications"},
		Summary:     "Shortlist an application",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ShortlistApplicationRequest{
						Message: "You are shortlisted, we will reach out soon.",
					},
				},
			},
		},
	}, controllers.ShortlistApplicationHandler)

	// Project-scoped
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "apply-to-project",
		Method:      "POST",
		Path:        "/v1/projects/{project_id}/apply",
		Tags:        []string{"Applications"},
		Summary:     "Apply to a project",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApplyProjectRequest{
						CoverLetter:       "I am interested in this project and have relevant experience in web development.",
						ProposedBudget:    1500.00,
						Currency:          "USD",
						EstimatedDuration: "2 weeks",
					},
				},
			},
		},
	}, controllers.ApplyProjectHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-project-applications", Method: "GET", Path: "/v1/projects/{project_id}/applications", Tags: []string{"Applications"}, Summary: "Get applications for a project", Middlewares: authMw}, controllers.GetProjectApplicationsHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func ProjectApplicationRoutes(router *gin.Engine) {
}
