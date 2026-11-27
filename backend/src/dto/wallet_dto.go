package dto

import "time"

// ============================================================
// Get Wallet
// ============================================================

type GetWalletInput struct{}

type WalletTransactionItem struct {
	ID           string    `json:"id"`
	Type         string    `json:"type" example:"credit"`
	Amount       float64   `json:"amount" example:"11400" doc:"Amount in rupees (major units). 10000 = ₹10,000.00 INR. NOT paise."`
	AmountMinor  int64     `json:"amount_minor" example:"1140000" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`
	Currency     string    `json:"currency" example:"INR"`
	BalanceAfter float64   `json:"balance_after" example:"11400" doc:"Amount in rupees (major units). 10000 = ₹10,000.00 INR. NOT paise."`
	Reference    string    `json:"reference,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type WalletResponse struct {
	WalletID string `json:"wallet_id"`

	AvailableBalance float64 `json:"available_balance" example:"11400" doc:"Wallet balance in rupees (major units). Paired with *_minor for paise."`
	PendingBalance   float64 `json:"pending_balance" example:"0" doc:"Wallet balance in rupees (major units). Paired with *_minor for paise."`
	TotalEarned      float64 `json:"total_earned" example:"11400" doc:"Wallet balance in rupees (major units). Paired with *_minor for paise."`
	TotalWithdrawn   float64 `json:"total_withdrawn" example:"0" doc:"Wallet balance in rupees (major units). Paired with *_minor for paise."`

	AvailableBalanceMinor int64 `json:"available_balance_minor" example:"1140000" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`
	PendingBalanceMinor   int64 `json:"pending_balance_minor" example:"0" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`
	TotalEarnedMinor      int64 `json:"total_earned_minor" example:"1140000" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`
	TotalWithdrawnMinor   int64 `json:"total_withdrawn_minor" example:"0" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`

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
	Limit  int `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Max items (default 20, max 100)"`
	Offset int `query:"offset" default:"0" minimum:"0" doc:"Offset items"`
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
