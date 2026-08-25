package dto

import "mime/multipart"

// =========================
// Individual Verification
// =========================

type IdentityVerificationRequest struct {
	GovtIDType   string `json:"govt_id_type" form:"govt_id_type" huma:"required" example:"aadhaar"`
	GovtIDNumber string `json:"govt_id_number" form:"govt_id_number" huma:"required" example:"1234-5678-9012"`
}

type IdentityVerificationResponse struct {
	Message              string `json:"message" example:"Identity verification submitted"`
	VerificationRecordID string `json:"verification_record_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type IdentityStatusResponse struct {
	Status          string `json:"status" example:"pending"`
	RejectionReason string `json:"rejection_reason,omitempty" example:""`
}

// =========================
// Business Verification
// =========================

type BusinessVerificationRequest struct {
	BusinessName       string `form:"business_name" binding:"required"`
	BusinessPAN        string `form:"business_pan" binding:"required"`
	GSTNumber          string `form:"gst_number" binding:"required"`
	RegistrationNumber string `form:"registration_number" binding:"required"`
	Website            string `form:"website" binding:"required"`
	Country            string `form:"country" binding:"required"`
	BankName           string `form:"bank_name" binding:"required"`
	AccountHolderName  string `form:"account_holder_name" binding:"required"`
	BankAccountNumber  string `form:"bank_account_number" binding:"required"`
	BankIFSC           string `form:"bank_ifsc" binding:"required"`
}

type BusinessVerificationResponse struct {
	Message              string `json:"message" example:"Business verification submitted"`
	VerificationRecordID string `json:"verification_record_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// =========================
// Admin Queue
// =========================

type VerificationQueueItem struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type      string `json:"type" example:"identity"`
	Status    string `json:"status" example:"pending"`
	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
}

type VerificationQueueResponse struct {
	Queue []VerificationQueueItem `json:"queue"`
}

// =========================
// Admin Approve
// =========================

type AdminApproveRequest struct {
	Note string `json:"note" example:"Documents verified successfully"`
}

type AdminApproveResponse struct {
	Message string `json:"message" example:"Verification approved successfully"`
}

// =========================
// Admin Reject
// =========================

type AdminRejectRequest struct {
	Reason string `json:"reason" huma:"required" example:"Documents are blurry"`
}

type AdminRejectResponse struct {
	Message string `json:"message" example:"Verification rejected successfully"`
}

// =========================
// Resubmit Verification
// =========================

type ResubmitVerificationRequest struct {
	DocumentType string `json:"document_type" form:"document_type" huma:"required" example:"aadhaar"`
}

type ResubmitVerificationResponse struct {
	Message              string `json:"message"`
	VerificationRecordID string `json:"verification_record_id"`
}

// =========================
// Generic Verification Status
// =========================

type VerificationStatusResponse struct {
	Type            string `json:"type"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// =========================
// Huma Input/Output wrappers
// =========================

type GetIdentityVerificationStatusInput struct{}

type GetIdentityVerificationStatusOutput struct {
	Body IdentityStatusResponse
}

type GetVerificationStatusInput struct{}

type GetVerificationStatusOutput struct {
	Body VerificationStatusResponse
}

type GetVerificationQueueInput struct{}

type GetVerificationQueueOutput struct {
	Body VerificationQueueResponse
}

type ApproveVerificationInput struct {
	ID   string `path:"id" doc:"Verification Record ID"`
	Body AdminApproveRequest
}

type ApproveVerificationOutput struct {
	Body AdminApproveResponse
}

type RejectVerificationInput struct {
	ID   string `path:"id" doc:"Verification Record ID"`
	Body AdminRejectRequest
}

type RejectVerificationOutput struct {
	Body AdminRejectResponse
}

// File-upload endpoint wrappers (multipart via RawBody)
type SubmitIdentityVerificationInput struct {
	RawBody multipart.Form
}

type SubmitIdentityVerificationOutput struct {
	Body IdentityVerificationResponse
}

type SubmitBusinessVerificationInput struct {
	RawBody multipart.Form
}

type SubmitBusinessVerificationOutput struct {
	Body BusinessVerificationResponse
}

type ResubmitVerificationInput struct {
	RecordID string `path:"record_id"`
	RawBody  multipart.Form
}

type ResubmitVerificationOutput struct {
	Body ResubmitVerificationResponse
}
