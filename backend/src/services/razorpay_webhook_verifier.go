package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// RazorpayWebhookVerifier verifies Razorpay webhooks using HMAC-SHA256.
//
// Razorpay sends:
//
//	Header: X-Razorpay-Signature: <hex hmac>
//	Body:   raw JSON
//
// HMAC is computed over the raw body with the webhook secret.
type RazorpayWebhookVerifier struct {
	webhookSecret string
}

func NewRazorpayWebhookVerifier(webhookSecret string) *RazorpayWebhookVerifier {
	return &RazorpayWebhookVerifier{webhookSecret: webhookSecret}
}

func (v *RazorpayWebhookVerifier) Provider() string {
	return "razorpay"
}

func (v *RazorpayWebhookVerifier) Verify(payload []byte, signature string) error {
	if v.webhookSecret == "" {
		return errors.New("razorpay webhook secret not configured")
	}
	if signature == "" {
		return errors.New("missing razorpay signature")
	}

	mac := hmac.New(sha256.New, []byte(v.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("invalid razorpay webhook signature")
	}
	return nil
}

func (v *RazorpayWebhookVerifier) ParseEvent(payload []byte) (*WebhookEvent, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid razorpay webhook payload: %w", err)
	}

	// Razorpay webhook envelope:
	// {
	//   "event": "fund_account.verified",
	//   "payload": { ... },
	//   "created_at": 1234567890
	// }
	//
	// Razorpay does NOT provide a top-level event ID. Their docs recommend
	// using the account_id + created_at or an internal id from the payload.
	// We'll use a deterministic combination.
	eventType, _ := raw["event"].(string)
	if eventType == "" {
		return nil, errors.New("razorpay webhook missing event type")
	}

	eventID := extractRazorpayEventID(raw)
	if eventID == "" {
		return nil, errors.New("razorpay webhook missing event id")
	}

	return &WebhookEvent{
		Provider:        "razorpay",
		ProviderEventID: eventID,
		EventType:       eventType,
		Raw:             raw,
	}, nil
}

// extractRazorpayEventID looks for a stable ID inside the payload.
// Falls back to combining event type + payload id + created_at.
func extractRazorpayEventID(raw map[string]any) string {
	// Try payload.<entity>.id first
	if p, ok := raw["payload"].(map[string]any); ok {
		for _, v := range p {
			if entity, ok := v.(map[string]any); ok {
				if inner, ok := entity["entity"].(map[string]any); ok {
					if id, ok := inner["id"].(string); ok && id != "" {
						eventType, _ := raw["event"].(string)
						createdAt := formatCreatedAt(raw["created_at"])
						return fmt.Sprintf("%s:%s:%s", eventType, id, createdAt)
					}
				}
			}
		}
	}
	// Fallback: event type + created_at
	if eventType, ok := raw["event"].(string); ok {
		createdAt := formatCreatedAt(raw["created_at"])
		if createdAt != "" {
			return fmt.Sprintf("%s:%s", eventType, createdAt)
		}
	}
	return ""
}

// formatCreatedAt converts created_at to a stable integer string,
// avoiding scientific notation from float64.
func formatCreatedAt(v any) string {
	switch t := v.(type) {
	case float64:
		return fmt.Sprintf("%d", int64(t))
	case int64:
		return fmt.Sprintf("%d", t)
	case string:
		return t
	default:
		return ""
	}
}
