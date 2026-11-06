package dto

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// ==========================================
// 1. INPUT DTOs (Step-by-Step)
// ==========================================

// Step 1: Individual Info
type UpdateIndividualInfoRequest struct {
	DocType  string `json:"doc_type" enum:"AADHAAR,PAN,PASSPORT,DRIVING_LICENSE" required:"true" example:"AADHAAR"`
	IDNumber string `json:"id_number" minLength:"4" maxLength:"20" required:"true" example:"123456789012"`
}

// Step 1: Business Info
type UpdateBusinessInfoRequest struct {
	CompanyName        string `json:"company_name" minLength:"2" maxLength:"255" required:"true" example:"Nimbus Analytics"`
	GSTNumber          string `json:"gst_number" pattern:"^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$" required:"true" example:"27ABCDE1234F1Z5"`
	PANNumber          string `json:"pan_number" pattern:"^[A-Z]{5}[0-9]{4}[A-Z]{1}$" required:"true" example:"ABCDE1234F"`
	RegistrationNumber string `json:"registration_number" example:"U12345MH2024PTC123456"`
	Website            string `json:"website" format:"uri" example:"https://nimbusanalytics.com"`
	Country            string `json:"country" required:"true" example:"India"`
}

// Step 3: Bank Details
type UpdateBankDetailsRequest struct {
	BankName          string `json:"bank_name" required:"true" example:"HDFC Bank"`
	AccountHolderName string `json:"account_holder_name" required:"true" example:"Nimbus Analytics Pvt Ltd"`
	AccountNumber     string `json:"account_number" minLength:"6" maxLength:"20" required:"true" example:"12345678901234"`
	IFSCCode          string `json:"ifsc_code" pattern:"^[A-Z]{4}0[A-Z0-9]{6}$" required:"true" example:"HDFC0001234"`
}

// Step 2: File Upload (Huma Multipart wrapper)
type UploadDocumentForm struct {
	DocumentType string        `form:"document_type" required:"true" enum:"aadhaar,govt_id,selfie,gst_certificate,incorporation_certificate,pan_card,business_registration,cancelled_cheque"`
	File         huma.FormFile `form:"file" required:"true"`
}

// Admin Action
type AdminReviewRequest struct {
	Action          string `json:"action" enum:"APPROVE,REJECT" required:"true"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// ==========================================
// 2. OUTPUT DTOs
// ==========================================

type VerificationMessageResponse struct {
	Message string `json:"message" example:"Data saved successfully"`
}

type UploadDocumentResponse struct {
	Message      string `json:"message"`
	FileURL      string `json:"file_url"`
	DocumentType string `json:"document_type"`
}

// Status & Progress (Frontend dashboard ke liye)
type VerificationStatusResponse struct {
	Type            string     `json:"type" example:"business"`
	Status          string     `json:"status" enum:"draft,pending,approved,rejected"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`

	// Progress flags
	HasIndividualInfo bool `json:"has_individual_info"`
	HasBusinessInfo   bool `json:"has_business_info"`
	HasBankDetails    bool `json:"has_bank_details"`
	UploadedDocsCount int  `json:"uploaded_docs_count"`
}

type AdminVerificationItem struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	UserEmail     string     `json:"user_email"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	CompanyName   string     `json:"company_name,omitempty"`
	AccountNumber string     `json:"account_number,omitempty"` // Decrypted for Admin
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
}

type AdminVerificationListResponse struct {
	Items []AdminVerificationItem `json:"items"`
}

// ==========================================
// 3. HUMA WRAPPERS
// ==========================================

type UpdateIndividualInfoInput struct{ Body UpdateIndividualInfoRequest }
type UpdateIndividualInfoOutput struct{ Body VerificationMessageResponse }

type UpdateBusinessInfoInput struct{ Body UpdateBusinessInfoRequest }
type UpdateBusinessInfoOutput struct{ Body VerificationMessageResponse }

type UpdateBankDetailsInput struct{ Body UpdateBankDetailsRequest }
type UpdateBankDetailsOutput struct{ Body VerificationMessageResponse }

