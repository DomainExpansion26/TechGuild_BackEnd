package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterContractRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-contract",
		Method:      "POST",
		Path:        "/contracts",
		Tags:        []string{"Contracts"},
		Summary:     "Create a new contract",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateContractRequest{
						ProjectID:       "550e8400-e29b-41d4-a716-446655440000",
						ApplicationID:   "550e8400-e29b-41d4-a716-446655440001",
						ContractAmount:  5000.00,
						Currency:        "USD",
						StartDate:       "2026-02-01T00:00:00Z",
						ExpectedEndDate: "2026-04-01T00:00:00Z",
					},
				},
			},
		},
	}, controllers.CreateContractHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "sign-contract",
		Method:      "PUT",
		Path:        "/contracts/{id}/sign",
		Tags:        []string{"Contracts"},
		Summary:     "Sign a contract",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SignContractRequest{
						Signature: "John Doe",
					},
				},
			},
		},
	}, controllers.SignContractHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "complete-contract",
		Method:      "PUT",
		Path:        "/contracts/{id}/complete",
		Tags:        []string{"Contracts"},
		Summary:     "Complete a contract",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CompleteContractRequest{
						CompletionNote: "All deliverables completed and approved",
					},
				},
			},
		},
	}, controllers.CompleteContractHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "cancel-contract",
		Method:      "PUT",
		Path:        "/contracts/{id}/cancel",
		Tags:        []string{"Contracts"},
		Summary:     "Cancel a contract",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CancelContractRequest{
						Reason: "Project requirements changed",
					},
				},
			},
		},
	}, controllers.CancelContractHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-contract-by-id",
		Method:      "GET",
		Path:        "/contracts/{id}",
		Tags:        []string{"Contracts"},
		Summary:     "Get contract by ID",
		Middlewares: authMw,
	}, controllers.GetContractByIDHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-client-contracts",
		Method:      "GET",
		Path:        "/contracts/client",
		Tags:        []string{"Contracts"},
		Summary:     "Get my contracts as client",
		Middlewares: authMw,
	}, controllers.GetClientContractsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-freelancer-contracts",
		Method:      "GET",
		Path:        "/contracts/freelancer",
		Tags:        []string{"Contracts"},
		Summary:     "Get my contracts as freelancer",
		Middlewares: authMw,
	}, controllers.GetFreelancerContractsHandler)
}
