package dto

import "time"

// ============================================================
// Get Wallet
// ============================================================

type GetWalletInput struct{}

type WalletTransactionItem struct {
	ID           string    `json:"id"`
	Type         string    `json:"type" example:"credit"`
	Amount       float64   `json:"amount" example:"11400"`
	AmountMinor  int64     `json:"amount_minor"`
	Currency     string    `json:"currency" example:"INR"`
	BalanceAfter float64   `json:"balance_after" example:"11400"`
	Reference    string    `json:"reference,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type WalletResponse struct {
	WalletID string `json:"wallet_id"`

	AvailableBalance float64 `json:"available_balance" example:"11400"`
	PendingBalance   float64 `json:"pending_balance" example:"0"`
	TotalEarned      float64 `json:"total_earned" example:"11400"`
	TotalWithdrawn   float64 `json:"total_withdrawn" example:"0"`

	AvailableBalanceMinor int64 `json:"available_balance_minor"`
	PendingBalanceMinor   int64 `json:"pending_balance_minor"`
	TotalEarnedMinor      int64 `json:"total_earned_minor"`
	TotalWithdrawnMinor   int64 `json:"total_withdrawn_minor"`

	Currency string `json:"currency" example:"INR"`

	RecentTransactions []WalletTransactionItem `json:"recent_transactions,omitempty"`
}

type GetWalletOutput struct {
	Body WalletResponse
}

// ============================================================
// List Wallet Transactions
// ============================================================

type ListWalletTransactionsInput struct {
	Limit  int `query:"limit" default:"20" doc:"Max items (default 20, max 100)"`
	Offset int `query:"offset" default:"0" doc:"Offset items"`
}

type ListWalletTransactionsResponse struct {
	Transactions []WalletTransactionItem `json:"transactions"`
	Total        int64                   `json:"total"`
	Limit        int                     `json:"limit"`
	Offset       int                     `json:"offset"`
}

type ListWalletTransactionsOutput struct {
	Body ListWalletTransactionsResponse
}
