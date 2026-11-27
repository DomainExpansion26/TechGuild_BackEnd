package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterPayoutMethodRoutes(api huma.API, c *controllers.PayoutMethodController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}

	// ============================================================
	// Payout Methods (money OUT)
	// ============================================================

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-payout-methods",
		Method:      "GET",
		Path:        "/v1/settings/billing/payout-methods",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] List payout methods",
		Middlewares: authMw,
	}, c.ListPayoutMethodsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "add-payout-method",
		Method:      "POST",
		Path:        "/v1/settings/billing/payout-methods",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Add payout method (bank/UPI/card)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddPayoutMethodRequest{
						Provider:      "razorpay",
						Type:          "bank",
						Purpose:       "payout",
						BankName:      ptr("HDFC Bank"),
						AccountHolder: ptr("Ananya Sharma"),
						AccountNumber: ptr("50100234567891"),
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
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Set primary payout method",
		Middlewares: authMw,
	}, c.SetPrimaryPayoutMethodHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "delete-payout-method",
		Method:      "DELETE",
		Path:        "/v1/settings/billing/payout-methods/{id}",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Delete payout method",
		Middlewares: authMw,
	}, c.DeletePayoutMethodHandler)
}
