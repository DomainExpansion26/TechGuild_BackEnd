package razorpay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"techguild-backend/src/services"
)

// Adapter implements services.PaymentProvider for Razorpay.
//
// Capability matrix (as of Razorpay + RazorpayX public APIs):
//
//	Payments:        ✅ (Orders + Capture + Refund)
//	Escrow:          ❌ (no provider API — DB-only synthetic state)
//	Payouts:         ❌ (needs RazorpayX — not activated yet)
//	FIRC:            ❌ (future)
type Adapter struct {
	client        *Client
	accountNumber string
}

// NewAdapter — payments + refunds only (no RazorpayX account number).
func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

// NewAdapterWithPayouts — payments + refunds + RazorpayX payouts.
// accountNumber = RazorpayX source account (paisa yahase jaayega).
func NewAdapterWithPayouts(client *Client, accountNumber string) *Adapter {
	return &Adapter{
		client:        client,
		accountNumber: accountNumber,
	}
}

func (a *Adapter) Name() string {
	return "razorpay"
}

// ============================================================
// Customer
// ============================================================

func (a *Adapter) CreateCustomer(userID, email, phone string) (string, error) {
	data := map[string]interface{}{
		"fail_existing": "0",
		"notes": map[string]string{
			"techguild_user_id": userID,
		},
	}
	if email != "" {
		data["email"] = email
	}
	if phone != "" {
		data["contact"] = phone
	}

	body, err := a.client.SDK().Customer.Create(data, nil)
	if err != nil {
		return "", fmt.Errorf("razorpay customer create failed: %w", err)
	}

	id := toStr(body["id"])
	if id == "" {
		return "", errors.New("razorpay: customer id missing in response")
	}
	return id, nil
}

// AttachPaymentMethod — Razorpay tokenizes via Checkout SDK client-side.
// No server-side attach is required.
func (a *Adapter) AttachPaymentMethod(providerCustomerID, providerMethodID string) error {
	return nil
}

// DetachPaymentMethod — no-op for Razorpay (tokens expire naturally).
func (a *Adapter) DetachPaymentMethod(providerMethodID string) error {
	return nil
}

// ============================================================
// Milestone funding (money IN) — Order create
// ============================================================
//
// Creates a Razorpay Order. The actual payment happens when the client
// completes Checkout on the frontend. Order status returned = "created".
//
// After the client pays, a webhook (payment.captured) or an explicit
// /fund/confirm call updates the local state.
func (a *Adapter) CreatePayment(req services.CreatePaymentRequest) (*services.PaymentResult, error) {
	if req.Amount <= 0 {
		return nil, errors.New("razorpay: amount must be positive")
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}

	receipt := fmt.Sprintf("ms_%s_%d", req.QuestMilestoneID, time.Now().Unix())

	data := map[string]interface{}{
		"amount":   req.Amount, // paise
		"currency": req.Currency,
		"receipt":  receipt,
		"notes": map[string]string{
			"techguild_user_id":         req.UserID,
			"techguild_quest_id":        req.QuestID,
			"techguild_milestone_id":    req.QuestMilestoneID,
			"techguild_idempotency_key": req.IdempotencyKey,
		},
	}

	// Pass idempotency key as extra header (Razorpay respects X-Razorpay-Idempotency-Key)
	headers := map[string]string{
		"X-Razorpay-Idempotency-Key": req.IdempotencyKey,
	}

	body, err := a.client.SDK().Order.Create(data, headers)
	if err != nil {
		return nil, fmt.Errorf("razorpay order create failed: %w", err)
	}

	orderID := toStr(body["id"])
	status := toStr(body["status"])
	if orderID == "" {
		return nil, errors.New("razorpay: order id missing in response")
	}
	if status == "" {
		status = "created"
	}

	return &services.PaymentResult{
		ProviderPaymentID:    orderID,
		Status:               status, // "created"
		Amount:               req.Amount,
		Currency:             req.Currency,
		RequiresConfirmation: true, // client must complete Checkout
		ClientKeyID:          a.client.KeyID(),
	}, nil
}

// CapturePayment captures an authorized payment.
// For Razorpay this is called after Checkout with the razorpay_payment_id.
func (a *Adapter) CapturePayment(providerPaymentID string, amount int64) (*services.PaymentResult, error) {
	if providerPaymentID == "" {
		return nil, errors.New("razorpay: payment id required")
	}
	if amount <= 0 {
		return nil, errors.New("razorpay: capture amount must be positive")
	}

	// SDK signature: Capture(paymentID string, amount int, data map, headers map)
	body, err := a.client.SDK().Payment.Capture(
		providerPaymentID,
		int(amount),
		map[string]interface{}{},
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("razorpay capture failed: %w", err)
	}

	status := toStr(body["status"])
	amt := toInt64(body["amount"])
	currency := toStr(body["currency"])
	if currency == "" {
		currency = "INR"
	}

	return &services.PaymentResult{
		ProviderPaymentID: providerPaymentID,
		Status:            status,
		Amount:            amt,
		Currency:          currency,
	}, nil
}

