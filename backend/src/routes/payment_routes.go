package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterPaymentMethodRoutes(api huma.API, c *controllers.PaymentMethodController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}

	// ============================================================
	// Client Payment Methods (money IN)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-client-payment-methods",
		Method:      "GET",
		Path:        "/v1/settings/billing/payment-methods",
		Tags:        []string{"Client / Billing & Payments"},
		Summary:     "[Client] List saved client payment methods",
		Middlewares: authMw,
	}, c.ListClientPaymentMethodsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "add-client-payment-method",
		Method:      "POST",
		Path:        "/v1/settings/billing/payment-methods",
		Tags:        []string{"Client / Billing & Payments"},
		Summary:     "[Client] Add a client payment method",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddClientPaymentMethodRequest{
						Provider:         "razorpay",
						ProviderMethodID: "pm_NqX8kL2mZ4vB1c",
						Type:             "card",
						Last4:            ptr("4242"),
						Brand:            ptr("visa"),
						ExpMonth:         ptr(12),
						ExpYear:          ptr(2028),
						SetPrimary:       true,
					},
				},
			},
		},
	}, c.AddClientPaymentMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "set-primary-client-payment-method",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payment-methods/{id}/primary",
		Tags:        []string{"Client / Billing & Payments"},
		Summary:     "[Client] Set a payment method as primary",
		Middlewares: authMw,
	}, c.SetPrimaryClientPaymentMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "delete-client-payment-method",
		Method:      "DELETE",
		Path:        "/v1/settings/billing/payment-methods/{id}",
		Tags:        []string{"Client / Billing & Payments"},
		Summary:     "[Client] Delete a saved payment method",
		Middlewares: authMw,
	}, c.DeleteClientPaymentMethodHandler)
}
