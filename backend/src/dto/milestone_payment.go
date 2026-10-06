package dto

// ============ Fund Milestone ============

type FundMilestoneRequest struct {
	// Client sends their saved payment method ID (from earlier setup).
	PaymentMethodID string `json:"payment_method_id" huma:"required" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Idempotency key — client generates, ensures retries don't double-charge.
	IdempotencyKey string `json:"idempotency_key" huma:"required" example:"fund-ms-42-abc123"`
}

type FundMilestoneResponse struct {
	PaymentID          string `json:"payment_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MilestonePaymentID string `json:"milestone_payment_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	EscrowID           string `json:"escrow_id" example:"770e8400-e29b-41d4-a716-446655440002"`

	Status string `json:"status" example:"funded"`

	GrossAmount      int64  `json:"gross_amount" example:"500000"`
	PlatformFee      int64  `json:"platform_fee" example:"25000"`
	FreelancerAmount int64  `json:"freelancer_amount" example:"475000"`
	Currency         string `json:"currency" example:"INR"`

	FeeRateBps int64  `json:"fee_rate_bps" example:"500"`
	FeeType    string `json:"fee_type" example:"percentage"`

	Message string `json:"message" example:"Milestone funded successfully, funds held in escrow"`
}

// ============ Approve Milestone ============

// ============ Approve Milestone Payment ============

// ApproveMilestonePaymentRequest is distinct from the existing
// ApproveMilestoneRequest in milestone_dto.go — this one triggers the
// FINANCIAL release of escrow funds, not just the workflow state change.
// ============ Release Milestone Payment ============
// Client approves milestone AND triggers escrow release → freelancer payout.

type ReleaseMilestonePaymentRequest struct {
	IdempotencyKey string `json:"idempotency_key" huma:"required" example:"release-ms-42-xyz789"`
}

type ReleaseMilestonePaymentResponse struct {
	MilestonePaymentID string `json:"milestone_payment_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	Status             string `json:"status" example:"release_pending"`

	FreelancerAmount int64  `json:"freelancer_amount" example:"475000"`
	Currency         string `json:"currency" example:"INR"`

	Message string `json:"message" example:"Milestone released, payout initiated"`
}

// ============ Read milestone payment ============

type MilestonePaymentResponse struct {
	ID                  string `json:"id"`
	MilestoneID         string `json:"milestone_id"`
	ContractID          string `json:"contract_id"`
	ProjectID           string `json:"project_id"`
	PaymentID           string `json:"payment_id"`
	EscrowTransactionID string `json:"escrow_transaction_id,omitempty"`

	GrossAmount      int64  `json:"gross_amount"`
	PlatformFee      int64  `json:"platform_fee"`
	FreelancerAmount int64  `json:"freelancer_amount"`
	Currency         string `json:"currency"`

	FeeRateBps int64  `json:"fee_rate_bps"`
	FeeType    string `json:"fee_type"`
	Status     string `json:"status"`

	ApprovedAt string `json:"approved_at,omitempty"`
	ReleasedAt string `json:"released_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// ============ Huma I/O wrappers ============

type FundMilestoneInput struct {
	ProjectID   string `path:"project_id"`
	MilestoneID string `path:"milestone_id"`
	Body        FundMilestoneRequest
}
type FundMilestoneOutput struct {
	Body FundMilestoneResponse
}

type ReleaseMilestonePaymentInput struct {
	ProjectID   string `path:"project_id"`
	MilestoneID string `path:"milestone_id"`
	Body        ReleaseMilestonePaymentRequest
}
type ReleaseMilestonePaymentOutput struct {
	Body ReleaseMilestonePaymentResponse
}

type GetMilestonePaymentInput struct {
	ProjectID   string `path:"project_id"`
	MilestoneID string `path:"milestone_id"`
}
type GetMilestonePaymentOutput struct {
	Body MilestonePaymentResponse
}