type GetVerificationStatusInput struct{}
type GetVerificationStatusOutput struct{ Body VerificationStatusResponse }

type UploadDocumentInput struct {
	RawBody huma.MultipartFormFiles[UploadDocumentForm]
}
type UploadDocumentOutput struct{ Body UploadDocumentResponse }

type SubmitVerificationInput struct{}
type SubmitVerificationOutput struct{ Body VerificationMessageResponse }

// Admin
type AdminReviewInput struct {
	ID   string `path:"id"`
	Body AdminReviewRequest
}
type AdminReviewOutput struct{ Body VerificationMessageResponse }

type GetAdminPendingListInput struct{}
type GetAdminPendingListOutput struct{ Body AdminVerificationListResponse }

// ==========================================
// 4. KYC AI PIPELINE (Admin Panel, Phase D)
// ==========================================

// VerificationAIScores surfaces the parsed AI/ML result bundle stored as JSONB
// on the verification record. Pointers distinguish "vendor did not return a
// score" from "score is zero".
type VerificationAIScores struct {
	FaceSimilarity *float64 `json:"face_similarity,omitempty"`
	LivenessScore  *float64 `json:"liveness_score,omitempty"`
	OCRConfidence  *float64 `json:"ocr_confidence,omitempty"`
	OCRName        *string  `json:"ocr_name,omitempty"`
	OCRDOB         *string  `json:"ocr_dob,omitempty"`
}

// VerificationDocRef is a document reference for admin review (type + URL).
type VerificationDocRef struct {
	Type string `json:"type" example:"govt_id"`
	URL  string `json:"url" example:"https://res.cloudinary.com/..."`
}

// VerificationQueueItem is the row shape for both admin queues.
type VerificationQueueItem struct {
	ID               string                `json:"id" format:"uuid"`
	UserID           string                `json:"user_id" format:"uuid"`
	UserEmail        string                `json:"user_email" example:"user@example.com"`
	VerificationType string                `json:"verification_type" example:"individual"`
	SubmittedAt      *time.Time            `json:"submitted_at,omitempty"`
	MLCompletedAt    *time.Time            `json:"ml_completed_at,omitempty"`
	AgeHours         float64               `json:"age_hours" example:"26.5"`
	ReviewPriority   string                `json:"review_priority" enum:"auto_eligible,manual_review,auto_reject,auto_approved"`
	AIScores         *VerificationAIScores `json:"ai_scores,omitempty"`
	DocumentURLs     []VerificationDocRef  `json:"document_urls"`
	Status           string                `json:"status" enum:"draft,pending,approved,rejected"`
}

// QueuePaginationMeta is the pagination block for queue responses.
type QueuePaginationMeta struct {
	Limit   int   `json:"limit" example:"20"`
	Offset  int   `json:"offset" example:"0"`
	Total   int64 `json:"total" example:"137"`
	HasMore bool  `json:"has_more"`
}

type GetAutoEligibleQueueResponse struct {
	Items      []VerificationQueueItem `json:"items"`
	Pagination QueuePaginationMeta     `json:"pagination"`
}

type GetManualReviewQueueResponse struct {
	Items      []VerificationQueueItem `json:"items"`
	Pagination QueuePaginationMeta     `json:"pagination"`
}

