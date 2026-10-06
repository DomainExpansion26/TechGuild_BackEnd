package services

import "errors"

// PaymentProvider is the provider-neutral interface that all payment
// adapters (Razorpay, Stripe, Payoneer) must implement.
//
// Core services (PaymentMethodService, PayoutMethodService) depend only
// on this interface, never on provider SDKs directly.

// ErrNotSupported — return by adapters for capabilities they don't have.
var (
	ErrProviderUnsupported = errors.New("operation not supported by provider")
	ErrProviderTimeout     = errors.New("payment provider timeout")
	ErrProviderUnavailable = errors.New("payment provider unavailable")
)

type PaymentProvider interface {
	Name() string

	// ---------- Customer + payment methods ----------
	CreateCustomer(userID, email, phone string) (string, error)
	AttachPaymentMethod(providerCustomerID, providerMethodID string) error
	DetachPaymentMethod(providerMethodID string) error

	// ---------- Payout methods (money OUT destination) ----------
	CreatePayoutFundAccount(req FundAccountRequest) (*FundAccountResult, error)
	DeletePayoutFundAccount(providerFundAccountID string) error

	// ---------- Milestone funding (money IN) ----------
	CreatePayment(req CreatePaymentRequest) (*PaymentResult, error)
	CapturePayment(providerPaymentID string, amount int64) (*PaymentResult, error)

	// ---------- Escrow ----------
	CreateEscrowHold(req EscrowHoldRequest) (*EscrowResult, error)
	ReleaseEscrow(req EscrowReleaseRequest) (*EscrowResult, error)

	// ---------- Payouts (money OUT) ----------
	CreatePayout(req PayoutRequest) (*PayoutResult, error)

	// ---------- Refunds ----------
	RefundPayment(req RefundRequest) (*RefundResult, error)
}

// ---------- Request/result types for milestone flow ----------

type CreatePaymentRequest struct {
	UserID         string
	ProjectID      string
	ContractID     string
	MilestoneID    string
	Amount         int64 // minor units
	Currency       string
	Description    string
	IdempotencyKey string
}

type PaymentResult struct {
	ProviderPaymentID string
	Status            string // pending / authorized / captured / failed
	Amount            int64
	Currency          string
}

type EscrowHoldRequest struct {
	ProviderPaymentID string
	Amount            int64
	Currency          string
	IdempotencyKey    string
}

type EscrowReleaseRequest struct {
	ProviderEscrowID string
	Amount           int64
	Currency         string
	IdempotencyKey   string
}

type EscrowResult struct {
	ProviderEscrowID string
	Status           string // held / released / failed
	Amount           int64
	Currency         string
}

type PayoutRequest struct {
	UserID                string
	Amount                int64
	Currency              string
	ProviderFundAccountID string
	IdempotencyKey        string
	Description           string
}

type PayoutResult struct {
	ProviderPayoutID string
	Status           string // pending / processing / paid / failed
	Amount           int64
	Currency         string
	FailureReason    string
}

type RefundRequest struct {
	ProviderPaymentID string
	Amount            int64
	Currency          string
	Reason            string
	IdempotencyKey    string
}

type RefundResult struct {
	ProviderRefundID string
	Status           string // pending / processing / completed / failed
	Amount           int64
	Currency         string
}

// Existing types (fund account — client PM + payout methods)
type FundAccountRequest struct {
	Type          string // bank, upi, card, wallet
	AccountHolder string
	AccountNumber string
	IFSC          string
	VPA           string
	Last4         string
	Brand         string
}

type FundAccountResult struct {
	ProviderFundAccountID string
	Status                string
	MaskedAccount         string
	FailureReason         string
}
