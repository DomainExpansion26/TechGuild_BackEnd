package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// MockPaymentProvider is a development/testing adapter that implements
// PaymentProvider without talking to any real provider.
//
// Use this while building and testing the end-to-end flow. Swap it with a
// real adapter (Razorpay/Stripe/Payoneer) via the provider factory in main.go.
type MockPaymentProvider struct {
	// SimulateFailure, when true, causes every money-moving call to fail.
	// Useful for testing error paths and reconciliation.
	SimulateFailure bool
}

func NewMockPaymentProvider() *MockPaymentProvider {
	return &MockPaymentProvider{SimulateFailure: false}
}

func (m *MockPaymentProvider) Name() string {
	return "mock"
}

// ============================================================
// Customer + payment methods
// ============================================================

func (m *MockPaymentProvider) CreateCustomer(userID, email, phone string) (string, error) {
	if m.SimulateFailure {
		return "", ErrProviderUnavailable
	}
	return "mock_cust_" + shortID(), nil
}

func (m *MockPaymentProvider) AttachPaymentMethod(providerCustomerID, providerMethodID string) error {
	if m.SimulateFailure {
		return ErrProviderUnavailable
	}
	if providerCustomerID == "" || providerMethodID == "" {
		return errors.New("mock: customer id and method id are required")
	}
	// Simulate failure for tokens prefixed with "fail_"
	if strings.HasPrefix(providerMethodID, "fail_") {
		return errors.New("mock: token rejected by provider")
	}
	return nil
}

func (m *MockPaymentProvider) DetachPaymentMethod(providerMethodID string) error {
	if m.SimulateFailure {
		return ErrProviderUnavailable
	}
	return nil
}

// ============================================================
// Payout fund accounts (money OUT destination)
// ============================================================

func (m *MockPaymentProvider) CreatePayoutFundAccount(req FundAccountRequest) (*FundAccountResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}

	switch req.Type {
	case "bank":
		if req.AccountNumber == "" || req.IFSC == "" {
			return nil, errors.New("mock: bank requires account_number and ifsc")
		}
		status := "verified"
		reason := ""
		if strings.HasPrefix(req.AccountNumber, "9999") {
			status = "failed"
			reason = "mock: account validation failed"
		}
		return &FundAccountResult{
			ProviderFundAccountID: "mock_fa_" + shortID(),
			Status:                status,
			MaskedAccount:         maskAccount(req.AccountNumber),
			FailureReason:         reason,
		}, nil

	case "upi":
		if req.VPA == "" {
			return nil, errors.New("mock: upi requires vpa")
		}
		status := "verified"
		if strings.HasPrefix(req.VPA, "fail") {
			status = "failed"
		}
		return &FundAccountResult{
			ProviderFundAccountID: "mock_fa_" + shortID(),
			Status:                status,
			MaskedAccount:         maskVPA(req.VPA),
		}, nil

	case "card", "wallet":
		if req.Last4 == "" {
			return nil, fmt.Errorf("mock: %s requires last4", req.Type)
		}
		return &FundAccountResult{
			ProviderFundAccountID: "mock_fa_" + shortID(),
			Status:                "verified",
			MaskedAccount:         "XXXX" + req.Last4,
		}, nil

	default:
		return nil, fmt.Errorf("mock: unsupported fund account type %q", req.Type)
	}
}

func (m *MockPaymentProvider) DeletePayoutFundAccount(providerFundAccountID string) error {
	if m.SimulateFailure {
		return ErrProviderUnavailable
	}
	return nil
}

// ============================================================
// Milestone funding (money IN)
// ============================================================

func (m *MockPaymentProvider) CreatePayment(req CreatePaymentRequest) (*PaymentResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if req.Amount <= 0 {
		return nil, errors.New("mock: amount must be positive")
	}
	return &PaymentResult{
		ProviderPaymentID: "mock_pay_" + shortID(),
		Status:            "captured", // mock me direct capture
		Amount:            req.Amount,
		Currency:          req.Currency,
	}, nil
}

func (m *MockPaymentProvider) CapturePayment(providerPaymentID string, amount int64) (*PaymentResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if providerPaymentID == "" {
		return nil, errors.New("mock: provider_payment_id required")
	}
	return &PaymentResult{
		ProviderPaymentID: providerPaymentID,
		Status:            "captured",
		Amount:            amount,
		Currency:          "INR",
	}, nil
}

// ============================================================
// Escrow
// ============================================================

func (m *MockPaymentProvider) CreateEscrowHold(req EscrowHoldRequest) (*EscrowResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if req.Amount <= 0 {
		return nil, errors.New("mock: escrow amount must be positive")
	}
	return &EscrowResult{
		ProviderEscrowID: "mock_esc_" + shortID(),
		Status:           "held",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

func (m *MockPaymentProvider) ReleaseEscrow(req EscrowReleaseRequest) (*EscrowResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if req.ProviderEscrowID == "" {
		return nil, errors.New("mock: provider_escrow_id required")
	}
	return &EscrowResult{
		ProviderEscrowID: req.ProviderEscrowID,
		Status:           "released",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

// ============================================================
// Payouts (money OUT)
// ============================================================

func (m *MockPaymentProvider) CreatePayout(req PayoutRequest) (*PayoutResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if req.Amount <= 0 {
		return nil, errors.New("mock: payout amount must be positive")
	}
	if req.ProviderFundAccountID == "" {
		return nil, errors.New("mock: payout requires fund_account_id")
	}
	return &PayoutResult{
		ProviderPayoutID: "mock_po_" + shortID(),
		Status:           "paid",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

// ============================================================
// Refunds
// ============================================================

func (m *MockPaymentProvider) RefundPayment(req RefundRequest) (*RefundResult, error) {
	if m.SimulateFailure {
		return nil, ErrProviderUnavailable
	}
	if req.Amount <= 0 {
		return nil, errors.New("mock: refund amount must be positive")
	}
	if req.ProviderPaymentID == "" {
		return nil, errors.New("mock: provider_payment_id required")
	}
	return &RefundResult{
		ProviderRefundID: "mock_rf_" + shortID(),
		Status:           "completed",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

// ============================================================
// Helpers
// ============================================================

func shortID() string {
	id := uuid.New().String()
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) > 12 {
		clean = clean[:12]
	}
	return clean
}

func maskAccount(account string) string {
	if len(account) <= 4 {
		return "XXXX"
	}
	return "XXXX" + account[len(account)-4:]
}