// AdminVerificationDetailResponse is the full record view for the review
// screen: identity fields, pipeline state, AI scores and documents.
type AdminVerificationDetailResponse struct {
	ID               string `json:"id" format:"uuid"`
	UserID           string `json:"user_id" format:"uuid"`
	UserEmail        string `json:"user_email"`
	VerificationType string `json:"verification_type" example:"individual"`
	Status           string `json:"status" enum:"draft,pending,approved,rejected"`
	RejectionReason  string `json:"rejection_reason,omitempty"`

	// Pipeline state
	ReviewPriority         string     `json:"review_priority" enum:"auto_eligible,manual_review,auto_reject,auto_approved"`
	MLStatus               string     `json:"ml_status" enum:"idle,queued,processing,ml_completed,ml_failed"`
	MLJobID                string     `json:"ml_job_id,omitempty"`
	MLSubmittedAt          *time.Time `json:"ml_submitted_at,omitempty"`
	MLCompletedAt          *time.Time `json:"ml_completed_at,omitempty"`
	MLVendor               string     `json:"ml_vendor,omitempty"`
	MLModelVersion         string     `json:"ml_model_version,omitempty"`
	MLProcessingTime       int        `json:"ml_processing_time,omitempty"`
	AdminReviewRequestedBy string     `json:"admin_review_requested_by,omitempty"`
	AdminReviewRequestedAt *time.Time `json:"admin_review_requested_at,omitempty"`
	BulkReviewRequestedAt  *time.Time `json:"bulk_review_requested_at,omitempty"`

	// Decision timestamps
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	BulkApprovedAt *time.Time `json:"bulk_approved_at,omitempty"`
	AutoApprovedAt *time.Time `json:"auto_approved_at,omitempty"`
	RejectedAt     *time.Time `json:"rejected_at,omitempty"`

	AIScores     *VerificationAIScores `json:"ai_scores,omitempty"`
	DocumentURLs []VerificationDocRef  `json:"document_urls"`
}

// ---- Requests ----

type BulkApproveRequest struct {
	VerificationIDs []string `json:"verification_ids" required:"true" minItems:"1" maxItems:"100" doc:"Verification record IDs to bulk approve"`
}

type BulkApproveResponse struct {
	ApprovedCount int    `json:"approved_count" example:"12"`
	Message       string `json:"message"`
}

type AdminManualActionRequest struct {
	Action          string `json:"action" enum:"APPROVE,REJECT" required:"true" doc:"Approve or reject the record"`
	RejectionReason string `json:"rejection_reason,omitempty" doc:"Required when action is REJECT"`
	ManualNote      string `json:"manual_note,omitempty" doc:"Optional internal note recorded in the audit trail"`
}

type AdminEscalateRequest struct {
	Reason string `json:"reason" required:"true" maxLength:"500" doc:"Why the admin wants human review before the auto-approve window"`
}

// ---- Huma wrappers ----

type GetAutoEligibleQueueInput struct {
	Limit       int     `query:"limit" default:"20" doc:"Max items (default 20, max 100)"`
	Offset      int     `query:"offset" default:"0" doc:"Offset items"`
	MinAgeHours float64 `query:"min_age_hours" default:"24" doc:"Only include records whose ML completion is at least this many hours old"`
}
type GetAutoEligibleQueueOutput struct{ Body GetAutoEligibleQueueResponse }

type GetManualReviewQueueInput struct {
	SortBy  string `query:"sort_by" default:"submitted_at" enum:"submitted_at,liveness_score,face_similarity" doc:"Weakest-first sorting uses the AI scores"`
	SortDir string `query:"sort_dir" default:"asc" enum:"asc,desc"`
	Limit   int    `query:"limit" default:"20" doc:"Max items (default 20, max 100)"`
	Offset  int    `query:"offset" default:"0" doc:"Offset items"`
}
type GetManualReviewQueueOutput struct{ Body GetManualReviewQueueResponse }

type AdminGetDetailInput struct {
	ID string `path:"id" format:"uuid"`
}
type AdminGetDetailOutput struct {
	Body AdminVerificationDetailResponse
}

type AdminBulkApproveInput struct{ Body BulkApproveRequest }
type AdminBulkApproveOutput struct{ Body BulkApproveResponse }

type AdminManualActionInput struct {
	ID   string `path:"id" format:"uuid"`
	Body AdminManualActionRequest
}
type AdminManualActionOutput struct{ Body VerificationMessageResponse }

type AdminEscalateInput struct {
	ID   string `path:"id" format:"uuid"`
	Body AdminEscalateRequest
}
type AdminEscalateOutput struct{ Body VerificationMessageResponse }
