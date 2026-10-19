package dto

import "time"

// ============================================================
// Fund Milestone
// ============================================================

type FundMilestoneBody struct {
	// Optional: specific saved payment method to use.
	PaymentMethodID string `json:"payment_method_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Required: idempotency key to prevent double-charge
	IdempotencyKey string `json:"idempotency_key" huma:"required,minLength=5,maxLength=255" example:"fund-ms-42-abc123"`
}

type FundMilestoneInput struct {
	QuestID     string `path:"quest_id" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        FundMilestoneBody
}

// FundMilestoneResponse — two variants based on RequiresConfirmation.
//
// Variant A (Mock, RequiresConfirmation=false):
//   - Status: "funded"
//   - Full amounts filled, escrow held immediately
//
// Variant B (Razorpay, RequiresConfirmation=true):
//   - Status: "funding_pending"
//   - ProviderOrderID set, ClientKeyID set
//   - Frontend must open Razorpay Checkout, then call /fund/confirm
type FundMilestoneResponse struct {
	PaymentID          string `json:"payment_id"`
	MilestonePaymentID string `json:"milestone_payment_id,omitempty"`
	EscrowID           string `json:"escrow_id,omitempty"`

	Status string `json:"status" example:"funded"`

	// RequiresConfirmation — if true, frontend must complete checkout
	RequiresConfirmation bool   `json:"requires_confirmation"`
	ProviderOrderID      string `json:"provider_order_id,omitempty" example:"order_xxx"`
	ClientKeyID          string `json:"client_key_id,omitempty" example:"rzp_test_xxx"`

	// Amounts (rupees) — populated in both variants
	MilestoneAmount float64 `json:"milestone_amount" example:"10000"`
	ClientFee       float64 `json:"client_fee" example:"500"`
	FreelancerFee   float64 `json:"freelancer_fee" example:"500"`
	ClientCharged   float64 `json:"client_charged" example:"10500"`
	FreelancerNet   float64 `json:"freelancer_net" example:"9500"`
	PlatformTotal   float64 `json:"platform_total" example:"1000"`

	// Raw minor units
	MilestoneAmountMinor int64 `json:"milestone_amount_minor" example:"1000000"`
	ClientFeeMinor       int64 `json:"client_fee_minor" example:"50000"`
	ClientChargedMinor   int64 `json:"client_charged_minor" example:"1050000"`

	Currency string `json:"currency" example:"INR"`

	ClientFeeRateBps     int64 `json:"client_fee_rate_bps" example:"500"`
	FreelancerFeeRateBps int64 `json:"freelancer_fee_rate_bps" example:"500"`

	Message string `json:"message" example:"Milestone funded successfully. Funds held in escrow."`
}

type FundMilestoneOutput struct {
	Body FundMilestoneResponse
}

// ============================================================
// Confirm Funding (Razorpay async path)
// ============================================================

type ConfirmFundingBody struct {
	RazorpayPaymentID string `json:"razorpay_payment_id" huma:"required" example:"pay_xxx"`
	RazorpayOrderID   string `json:"razorpay_order_id" huma:"required" example:"order_xxx"`
	RazorpaySignature string `json:"razorpay_signature" huma:"required" example:"<hmac>"`
}

type ConfirmFundingInput struct {
	QuestID     string `path:"quest_id" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        ConfirmFundingBody
}

type ConfirmFundingOutput struct {
	Body FundMilestoneResponse
}

// ============================================================
// Read Milestone Payment
// ============================================================

type GetMilestonePaymentInput struct {
	QuestID     string `path:"quest_id" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
}

type MilestonePaymentDetailResponse struct {
	PaymentID          string `json:"payment_id"`
	MilestonePaymentID string `json:"milestone_payment_id"`
	EscrowID           string `json:"escrow_id,omitempty"`

	QuestID          string `json:"quest_id"`
	QuestMilestoneID string `json:"quest_milestone_id"`

	MilestoneAmount float64 `json:"milestone_amount"`
	ClientFee       float64 `json:"client_fee"`
	FreelancerFee   float64 `json:"freelancer_fee"`
	ClientCharged   float64 `json:"client_charged"`
	FreelancerNet   float64 `json:"freelancer_net"`
	PlatformTotal   float64 `json:"platform_total"`

	MilestoneAmountMinor int64 `json:"milestone_amount_minor"`
	ClientFeeMinor       int64 `json:"client_fee_minor"`
	FreelancerFeeMinor   int64 `json:"freelancer_fee_minor"`
	ClientChargedMinor   int64 `json:"client_charged_minor"`
	FreelancerNetMinor   int64 `json:"freelancer_net_minor"`
	PlatformTotalMinor   int64 `json:"platform_total_minor"`

	Currency string `json:"currency" example:"INR"`

	ClientFeeRateBps     int64 `json:"client_fee_rate_bps"`
	FreelancerFeeRateBps int64 `json:"freelancer_fee_rate_bps"`

	Status string `json:"status" example:"funded"`

	PaymentStatus string     `json:"payment_status" example:"captured"`
	EscrowStatus  string     `json:"escrow_status" example:"held"`
	CapturedAt    *time.Time `json:"captured_at,omitempty"`
	EscrowHeldAt  *time.Time `json:"escrow_held_at,omitempty"`
	ReleasedAt    *time.Time `json:"released_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetMilestonePaymentOutput struct {
	Body MilestonePaymentDetailResponse
}
