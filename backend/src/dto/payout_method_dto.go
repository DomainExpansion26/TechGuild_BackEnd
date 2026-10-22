package dto

// ============ Payout Methods ============

// AddPayoutMethodRequest carries the raw inputs the user types.
// Sensitive fields (AccountNumber, VPA, etc.) are sent to the provider
// for tokenization and MUST NOT be persisted by TechGuild.
type AddPayoutMethodRequest struct {
	Provider string `json:"provider" huma:"required,enum=razorpay;stripe;payoneer" example:"razorpay"`

	Type    string `json:"type" huma:"required,enum=bank;upi;card;wallet" example:"bank"`
	Purpose string `json:"purpose" huma:"required,enum=payout" example:"payout"`

	// Bank fields (used when type = "bank")
	BankName      *string `json:"bank_name,omitempty" example:"HDFC Bank"`
	AccountHolder *string `json:"account_holder,omitempty" example:"John Doe"`
	AccountNumber *string `json:"account_number,omitempty" example:"1234567890123456"` // never stored
	IFSC          *string `json:"ifsc,omitempty" example:"HDFC0001234"`

	// UPI fields (used when type = "upi")
	VPA *string `json:"vpa,omitempty" example:"johndoe@okhdfc"`

	// Card/wallet fields (used when type = "card" or "wallet")
	Last4 *string `json:"last4,omitempty" example:"4242"`
	Brand *string `json:"brand,omitempty" example:"visa"`

	SetPrimary bool `json:"set_primary" example:"false"`
}

type PayoutMethodResponse struct {
	ID      string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type    string `json:"type" example:"bank"`
	Purpose string `json:"purpose" example:"payout"`
	Status  string `json:"status" example:"pending_verification"`

	Provider string `json:"provider" example:"razorpay"`

	// Bank fields
	BankName      string `json:"bank_name,omitempty" example:"HDFC Bank"`
	AccountHolder string `json:"account_holder,omitempty" example:"John Doe"`
	MaskedAccount string `json:"masked_account,omitempty" example:"XXXX1234"`
	IFSC          string `json:"ifsc,omitempty" example:"HDFC0001234"`

	// UPI
	VPA string `json:"vpa,omitempty" example:"johndoe@okhdfc"`

	// Card/wallet
	Last4 string `json:"last4,omitempty" example:"4242"`
	Brand string `json:"brand,omitempty" example:"visa"`

	IsPrimary bool `json:"is_primary" example:"false"`

	CreatedAt string `json:"created_at" example:"2026-09-30T12:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-09-30T12:00:00Z"`
}

type ListPayoutMethodsResponse struct {
	Methods []PayoutMethodResponse `json:"methods"`
}

type DeletePayoutMethodResponse struct {
	Message string `json:"message" example:"payout method deleted successfully"`
}

// ---------- Huma I/O wrappers ----------

type AddPayoutMethodInput struct {
	Body AddPayoutMethodRequest
}
type AddPayoutMethodOutput struct {
	Body PayoutMethodResponse
}

type ListPayoutMethodsInput struct{}
type ListPayoutMethodsOutput struct {
	Body ListPayoutMethodsResponse
}

type SetPrimaryPayoutMethodInput struct {
	ID string `path:"id" doc:"Payout method ID"`
}
type SetPrimaryPayoutMethodOutput struct {
	Body PayoutMethodResponse
}

type DeletePayoutMethodInput struct {
	ID string `path:"id" doc:"Payout method ID"`
}
type DeletePayoutMethodOutput struct {
	Body DeletePayoutMethodResponse
}
