package dto

import "time"

// ============================================================
// Request Withdrawal
// ============================================================

type RequestWithdrawalBody struct {
	// Amount in rupees (float for frontend)
	Amount float64 `json:"amount" huma:"required,minimum=1" example:"5000" doc:"Withdrawal amount in rupees"`

	// Which payout method to use (must be verified + active)
	PayoutMethodID string `json:"payout_method_id" huma:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`

	IdempotencyKey string `json:"idempotency_key" huma:"required,minLength=5,maxLength=255" example:"wd-2026-abc123"`
}

type RequestWithdrawalInput struct {
	Body RequestWithdrawalBody
}

type RequestWithdrawalResponse struct {
	PayoutID   string `json:"payout_id"`
	WalletTxID string `json:"wallet_transaction_id"`

	Amount      float64 `json:"amount" example:"5000"`
	AmountMinor int64   `json:"amount_minor"`
	Currency    string  `json:"currency" example:"INR"`

	Status string `json:"status" example:"paid"`

	ProviderPayoutID string `json:"provider_payout_id,omitempty"`

	// Wallet snapshot after debit
	WalletAvailableBalance      float64 `json:"wallet_available_balance" example:"6400"`
	WalletAvailableBalanceMinor int64   `json:"wallet_available_balance_minor"`

	Message string `json:"message" example:"Withdrawal successful. Funds on the way to your account."`
}

type RequestWithdrawalOutput struct {
	Body RequestWithdrawalResponse
}

// ============================================================
// List Withdrawals
// ============================================================

type ListWithdrawalsInput struct {
	Limit  int `query:"limit" default:"20" doc:"Max items (default 20, max 100)"`
	Offset int `query:"offset" default:"0" doc:"Offset items"`
}

type WithdrawalItem struct {
	ID string `json:"id"`

	Amount      float64 `json:"amount"`
	AmountMinor int64   `json:"amount_minor"`
	Currency    string  `json:"currency" example:"INR"`

	Status string `json:"status" example:"paid"`

	ProviderPayoutID string `json:"provider_payout_id,omitempty"`

	PayoutMethodID string `json:"payout_method_id"`

	FailureReason string     `json:"failure_reason,omitempty"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ListWithdrawalsResponse struct {
	Withdrawals []WithdrawalItem `json:"withdrawals"`
	Total       int64            `json:"total"`
	Limit       int              `json:"limit"`
	Offset      int              `json:"offset"`
}

type ListWithdrawalsOutput struct {
	Body ListWithdrawalsResponse
}

// ============================================================
// Get Withdrawal
// ============================================================

type GetWithdrawalInput struct {
	PayoutID string `path:"payout_id" doc:"Payout (withdrawal) ID"`
}

type GetWithdrawalOutput struct {
	Body WithdrawalItem
}
