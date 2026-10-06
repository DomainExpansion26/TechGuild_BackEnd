package services

// WebhookVerifier verifies and parses webhooks from one provider.
// Each provider adapter implements this. The domain layer depends only
// on this interface.
type WebhookVerifier interface {
	// Provider returns the provider identifier, e.g. "razorpay", "mock".
	Provider() string

	// Verify checks the webhook signature. Must be called BEFORE any
	// DB write. Returns nil if valid.
	Verify(payload []byte, signature string) error

	// ParseEvent extracts the provider-neutral event shape.
	// Must be called only after Verify succeeds.
	ParseEvent(payload []byte) (*WebhookEvent, error)
}

// WebhookEvent is the provider-neutral shape of a webhook event.
// Provider-specific details stay inside the adapter's payload.
type WebhookEvent struct {
	Provider        string
	ProviderEventID string
	EventType       string
	Raw             map[string]any
}
