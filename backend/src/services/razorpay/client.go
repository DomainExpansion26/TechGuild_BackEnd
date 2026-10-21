package razorpay

import (
	"errors"

	razorpay "github.com/razorpay/razorpay-go"
)

var (
	ErrKeyIDMissing     = errors.New("razorpay key_id not configured")
	ErrKeySecretMissing = errors.New("razorpay key_secret not configured")
)

// Client wraps the Razorpay SDK with helpers and stores key_id for frontend.
type Client struct {
	sdk       *razorpay.Client
	keyID     string
	keySecret string
}

func NewClient(keyID, keySecret string) (*Client, error) {
	if keyID == "" {
		return nil, ErrKeyIDMissing
	}
	if keySecret == "" {
		return nil, ErrKeySecretMissing
	}

	sdk := razorpay.NewClient(keyID, keySecret)
	return &Client{
		sdk:       sdk,
		keyID:     keyID,
		keySecret: keySecret,
	}, nil
}

// SDK exposes the underlying Razorpay SDK client.
func (c *Client) SDK() *razorpay.Client {
	return c.sdk
}

// KeyID exposes the public key id (safe for frontend + Basic Auth).
func (c *Client) KeyID() string {
	return c.keyID
}

// KeySecret exposes the secret (used only server-side for Basic Auth on
// direct HTTP calls to RazorpayX endpoints).
func (c *Client) KeySecret() string {
	return c.keySecret
}
