package dto

import "time"

// ============================================================
// Request Refund (client only)
// ============================================================

type RequestRefundBody struct {
	Reason         string `json:"reason" required:"true" minLength:"5" maxLength:"500" example:"Freelancer did not deliver as agreed"`
	IdempotencyKey string `json:"idempotency_key" required:"true" minLength:"5" maxLength:"255" example:"refund-ms-f4aa-001"`
}

type RequestRefundInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        RequestRefundBody
}

type RefundResponse struct {
	RefundID           string `json:"refund_id"`
	EscrowID           string `json:"escrow_id,omitempty"`
	MilestonePaymentID string `json:"milestone_payment_id"`

	QuestID     string `json:"quest_id"`
	MilestoneID string `json:"milestone_id"`

	Status string `json:"status" example:"completed"`

	// Refund amount = milestone amount only
	RefundAmount      float64 `json:"refund_amount" example:"12000" doc:"Amount in rupees (major units). 12000 = ₹12,000.00 INR. NOT paise."`
	RefundAmountMinor int64   `json:"refund_amount_minor" example:"1200000" doc:"Amount in paise (minor units). 1200000 = ₹12,000.00 INR"`

	// Info about non-refunded amounts
	ClientFeeRetained      float64 `json:"client_fee_retained" example:"600" doc:"Amount in rupees (major units). 600 = ₹600.00 INR. NOT paise."`
	ClientFeeRetainedMinor int64   `json:"client_fee_retained_minor" example:"60000" doc:"Amount in paise (minor units). 60000 = ₹600.00 INR"`

	Currency string `json:"currency" example:"INR"`

	EscrowStatus    string `json:"escrow_status" example:"refunded"`
	MilestoneStatus string `json:"milestone_status" example:"refunded"`

	Reason      string     `json:"reason"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Message string `json:"message" example:"Refund processed. Milestone amount credited back to your original payment method."`
}

type RequestRefundOutput struct {
	Body RefundResponse
}

// ============================================================
// Get Refund
// ============================================================

type GetRefundInput struct {
	QuestID     string `path:"id" format:"uuid"`
	MilestoneID string `path:"milestone_id" format:"uuid"`
}

type GetRefundOutput struct {
	Body RefundResponse
}
