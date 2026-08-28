package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterProjectRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Public
	huma.Register(api, huma.Operation{OperationID: "browse-projects", Method: "GET", Path: "/v1/projects", Tags: []string{"Projects"}, Summary: "Browse published projects", Security: []map[string][]string{}}, controllers.BrowseProjectsHandler)
	huma.Register(api, huma.Operation{OperationID: "search-projects", Method: "GET", Path: "/v1/projects/search", Tags: []string{"Projects"}, Summary: "Search projects", Security: []map[string][]string{}}, controllers.SearchProjectsHandler)
	huma.Register(api, huma.Operation{OperationID: "get-project-by-id", Method: "GET", Path: "/v1/projects/{project_id}", Tags: []string{"Projects"}, Summary: "Get project by ID", Security: []map[string][]string{}}, controllers.GetProjectByIDHandler)

	// Protected
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-project",
		Method:      "POST",
		Path:        "/v1/projects",
		Tags:        []string{"Projects"},
		Summary:     "Create a new project",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateProjectRequest{
						Title:               "Build a Web Application",
						Description:         "Need a full-stack web application with user authentication and dashboard",
						Category:            "web-development",
						BudgetType:          "fixed",
						MinBudget:           1000.00,
						MaxBudget:           5000.00,
						Currency:            "USD",
						ExperienceLevel:     "intermediate",
						ProjectType:         "remote",
						Duration:            "2 months",
						RequiredSkills:      []string{"React", "Node.js", "PostgreSQL"},
						Visibility:          "public",
						ApplicationDeadline: "2026-03-01T00:00:00Z",
						EstimatedStartDate:  "2026-03-15T00:00:00Z",
						EstimatedEndDate:    "2026-05-15T00:00:00Z",
						MaxApplications:     10,
						IsFeatured:          false,
						IsUrgent:            false,
					},
				},
			},
		},
	}, controllers.CreateProjectHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-project",
		Method:      "PATCH",
		Path:        "/v1/projects/{project_id}",
		Tags:        []string{"Projects"},
		Summary:     "Update a project",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateProjectRequest{
						Title:               "Build a Web Application v2",
						Description:         "Updated requirements for the web application",
						Category:            "web-development",
						BudgetType:          "fixed",
						MinBudget:           2000.00,
						MaxBudget:           8000.00,
						Currency:            "USD",
						ExperienceLevel:     "senior",
						ProjectType:         "remote",
						Duration:            "3 months",
						RequiredSkills:      []string{"React", "Node.js", "PostgreSQL", "Redis"},
						Visibility:          "public",
						ApplicationDeadline: "2026-03-01T00:00:00Z",
						EstimatedStartDate:  "2026-03-15T00:00:00Z",
						EstimatedEndDate:    "2026-05-15T00:00:00Z",
						MaxApplications:     15,
						IsFeatured:          true,
						IsUrgent:            false,
					},
				},
			},
		},
	}, controllers.UpdateProjectHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-project", Method: "DELETE", Path: "/v1/projects/{project_id}", Tags: []string{"Projects"}, Summary: "Delete a project", Middlewares: authMw}, controllers.DeleteProjectHandler)

	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "publish-project", Method: "POST", Path: "/v1/projects/{project_id}/publish", Tags: []string{"Projects"}, Summary: "Publish a project", Middlewares: authMw}, controllers.PublishProjectHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "close-project",
		Method:      "POST",
		Path:        "/v1/projects/{project_id}/close",
		Tags:        []string{"Projects"},
		Summary:     "Close a project",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CloseProjectRequest{
						Reason: "Project requirements changed",
					},
				},
			},
		},
	}, controllers.CloseProjectHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reopen-project",
		Method:      "POST",
		Path:        "/v1/projects/{project_id}/reopen",
		Tags:        []string{"Projects"},
		Summary:     "Reopen a project",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ReopenProjectRequest{
						Reason: "Client needs additional changes",
					},
				},
			},
		},
	}, controllers.ReopenProjectHandler)

	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-my-projects", Method: "GET", Path: "/v1/projects/my", Tags: []string{"Projects"}, Summary: "Get my projects", Middlewares: authMw}, controllers.GetMyProjectsHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func ProjectRoutes(router *gin.Engine) {
}
