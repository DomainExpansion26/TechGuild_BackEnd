package dto

import "time"

// ============================================================
// Submit for release (freelancer)
// ============================================================

type SubmitForReleaseBody struct {
	Note string `json:"note,omitempty" example:"All tasks completed and tested"`
}

type SubmitForReleaseInput struct {
	QuestID     string `path:"quest_id" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        SubmitForReleaseBody
}

type SubmitForReleaseOutput struct {
	Body MilestoneDetailResponse
}

// ============================================================
// Release (client)
// ============================================================

type ReleaseMilestoneBody struct {
	IdempotencyKey string `json:"idempotency_key" huma:"required,minLength=5,maxLength=255" example:"release-ms-42-xyz"`
	Note           string `json:"note,omitempty" example:"Great work, releasing payment"`
}

type ReleaseMilestoneInput struct {
	QuestID     string `path:"quest_id" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        ReleaseMilestoneBody
}

type ReleaseMilestoneResponse struct {
	MilestonePaymentID string `json:"milestone_payment_id"`
	EscrowID           string `json:"escrow_id"`
	PayoutWalletTxID   string `json:"wallet_transaction_id"`

	MilestoneID string `json:"milestone_id"`
	QuestID     string `json:"quest_id"`

	Status string `json:"status" example:"released"`

	// Amounts (rupees)
	MilestoneAmount float64 `json:"milestone_amount" example:"12000"`
	PlatformFee     float64 `json:"platform_fee" example:"600"`
	FreelancerNet   float64 `json:"freelancer_net" example:"11400"`

	// Minor units
	MilestoneAmountMinor int64 `json:"milestone_amount_minor"`
	PlatformFeeMinor     int64 `json:"platform_fee_minor"`
	FreelancerNetMinor   int64 `json:"freelancer_net_minor"`

	Currency string `json:"currency" example:"INR"`

	// Wallet snapshot after credit
	WalletAvailableBalance float64 `json:"wallet_available_balance" example:"11400"`

	ReleasedAt time.Time `json:"released_at"`

	Message string `json:"message" example:"Milestone released. Funds credited to freelancer wallet."`
}

type ReleaseMilestoneOutput struct {
	Body ReleaseMilestoneResponse
}
