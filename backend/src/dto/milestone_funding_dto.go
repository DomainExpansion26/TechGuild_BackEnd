package dto

import "time"

// ============================================================
// Fund Milestone
// ============================================================

type FundMilestoneBody struct {
	// Optional: specific saved payment method to use.
	PaymentMethodID string `json:"payment_method_id,omitempty" maxLength:"36" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Required: idempotency key to prevent double-charge
	IdempotencyKey string `json:"idempotency_key" required:"true" minLength:"5" maxLength:"255" example:"fund-ms-42-abc123"`
}

type FundMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
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
	ProviderOrderID      string `json:"provider_order_id,omitempty" example:"order_NqX7tR5wY3uA9p"`
	ClientKeyID          string `json:"client_key_id,omitempty" example:"rzp_test_NqX8kL2mZ4vB1c"`

	// Amounts (rupees) — populated in both variants
	MilestoneAmount float64 `json:"milestone_amount" example:"10000" doc:"Amount in rupees (major units). 10000 = ₹10,000.00 INR. NOT paise."`
	ClientFee       float64 `json:"client_fee" example:"500" doc:"Amount in rupees (major units). 500 = ₹500.00 INR. NOT paise."`
	FreelancerFee   float64 `json:"freelancer_fee" example:"500" doc:"Amount in rupees (major units). 500 = ₹500.00 INR. NOT paise."`
	ClientCharged   float64 `json:"client_charged" example:"10500" doc:"Amount in rupees (major units). 10500 = ₹10,500.00 INR. NOT paise."`
	FreelancerNet   float64 `json:"freelancer_net" example:"9500" doc:"Amount in rupees (major units). 9500 = ₹9,500.00 INR. NOT paise."`
	PlatformTotal   float64 `json:"platform_total" example:"1000" doc:"Amount in rupees (major units). 1000 = ₹1,000.00 INR. NOT paise."`

	// Raw minor units
	MilestoneAmountMinor int64 `json:"milestone_amount_minor" example:"1000000" doc:"Amount in paise (minor units). 1000000 = ₹10,000.00 INR"`
	ClientFeeMinor       int64 `json:"client_fee_minor" example:"50000" doc:"Amount in paise (minor units). 50000 = ₹500.00 INR"`
	FreelancerFeeMinor   int64 `json:"freelancer_fee_minor" example:"50000" doc:"Amount in paise (minor units). 50000 = ₹500.00 INR"`
	ClientChargedMinor   int64 `json:"client_charged_minor" example:"1050000" doc:"Amount in paise (minor units). 1050000 = ₹10,500.00 INR"`
	FreelancerNetMinor   int64 `json:"freelancer_net_minor" example:"950000" doc:"Amount in paise (minor units). 950000 = ₹9,500.00 INR"`
	PlatformTotalMinor   int64 `json:"platform_total_minor" example:"100000" doc:"Amount in paise (minor units). 100000 = ₹1,000.00 INR"`

	Currency string `json:"currency" example:"INR"`

	ClientFeeRateBps     int64 `json:"client_fee_rate_bps" example:"500" doc:"Rate in basis points. 500 = 5.00%"`
	FreelancerFeeRateBps int64 `json:"freelancer_fee_rate_bps" example:"500" doc:"Rate in basis points. 500 = 5.00%"`

	Message string `json:"message" example:"Milestone funded successfully. Funds held in escrow."`
}

type FundMilestoneOutput struct {
	Body FundMilestoneResponse
}

// ============================================================
// Confirm Funding (Razorpay async path)
// ============================================================

type ConfirmFundingBody struct {
	RazorpayPaymentID string `json:"razorpay_payment_id" required:"true" minLength:"1" maxLength:"100" example:"pay_NqX8kL2mZ4vB1c"`
	RazorpayOrderID   string `json:"razorpay_order_id" required:"true" minLength:"1" maxLength:"100" example:"order_NqX7tR5wY3uA9p"`
	RazorpaySignature string `json:"razorpay_signature" required:"true" minLength:"1" maxLength:"256" example:"9f2c1a4b7e8d3f6a0b5c9e2d4f7a1b8c3e6d9f2a5b8c1e4d7f0a3b6c9e2d5f8a"`
}

type ConfirmFundingInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        ConfirmFundingBody
}

type ConfirmFundingOutput struct {
	Body FundMilestoneResponse
}

// ============================================================
// Read Milestone Payment
// ============================================================

type GetMilestonePaymentInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type MilestonePaymentDetailResponse struct {
	PaymentID          string `json:"payment_id"`
	MilestonePaymentID string `json:"milestone_payment_id"`
	EscrowID           string `json:"escrow_id,omitempty"`

	QuestID          string `json:"quest_id"`
	QuestMilestoneID string `json:"quest_milestone_id"`

	MilestoneAmount float64 `json:"milestone_amount" example:"10000" doc:"Amount in rupees (major units). 10000 = ₹10,000.00 INR. NOT paise."`
	ClientFee       float64 `json:"client_fee" example:"500" doc:"Amount in rupees (major units). 500 = ₹500.00 INR. NOT paise."`
	FreelancerFee   float64 `json:"freelancer_fee" example:"500" doc:"Amount in rupees (major units). 500 = ₹500.00 INR. NOT paise."`
	ClientCharged   float64 `json:"client_charged" example:"10500" doc:"Amount in rupees (major units). 10500 = ₹10,500.00 INR. NOT paise."`
	FreelancerNet   float64 `json:"freelancer_net" example:"9500" doc:"Amount in rupees (major units). 9500 = ₹9,500.00 INR. NOT paise."`
	PlatformTotal   float64 `json:"platform_total" example:"1000" doc:"Amount in rupees (major units). 1000 = ₹1,000.00 INR. NOT paise."`

	MilestoneAmountMinor int64 `json:"milestone_amount_minor" example:"1000000" doc:"Amount in paise (minor units). 1000000 = ₹10,000.00 INR"`
	ClientFeeMinor       int64 `json:"client_fee_minor" example:"50000" doc:"Amount in paise (minor units). 50000 = ₹500.00 INR"`
	FreelancerFeeMinor   int64 `json:"freelancer_fee_minor" example:"50000" doc:"Amount in paise (minor units). 50000 = ₹500.00 INR"`
	ClientChargedMinor   int64 `json:"client_charged_minor" example:"1050000" doc:"Amount in paise (minor units). 1050000 = ₹10,500.00 INR"`
	FreelancerNetMinor   int64 `json:"freelancer_net_minor" example:"950000" doc:"Amount in paise (minor units). 950000 = ₹9,500.00 INR"`
	PlatformTotalMinor   int64 `json:"platform_total_minor" example:"100000" doc:"Amount in paise (minor units). 100000 = ₹1,000.00 INR"`

	Currency string `json:"currency" example:"INR"`

	ClientFeeRateBps     int64 `json:"client_fee_rate_bps" example:"500" doc:"Rate in basis points. 500 = 5.00%"`
	FreelancerFeeRateBps int64 `json:"freelancer_fee_rate_bps" example:"500" doc:"Rate in basis points. 500 = 5.00%"`

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
