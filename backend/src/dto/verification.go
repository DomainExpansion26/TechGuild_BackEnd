package dto

// =========================
// Individual Verification
// =========================

type IdentityVerificationRequest struct {
	GovtIDType   string `form:"govt_id_type" binding:"required"`
	GovtIDNumber string `form:"govt_id_number" binding:"required"`
}

type IdentityVerificationResponse struct {
	Message              string `json:"message"`
	VerificationRecordID string `json:"verification_record_id"`
}

type IdentityStatusResponse struct {
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// =========================
// Business Verification
// =========================

type BusinessVerificationRequest struct {
	BusinessName            string `form:"business_name" binding:"required"`
	BusinessPAN             string `form:"business_pan" binding:"required"`
	GSTNumber               string `form:"gst_number" binding:"required"`
	BankAccountNumber       string `form:"bank_account_number" binding:"required"`
	BankIFSC                string `form:"bank_ifsc" binding:"required"`
}

type BusinessVerificationResponse struct {
	Message              string `json:"message"`
	VerificationRecordID string `json:"verification_record_id"`
}

// =========================
// Admin Queue
// =========================

type VerificationQueueItem struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// =========================
// Admin Approve
// =========================

type AdminApproveResponse struct {
	Message string `json:"message"`
}

// =========================
// Admin Reject
// =========================

type AdminRejectRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type AdminRejectResponse struct {
	Message string `json:"message"`
}
// =========================
// Resubmit Verification
// =========================
type ResubmitVerificationRequest struct {
	DocumentType string `form:"document_type" binding:"required"`
}

type ResubmitVerificationResponse struct {
	Message              string `json:"message"`
	VerificationRecordID string `json:"verification_record_id"`
}