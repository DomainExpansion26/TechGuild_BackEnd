package razorpay

// ============================================================
// Razorpay-specific request/response shapes
// ============================================================
//
// Ye types sirf razorpay package ke andar use hote hain.
// Service layer ko inka pata nahi — wo provider-neutral types use karti hai.
//
// SDK mostly map[string]interface{} use karta hai, par humne kuch
// common shapes define kar rakhe hain documentation + future use ke liye.

// ============================================================
// Order (payment intent)
// ============================================================

// OrderResponse represents Razorpay Order object.
// Created via CreatePayment() → maps to services.PaymentResult.
type OrderResponse struct {
	ID         string            `json:"id"`     // order_xxx
	Entity     string            `json:"entity"` // "order"
	Amount     int64             `json:"amount"` // paise
	AmountPaid int64             `json:"amount_paid"`
	AmountDue  int64             `json:"amount_due"`
	Currency   string            `json:"currency"`
	Receipt    string            `json:"receipt"`
	Status     string            `json:"status"` // created | attempted | paid
	Attempts   int               `json:"attempts"`
	Notes      map[string]string `json:"notes,omitempty"`
	CreatedAt  int64             `json:"created_at"`
}

// ============================================================
// Payment
// ============================================================

// PaymentResponse represents Razorpay Payment object.
type PaymentResponse struct {
	ID        string            `json:"id"`     // pay_xxx
	Entity    string            `json:"entity"` // "payment"
	Amount    int64             `json:"amount"`
	Currency  string            `json:"currency"`
	Status    string            `json:"status"`   // created | authorized | captured | refunded | failed
	OrderID   string            `json:"order_id"` // order_xxx
	Method    string            `json:"method"`   // card | upi | netbanking | wallet
	Captured  bool              `json:"captured"`
	Email     string            `json:"email,omitempty"`
	Contact   string            `json:"contact,omitempty"`
	Fee       int64             `json:"fee,omitempty"` // Razorpay's own fee (not ours)
	Tax       int64             `json:"tax,omitempty"`
	Notes     map[string]string `json:"notes,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// ============================================================
// Refund
// ============================================================

// RefundResponse represents Razorpay Refund object.
type RefundResponse struct {
	ID             string            `json:"id"`     // rfnd_xxx
	Entity         string            `json:"entity"` // "refund"
	Amount         int64             `json:"amount"`
	Currency       string            `json:"currency"`
	PaymentID      string            `json:"payment_id"`
	Notes          map[string]string `json:"notes,omitempty"`
	Receipt        string            `json:"receipt,omitempty"`
	Status         string            `json:"status"`                    // pending | processed | failed
	SpeedRequested string            `json:"speed_requested,omitempty"` // normal | optimum
	SpeedProcessed string            `json:"speed_processed,omitempty"`
	CreatedAt      int64             `json:"created_at"`
}

// ============================================================
// Customer
// ============================================================

// CustomerResponse represents Razorpay Customer object.
type CustomerResponse struct {
	ID        string            `json:"id"`     // cust_xxx
	Entity    string            `json:"entity"` // "customer"
	Name      string            `json:"name,omitempty"`
	Email     string            `json:"email,omitempty"`
	Contact   string            `json:"contact,omitempty"`
	Notes     map[string]string `json:"notes,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// ============================================================
// Webhook event types (subset — only what we handle)
// ============================================================

const (
	// Payment events
	EventPaymentAuthorized = "payment.authorized"
	EventPaymentCaptured   = "payment.captured"
	EventPaymentFailed     = "payment.failed"

	// Refund events
	EventRefundCreated   = "refund.created"
	EventRefundProcessed = "refund.processed"
	EventRefundFailed    = "refund.failed"

	// Order events
	EventOrderPaid = "order.paid"

	// Fund account (RazorpayX)
	EventFundAccountValidationCompleted = "fund_account.validation.completed"
	EventFundAccountValidationFailed    = "fund_account.validation.failed"

	// Payout (RazorpayX)
	EventPayoutProcessed = "payout.processed"
	EventPayoutFailed    = "payout.failed"
	EventPayoutReversed  = "payout.reversed"
)

// ============================================================
// Razorpay error codes (subset)
// ============================================================

const (
	ErrCodeBadRequest        = "BAD_REQUEST_ERROR"
	ErrCodeGatewayError      = "GATEWAY_ERROR"
	ErrCodeServerError       = "SERVER_ERROR"
	ErrCodePaymentFailed     = "PAYMENT_FAILED"
	ErrCodePaymentIDNotFound = "PAYMENT_ID_NOT_FOUND"
	ErrCodeRefundFailed      = "REFUND_FAILED"
)

// ============================================================
// Helper — isEventCapturedPayment
// ============================================================

func IsPaymentCapturedEvent(eventType string) bool {
	return eventType == EventPaymentCaptured
}

func IsRefundProcessedEvent(eventType string) bool {
	return eventType == EventRefundProcessed
}
