package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterPaymentMethodRoutes(api huma.API, c *controllers.PaymentMethodController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}

	// ============================================================
	// Client Payment Methods (money IN)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-client-payment-methods",
		Method:      "GET",
		Path:        "/v1/settings/billing/payment-methods",
		Tags:        []string{"Payment Methods"},
		Summary:     "List saved client payment methods",
		Middlewares: authMw,
	}, c.ListClientPaymentMethodsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "add-client-payment-method",
		Method:      "POST",
		Path:        "/v1/settings/billing/payment-methods",
		Tags:        []string{"Payment Methods"},
		Summary:     "Add a client payment method (provider token)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddClientPaymentMethodRequest{
						Provider:         "mock",
						ProviderMethodID: "pm_test_123",
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
		Tags:        []string{"Payment Methods"},
		Summary:     "Set a payment method as primary",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: map[string]any{},
				},
			},
		},
	}, c.SetPrimaryClientPaymentMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "delete-client-payment-method",
		Method:      "DELETE",
		Path:        "/v1/settings/billing/payment-methods/{id}",
		Tags:        []string{"Payment Methods"},
		Summary:     "Delete a saved payment method",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: map[string]any{},
				},
			},
		},
	}, c.DeleteClientPaymentMethodHandler)
}
