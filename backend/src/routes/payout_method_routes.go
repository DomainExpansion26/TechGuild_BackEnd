package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterPayoutMethodRoutes(api huma.API, c *controllers.PayoutMethodController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}

	// ============================================================
	// Payout Methods (money OUT)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-payout-methods",
		Method:      "GET",
		Path:        "/v1/settings/billing/payout-methods",
		Tags:        []string{"Payout Methods"},
		Summary:     "List payout methods for the authenticated user",
		Middlewares: authMw,
	}, c.ListPayoutMethodsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "add-payout-method",
		Method:      "POST",
		Path:        "/v1/settings/billing/payout-methods",
		Tags:        []string{"Payout Methods"},
		Summary:     "Add a payout method (bank / UPI / card / wallet)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddPayoutMethodRequest{
						Provider:      "mock",
						Type:          "bank",
						Purpose:       "payout",
						BankName:      ptr("HDFC Bank"),
						AccountHolder: ptr("John Doe"),
						AccountNumber: ptr("1234567890123456"),
						IFSC:          ptr("HDFC0001234"),
						SetPrimary:    true,
					},
				},
			},
		},
	}, c.AddPayoutMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "set-primary-payout-method",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payout-methods/{id}/primary",
		Tags:        []string{"Payout Methods"},
		Summary:     "Set a payout method as primary",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: map[string]any{},
				},
			},
		},
	}, c.SetPrimaryPayoutMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "delete-payout-method",
		Method:      "DELETE",
		Path:        "/v1/settings/billing/payout-methods/{id}",
		Tags:        []string{"Payout Methods"},
		Summary:     "Delete a payout method",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: map[string]any{},
				},
			},
		},
	}, c.DeletePayoutMethodHandler)
}