// ============================================================
// Escrow — synthetic (Razorpay has no escrow API)
// ============================================================
//
// Escrow is tracked as DB state in our system. Real money is held in the
// TechGuild Razorpay account until release. Release = our internal status
// change + later payout to freelancer.

func (a *Adapter) CreateEscrowHold(req services.EscrowHoldRequest) (*services.EscrowResult, error) {
	if req.Amount <= 0 {
		return nil, errors.New("razorpay: escrow amount must be positive")
	}

	// Synthetic ID — no provider call.
	syntheticID := "razorpay_escrow_" + shortID()

	return &services.EscrowResult{
		ProviderEscrowID: syntheticID,
		Status:           "held",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

func (a *Adapter) ReleaseEscrow(req services.EscrowReleaseRequest) (*services.EscrowResult, error) {
	if req.ProviderEscrowID == "" {
		return nil, errors.New("razorpay: provider_escrow_id required")
	}
	// No-op — escrow is DB state. Money stays in Razorpay account.
	return &services.EscrowResult{
		ProviderEscrowID: req.ProviderEscrowID,
		Status:           "released",
		Amount:           req.Amount,
		Currency:         req.Currency,
	}, nil
}

// ============================================================
// Payouts — RazorpayX (direct HTTP — SDK lacks these endpoints)
// ============================================================
//
// RazorpayX flow:
//   1. Create Contact (beneficiary)
//   2. Create Fund Account (bank/UPI) under Contact
//   3. Create Payout from source account
//
// SDK v1.3.3 me Contact/FundAccount/CreatePayout methods nahi hain,
// isliye hum direct REST calls kar rahe hain.

func (a *Adapter) CreatePayoutFundAccount(req services.FundAccountRequest) (*services.FundAccountResult, error) {
	if a.accountNumber == "" {
		return nil, services.ErrProviderUnsupported
	}

	// 1. Create Contact
	name := req.AccountHolder
	if name == "" {
		name = "TechGuild Beneficiary"
	}

	contactData := map[string]interface{}{
		"name":         name,
		"type":         "vendor",
		"reference_id": "tg_fa_" + shortID(),
	}

	contactResp, err := a.razorpayXPost("/contacts", contactData, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpayx contact create failed: %w", err)
	}
	contactID := toStr(contactResp["id"])
	if contactID == "" {
		return nil, errors.New("razorpayx: contact id missing")
	}

	// 2. Create Fund Account under Contact
	faData := map[string]interface{}{
		"contact_id":   contactID,
		"account_type": req.Type,
	}

	switch req.Type {
	case "bank":
		if req.AccountNumber == "" || req.IFSC == "" {
			return nil, errors.New("razorpayx: bank requires account_number and ifsc")
		}
		faData["bank_account"] = map[string]interface{}{
			"name":           name,
			"ifsc":           req.IFSC,
			"account_number": req.AccountNumber,
		}
	case "upi":
		if req.VPA == "" {
			return nil, errors.New("razorpayx: upi requires vpa")
		}
		faData["vpa"] = map[string]interface{}{
			"address": req.VPA,
		}
	default:
		return nil, fmt.Errorf("razorpayx: unsupported fund account type %q", req.Type)
	}

	faResp, err := a.razorpayXPost("/fund_accounts", faData, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpayx fund account create failed: %w", err)
	}

	fundAccountID := toStr(faResp["id"])
	if fundAccountID == "" {
		return nil, errors.New("razorpayx: fund account id missing")
	}

	// Normalize RazorpayX status to our internal status
	status := toStr(faResp["status"])
	switch status {
	case "":
		status = "pending_verification"
	case "active":
		status = "verified"
	}

	var masked string
	if req.Type == "bank" && req.AccountNumber != "" {
		masked = maskAccount(req.AccountNumber)
	} else if req.Type == "upi" && req.VPA != "" {
		masked = maskVPA(req.VPA)
	}

	return &services.FundAccountResult{
		ProviderFundAccountID: fundAccountID,
		Status:                status,
		MaskedAccount:         masked,
	}, nil
}

func (a *Adapter) DeletePayoutFundAccount(providerFundAccountID string) error {
	// RazorpayX does not expose a delete API for fund accounts.
	return nil
}

func (a *Adapter) CreatePayout(req services.PayoutRequest) (*services.PayoutResult, error) {
	if a.accountNumber == "" {
		return nil, services.ErrProviderUnsupported
	}
	if req.Amount <= 0 {
		return nil, errors.New("razorpayx: amount must be positive")
	}
	if req.ProviderFundAccountID == "" {
		return nil, errors.New("razorpayx: fund_account_id required")
	}

	payoutData := map[string]interface{}{
		"account_number":       a.accountNumber,
		"fund_account_id":      req.ProviderFundAccountID,
		"amount":               req.Amount,
		"currency":             req.Currency,
		"mode":                 "IMPS",
		"purpose":              "payout",
		"queue_if_low_balance": true,
		"reference_id":         "tg_po_" + shortID(),
		"narration":            req.Description,
		"notes": map[string]string{
			"techguild_user_id":         req.UserID,
			"techguild_idempotency_key": req.IdempotencyKey,
		},
	}

	headers := map[string]string{
		"X-Payout-Idempotency": req.IdempotencyKey,
	}

	body, err := a.razorpayXPost("/payouts", payoutData, headers)
	if err != nil {
		return nil, fmt.Errorf("razorpayx payout create failed: %w", err)
	}

	payoutID := toStr(body["id"])
	status := toStr(body["status"])
	amt := toInt64(body["amount"])
	currency := toStr(body["currency"])

	switch status {
	case "processed":
		status = "paid"
	case "queued", "pending":
		status = "pending"
	case "processing":
		status = "processing"
	case "failed", "cancelled", "reversed":
		status = "failed"
	}

	if payoutID == "" {
		return nil, errors.New("razorpayx: payout id missing")
	}
	if currency == "" {
		currency = "INR"
	}

	return &services.PayoutResult{
		ProviderPayoutID: payoutID,
		Status:           status,
		Amount:           amt,
		Currency:         currency,
		FailureReason:    toStr(body["failure_reason"]),
	}, nil
}

// ============================================================
// razorpayXPost — direct HTTP POST to RazorpayX API.
// SDK v1.3.3 me Contact / FundAccount / Payout.Create nahi hain.
// ============================================================

func (a *Adapter) razorpayXPost(
	path string,
	body map[string]interface{},
	headers map[string]string,
) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://api.razorpay.com/v1" + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.client.KeyID(), a.client.KeySecret())
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("razorpayx HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("razorpayx response parse: %w", err)
	}
	return result, nil
}

// ============================================================
// Refunds
// ============================================================

func (a *Adapter) RefundPayment(req services.RefundRequest) (*services.RefundResult, error) {
	if req.Amount <= 0 {
		return nil, errors.New("razorpay: refund amount must be positive")
	}
	if req.ProviderPaymentID == "" {
		return nil, errors.New("razorpay: payment id required")
	}

	// SDK signature: Refund(paymentID string, amount int, data map, headers map)
	// Note: "amount" is passed separately, not inside data map
	data := map[string]interface{}{
		"notes": map[string]string{
			"reason":                    req.Reason,
			"techguild_idempotency_key": req.IdempotencyKey,
		},
	}

	headers := map[string]string{
		"X-Razorpay-Idempotency-Key": req.IdempotencyKey,
	}

	body, err := a.client.SDK().Payment.Refund(
		req.ProviderPaymentID,
		int(req.Amount),
		data,
		headers,
	)
	if err != nil {
		return nil, fmt.Errorf("razorpay refund failed: %w", err)
	}

	refundID := toStr(body["id"])
	status := toStr(body["status"])
	amt := toInt64(body["amount"])

	// Razorpay status: "pending" | "processed" | "failed"
	// We map "processed" to "completed" for our internal status.
	switch status {
	case "processed":
		status = "completed"
	case "pending":
		status = "processing"
	}

	return &services.RefundResult{
		ProviderRefundID: refundID,
		Status:           status,
		Amount:           amt,
		Currency:         "INR",
	}, nil
}

func (a *Adapter) GetPayment(providerPaymentID string) (*services.PaymentResult, error) {
	body, err := a.client.SDK().Payment.Fetch(providerPaymentID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay fetch payment failed: %w", err)
	}

	status := toStr(body["status"])
	amt := toInt64(body["amount"])
	currency := toStr(body["currency"])

	if status == "captured" {
		return &services.PaymentResult{
			ProviderPaymentID: providerPaymentID,
			Status:            "captured",
			Amount:            amt,
			Currency:          currency,
		}, nil
	}

	return nil, errors.New("payment not captured yet")
}
