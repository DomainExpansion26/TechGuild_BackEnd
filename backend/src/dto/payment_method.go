package dto

// ============ Client Payment Methods ============

type AddClientPaymentMethodRequest struct {
	// Provider token from client-side SDK (Razorpay/Stripe checkout).
	// TechGuild never sees raw card/UPI data.
	Provider         string `json:"provider" huma:"required,enum=razorpay;stripe" example:"razorpay"`
	ProviderMethodID string `json:"provider_method_id" huma:"required" example:"pm_xxxxxxxxxxxx"`

	Type string `json:"type" huma:"required,enum=card;upi;netbanking" example:"card"`

	// Optional display metadata, client can send but server may override
	// from provider response.
	Last4    *string `json:"last4,omitempty" huma:"maxLength=4" example:"4242"`
	Brand    *string `json:"brand,omitempty" example:"visa"`
	ExpMonth *int    `json:"exp_month,omitempty" example:"12"`
	ExpYear  *int    `json:"exp_year,omitempty" example:"2028"`

	VPAMasked *string `json:"vpa_masked,omitempty" example:"user@okhdfc"`

	SetPrimary bool `json:"set_primary" example:"true"`
}

type ConfirmClientPaymentMethodRequest struct {
	// Some providers need an explicit confirm/verify step (e.g. OTP, mandate).
	// Payload is provider-specific and passed through as-is.
	ProviderPayload map[string]any `json:"provider_payload,omitempty"`
}

type ClientPaymentMethodResponse struct {
	ID     string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type   string `json:"type" example:"card"`
	Status string `json:"status" example:"active"`

	Provider string `json:"provider" example:"razorpay"`

	Last4    string `json:"last4,omitempty" example:"4242"`
	Brand    string `json:"brand,omitempty" example:"visa"`
	ExpMonth int    `json:"exp_month,omitempty" example:"12"`
	ExpYear  int    `json:"exp_year,omitempty" example:"2028"`

	VPAMasked string `json:"vpa_masked,omitempty" example:"user@okhdfc"`

	IsPrimary bool `json:"is_primary" example:"true"`

	CreatedAt string `json:"created_at" example:"2026-09-30T12:00:00Z"`
}

type ListClientPaymentMethodsResponse struct {
	Methods []ClientPaymentMethodResponse `json:"methods"`
}

type DeleteClientPaymentMethodResponse struct {
	Message string `json:"message" example:"payment method deleted successfully"`
}

// ---------- Huma I/O wrappers ----------

type AddClientPaymentMethodInput struct {
	Body AddClientPaymentMethodRequest
}
type AddClientPaymentMethodOutput struct {
	Body ClientPaymentMethodResponse
}

type ListClientPaymentMethodsInput struct{}
type ListClientPaymentMethodsOutput struct {
	Body ListClientPaymentMethodsResponse
}

type ConfirmClientPaymentMethodInput struct {
	ID   string `path:"id" doc:"Payment method ID"`
	Body ConfirmClientPaymentMethodRequest
}
type ConfirmClientPaymentMethodOutput struct {
	Body ClientPaymentMethodResponse
}

type SetPrimaryClientPaymentMethodInput struct {
	ID string `path:"id" doc:"Payment method ID"`
}
type SetPrimaryClientPaymentMethodOutput struct {
	Body ClientPaymentMethodResponse
}

type DeleteClientPaymentMethodInput struct {
	ID string `path:"id" doc:"Payment method ID"`
}
type DeleteClientPaymentMethodOutput struct {
	Body DeleteClientPaymentMethodResponse
}
