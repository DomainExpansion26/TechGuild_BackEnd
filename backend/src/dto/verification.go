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

type AdminApproveRequest struct {
	GovtIDHash string `json:"govt_id_hash"`
}

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

// =========================
// Generic Verification Status
// =========================
type VerificationStatusResponse struct {
	Type            string `json:"type"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}