package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/utils"

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
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Get current verification draft/status",
		Middlewares: authMw,
	}, c.GetStatusHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-individual-info",
		Method:      "PUT",
		Path:        "/v1/verification/individual/info",
		Tags:        []string{"Freelancer / Profile & Verification"},
		Summary:     "[Freelancer] Step 1: Save Individual Info (Aadhaar/PAN)",
		Description: "Accepted `doc_type` values: AADHAAR, PAN, PASSPORT, DRIVING_LICENSE. Expected `id_number` shape per type: AADHAAR = 12 digits; PAN = 10 characters (5 letters, 4 digits, 1 letter); PASSPORT = 8 characters; DRIVING_LICENSE = 9 to 16 characters.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateIndividualInfoRequest{
						DocType:  "AADHAAR",
						IDNumber: "234567890123",
					},
				},
			},
		},
	}, c.UpdateIndividualInfoHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-business-info",
		Method:      "PUT",
		Path:        "/v1/verification/business/info",
		Tags:        []string{"Client / Profile & Verification"},
		Summary:     "[Client] Step 1: Save Business Info",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateBusinessInfoRequest{
						CompanyName:        "Nimbus Analytics",
						GSTNumber:          "27ABCDE1234F1Z5",
						PANNumber:          "ABCDE1234F",
						RegistrationNumber: "U12345MH2024PTC123456",
						Website:            "https://nimbusanalytics.com",
						Country:            "India",
					},
				},
			},
		},
	}, c.UpdateBusinessInfoHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-bank-details",
		Method:      "PUT",
		Path:        "/v1/verification/bank-details",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Step 3: Save Bank Details",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateBankDetailsRequest{
						BankName:          "Kotak Mahindra Bank",
						AccountHolderName: "Nimbus Analytics Pvt Ltd",
						AccountNumber:     "9012345678901234",
						IFSCCode:          "KKBK0001234",
					},
				},
			},
		},
	}, c.UpdateBankDetailsHandler)

	// Upload Endpoint
	huma.Register(api, huma.Operation{
		OperationID:  "upload-verification-document",
		Method:       "POST",
		Path:         "/v1/verification/upload",
		Tags:         []string{"Common / Profile & Verification"},
		Summary:      "[Common] Step 2: Upload KYC Document",
		Description:  "Uploads a KYC document as `multipart/form-data` with two form fields: `document_type` (one of aadhaar, govt_id, selfie, gst_certificate, incorporation_certificate, pan_card, business_registration, cancelled_cheque) and `file` (PDF or image: JPEG, PNG, WebP). Maximum file size: 2 MiB.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw,
	}, c.UploadDocumentHandler)

	// Submit Endpoint
	huma.Register(api, huma.Operation{
		OperationID: "submit-verification",
		Method:      "POST",
		Path:        "/v1/verification/submit",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Step 4: Submit Verification for Review",
		Description: "No request body is required; the document must already be uploaded and the caller must be authenticated.",
		Middlewares: authMw,
	}, c.SubmitVerificationHandler)

	// ==========================================
	// 2. Admin Flow
	// ==========================================

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-verification-queue",
		Method:      "GET",
		Path:        "/v1/admin/verification/queue",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] List all pending verifications",
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
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Queue of AI auto-eligible records",
		Middlewares: adminMw,
	}, c.AdminGetAutoEligibleHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-get-manual-review",
		Method:      "GET",
		Path:        "/v1/admin/verification/manual-review",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Queue of borderline records",
		Middlewares: adminMw,
	}, c.AdminGetManualReviewHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-verification-detail",
		Method:      "GET",
		Path:        "/v1/admin/verification/{id}/detail",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Full verification detail with AI scores",
		Middlewares: adminMw,
	}, c.AdminGetDetailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-bulk-approve",
		Method:      "POST",
		Path:        "/v1/admin/verification/bulk-approve",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Bulk approve auto-eligible records",
		Middlewares: adminMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.BulkApproveRequest{
						VerificationIDs: []string{
							"3f2504e0-4f89-41d3-9a0c-0305e82c3301",
							"7c9e6679-7425-40de-944b-e07fc1f90ae7",
						},
					},
				},
			},
		},
	}, c.AdminBulkApproveHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-manual-action",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/review",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Single approve/reject from manual review",
		Description: "`rejection_reason` is required when `action` is REJECT and must be omitted when `action` is APPROVE.",
		Middlewares: adminMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AdminManualActionRequest{
						Action:          "REJECT",
						RejectionReason: "PAN details are illegible; please re-upload a clearer document.",
						ManualNote:      "Rejected during manual review: document quality insufficient.",
					},
				},
			},
		},
	}, c.AdminManualActionHandler)

	huma.Register(api, huma.Operation{
		OperationID: "admin-escalate-manual",
		Method:      "POST",
		Path:        "/v1/admin/verification/{id}/escalate",
		Tags:        []string{"Admin / Verification"},
		Summary:     "[Admin] Escalate auto-eligible to manual review",
		Middlewares: adminMw,
		RequestBody: &huma.RequestBody{
			Required: true, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AdminEscalateRequest{
						Reason: "AI confidence is borderline; needs a human check before the auto-approve window closes.",
					},
				},
			},
		},
	}, c.AdminEscalateHandler)
}
