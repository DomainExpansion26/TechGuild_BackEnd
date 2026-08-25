package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterVerificationRoutes registers all verification endpoints via Huma
func RegisterVerificationRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Individual Verification (file-upload)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "submit-identity-verification",
		Method:      "POST",
		Path:        "/v1/verification/identity/submit",
		Tags:        []string{"Verification"},
		Summary:     "Submit identity verification",
		Middlewares: authMw,
	}, controllers.SubmitIdentityVerificationHandler)

	// Business Verification (file-upload)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "submit-business-verification",
		Method:      "POST",
		Path:        "/v1/verification/business/submit",
		Tags:        []string{"Verification"},
		Summary:     "Submit business verification",
		Middlewares: authMw,
	}, controllers.SubmitBusinessVerificationHandler)

	// Identity Verification Status
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-identity-verification-status",
		Method:      "GET",
		Path:        "/v1/verification/identity/status",
		Tags:        []string{"Verification"},
		Summary:     "Get identity verification status",
		Middlewares: authMw,
	}, controllers.GetIdentityVerificationStatusHandler)

	// Generic Verification Status
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-verification-status",
		Method:      "GET",
		Path:        "/v1/verification/status",
		Tags:        []string{"Verification"},
		Summary:     "Get verification status",
		Middlewares: authMw,
	}, controllers.GetVerificationStatusHandler)

	// Resubmit Verification (file-upload)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "resubmit-verification",
		Method:      "POST",
		Path:        "/v1/verification/resubmit/{record_id}",
		Tags:        []string{"Verification"},
		Summary:     "Resubmit verification",
		Middlewares: authMw,
	}, controllers.ResubmitVerificationHandler)

	// Admin Verification
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-verification-queue",
		Method:      "GET",
		Path:        "/v1/admin/verification/queue",
		Tags:        []string{"Admin Verification"},
		Summary:     "Get verification queue",
		Middlewares: authMw,
	}, controllers.GetVerificationQueueHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "approve-verification",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/approve",
		Tags:        []string{"Admin Verification"},
		Summary:     "Approve verification",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AdminApproveRequest{
						Note: "Documents verified successfully",
					},
				},
			},
		},
	}, controllers.ApproveVerificationHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-verification",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/reject",
		Tags:        []string{"Admin Verification"},
		Summary:     "Reject verification",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AdminRejectRequest{
						Reason: "Documents are blurry",
					},
				},
			},
		},
	}, controllers.RejectVerificationHandler)
}
