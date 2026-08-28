package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterMilestoneRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-milestone",
		Method:      "POST",
		Path:        "/milestones",
		Tags:        []string{"Milestones"},
		Summary:     "Create a milestone",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateMilestoneRequest{
						ContractID:  "550e8400-e29b-41d4-a716-446655440000",
						Title:       "Design Phase",
						Description: "Complete UI/UX design for the homepage",
						Amount:      1500.00,
						DueDate:     "2026-03-01T00:00:00Z",
					},
				},
			},
		},
	}, controllers.CreateMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-contract-milestones",
		Method:      "GET",
		Path:        "/milestones/contract/{contract_id}",
		Tags:        []string{"Milestones"},
		Summary:     "Get milestones for a contract",
		Middlewares: authMw,
	}, controllers.GetContractMilestonesHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-milestone-by-id",
		Method:      "GET",
		Path:        "/milestones/{id}",
		Tags:        []string{"Milestones"},
		Summary:     "Get milestone by ID",
		Middlewares: authMw,
	}, controllers.GetMilestoneByIDHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-milestone",
		Method:      "PUT",
		Path:        "/milestones/{id}",
		Tags:        []string{"Milestones"},
		Summary:     "Update a milestone",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateMilestoneRequest{
						Title:       "Design Phase v2",
						Description: "Updated UI/UX design with feedback",
						Amount:      2000.00,
						DueDate:     "2026-03-15T00:00:00Z",
					},
				},
			},
		},
	}, controllers.UpdateMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-milestone",
		Method:      "DELETE",
		Path:        "/milestones/{id}",
		Tags:        []string{"Milestones"},
		Summary:     "Delete a milestone",
		Middlewares: authMw,
	}, controllers.DeleteMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "submit-milestone",
		Method:      "POST",
		Path:        "/milestones/{id}/submit",
		Tags:        []string{"Milestones"},
		Summary:     "Submit a milestone (freelancer)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SubmitMilestoneRequest{
						Notes:          "Completed design files, Figma link attached",
						DeliverableURL: "https://drive.google.com/file/d/abc123",
					},
				},
			},
		},
	}, controllers.SubmitMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "approve-milestone",
		Method:      "POST",
		Path:        "/milestones/{id}/approve",
		Tags:        []string{"Milestones"},
		Summary:     "Approve a milestone (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApproveMilestoneRequest{
						Note: "Great work, approved!",
					},
				},
			},
		},
	}, controllers.ApproveMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-milestone",
		Method:      "POST",
		Path:        "/milestones/{id}/reject",
		Tags:        []string{"Milestones"},
		Summary:     "Reject a milestone (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectMilestoneRequest{
						Reason: "Logo colors do not match brand guidelines",
					},
				},
			},
		},
	}, controllers.RejectMilestoneHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "mark-milestone-paid",
		Method:      "POST",
		Path:        "/milestones/{id}/pay",
		Tags:        []string{"Milestones"},
		Summary:     "Mark milestone as paid (client)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.MarkMilestonePaidRequest{
						PaymentReference: "TXN-2026-001234",
					},
				},
			},
		},
	}, controllers.MarkMilestonePaidHandler)
}
