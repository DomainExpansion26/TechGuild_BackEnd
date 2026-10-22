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
		Tags:        []string{"Webhooks"},
		Summary:     "Razorpay webhook receiver (signature-verified)",
	}, c.RazorpayWebhookHandler)
}
