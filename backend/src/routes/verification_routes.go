package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterVerificationRoutes(api huma.API, c *controllers.VerificationController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	adminMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api), middleware.AdminMiddlewareHuma(api)}

	// ==========================================
	// 1. User Flow
	// ==========================================

	huma.Register(api, huma.Operation{
		OperationID: "get-verification-status",
		Method:      "GET",
		Path:        "/v1/verification/status",
		Tags:        []string{"Verification"},
		Summary:     "Get current verification draft/status",
		Middlewares: authMw,
	}, c.GetStatusHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-individual-info",
		Method:      "PUT",
		Path:        "/v1/verification/individual/info",
		Tags:        []string{"Verification"},
		Summary:     "Step 1: Save Individual Info (Aadhaar/PAN)",
		Middlewares: authMw,
	}, c.UpdateIndividualInfoHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-business-info",
		Method:      "PUT",
		Path:        "/v1/verification/business/info",
		Tags:        []string{"Verification"},
		Summary:     "Step 1: Save Business Info",
		Middlewares: authMw,
	}, c.UpdateBusinessInfoHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-bank-details",
		Method:      "PUT",
		Path:        "/v1/verification/bank-details",
		Tags:        []string{"Verification"},
		Summary:     "Step 3: Save Bank Details",
		Middlewares: authMw,
	}, c.UpdateBankDetailsHandler)

	// Upload Endpoint
	huma.Register(api, huma.Operation{
		OperationID: "upload-verification-document",
		Method:      "POST",
		Path:        "/v1/verification/upload",
		Tags:        []string{"Verification"},
		Summary:     "Step 2: Upload KYC Document",
		Middlewares: authMw,
	}, c.UploadDocumentHandler)

	// Submit Endpoint
	huma.Register(api, huma.Operation{
		OperationID: "submit-verification",
		Method:      "POST",
		Path:        "/v1/verification/submit",
		Tags:        []string{"Verification"},
		Summary:     "Step 4: Submit Verification for Review",
		Middlewares: authMw,
	}, c.SubmitVerificationHandler)

	// ==========================================
	// 2. Admin Flow
	// ==========================================

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-verification-queue",
		Method:      "GET",
		Path:        "/v1/admin/verification/queue",
		Tags:        []string{"Admin Verification"},
		Summary:     "List all pending verifications",
		Middlewares: adminMw,
	}, c.AdminGetQueueHandler)

	// ==========================================
	// 3. KYC AI Pipeline (Phase D)
	// ==========================================
	// NOTE: the legacy AdminReviewHandler registration was replaced by
	// admin-manual-action below (same path, richer audit + queue semantics).

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-ai-auto-eligible",
		Method:      "GET",
		Path:        "/v1/admin/verification/ai-auto-eligible",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Queue of AI auto-eligible records (24-48h batch approval window)",
		Middlewares: adminMw,
	}, c.AdminGetAutoEligibleHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-manual-review",
		Method:      "GET",
		Path:        "/v1/admin/verification/manual-review",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Queue of borderline records (supports weakest-first AI score sorting)",
		Middlewares: adminMw,
	}, c.AdminGetManualReviewHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-verification-detail",
		Method:      "GET",
		Path:        "/v1/admin/verification/{id}/detail",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Full verification detail with AI scores and documents",
		Middlewares: adminMw,
	}, c.AdminGetDetailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-bulk-approve",
		Method:      "POST",
		Path:        "/v1/admin/verification/bulk-approve",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Bulk approve auto-eligible records (transactional, audited)",
		Middlewares: adminMw,
	}, c.AdminBulkApproveHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-manual-action",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/review",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Single approve/reject from manual review (audited)",
		Middlewares: adminMw,
	}, c.AdminManualActionHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-escalate-manual",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/escalate",
		Tags:        []string{"Admin Verification", "KYC AI Pipeline"},
		Summary:     "Escalate auto-eligible record to manual review",
		Middlewares: adminMw,
	}, c.AdminEscalateHandler)
}
