package routes

import (
	"techguild-backend/src/controllers"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterWebhookRoutes(api huma.API, c *controllers.WebhookController) {
	// Webhooks are PUBLIC — no auth middleware. Verification is done
	// inside the handler via signature check.

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{},
		OperationID: "razorpay-webhook",
		Method:      "POST",
		Path:        "/v1/webhooks/razorpay",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Razorpay webhook receiver (signature-verified)",
		Description: "Receives Razorpay events such as payment.captured. The X-Razorpay-Signature header is required and is verified by the handler against the webhook secret. Razorpay amounts are in paise, not rupees: 250000 = ₹2,500.00. The request body is documentation-only; the handler consumes it as raw JSON.",
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: map[string]any{
						"event":      "payment.captured",
						"account_id": "acc_FAKE1234567890",
						"contains":   []string{"payment"},
						"payload": map[string]any{
							"payment": map[string]any{
								"entity": map[string]any{
									"id":         "pay_FAKE1234567890",
									"entity":     "payment",
									"amount":     250000,
									"currency":   "INR",
									"status":     "captured",
									"order_id":   "order_FAKE1234567890",
									"method":     "upi",
									"email":      "aarav.sharma@example.in",
									"contact":    "+919800000000",
									"created_at": 1764230400,
								},
							},
						},
						"created_at": 1764230400,
					},
				},
			},
		},
	}, c.RazorpayWebhookHandler)
}
