package dto

import "time"

// ============================================================
// Submit for release (freelancer)
// ============================================================

type SubmitForReleaseBody struct {
	Note string `json:"note,omitempty" maxLength:"2000" example:"All tasks completed and tested"`
}

type SubmitForReleaseInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        SubmitForReleaseBody
}

type SubmitForReleaseOutput struct {
	Body MilestoneDetailResponse
}

// ============================================================
// Release (client)
// ============================================================

type ReleaseMilestoneBody struct {
	IdempotencyKey string `json:"idempotency_key" required:"true" minLength:"5" maxLength:"255" example:"release-ms-42-xyz"`
	Note           string `json:"note,omitempty" maxLength:"2000" example:"Great work, releasing payment"`
}

type ReleaseMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
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
	MilestoneAmount float64 `json:"milestone_amount" example:"12000" doc:"Amount in rupees (major units). 12000 = ₹12,000.00 INR. NOT paise."`
	PlatformFee     float64 `json:"platform_fee" example:"600" doc:"Amount in rupees (major units). 600 = ₹600.00 INR. NOT paise."`
	FreelancerNet   float64 `json:"freelancer_net" example:"11400" doc:"Amount in rupees (major units). 11400 = ₹11,400.00 INR. NOT paise."`

	// Minor units
	MilestoneAmountMinor int64 `json:"milestone_amount_minor" example:"1200000" doc:"Amount in paise (minor units). 1200000 = ₹12,000.00 INR"`
	PlatformFeeMinor     int64 `json:"platform_fee_minor" example:"60000" doc:"Amount in paise (minor units). 60000 = ₹600.00 INR"`
	FreelancerNetMinor   int64 `json:"freelancer_net_minor" example:"1140000" doc:"Amount in paise (minor units). 1140000 = ₹11,400.00 INR"`

	Currency string `json:"currency" example:"INR"`

	// Wallet snapshot after credit
	WalletAvailableBalance float64 `json:"wallet_available_balance" example:"11400" doc:"Amount in rupees (major units). 11400 = ₹11,400.00 INR. NOT paise."`
	WalletPendingBalance   float64 `json:"wallet_pending_balance" example:"11400" doc:"Amount in rupees (major units). 11400 = ₹11,400.00 INR. NOT paise."`

	// Wallet snapshot in minor units
	WalletAvailableBalanceMinor int64 `json:"wallet_available_balance_minor" example:"1140000" doc:"Amount in paise (minor units). 1140000 = ₹11,400.00 INR"`
	WalletPendingBalanceMinor   int64 `json:"wallet_pending_balance_minor" example:"1140000" doc:"Amount in paise (minor units). 1140000 = ₹11,400.00 INR"`

	ReleasedAt time.Time `json:"released_at"`

	Message string `json:"message" example:"Milestone released. Funds credited to freelancer wallet."`
}

type ReleaseMilestoneOutput struct {
	Body ReleaseMilestoneResponse
}
