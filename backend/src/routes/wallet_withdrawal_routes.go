package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterWalletWithdrawalRoutes(api huma.API, c *controllers.WalletWithdrawalController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-wallet-withdrawal",
		Method:      "POST",
		Path:        "/v1/me/wallet/withdraw",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Request withdrawal to bank/UPI",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RequestWithdrawalBody{
						Amount:         5000,
						PayoutMethodID: "550e8400-e29b-41d4-a716-446655440000",
						IdempotencyKey: "wd-2026-abc123",
					},
				},
			},
		},
	}, c.RequestWithdrawalHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-wallet-withdrawals",
		Method:      "GET",
		Path:        "/v1/me/wallet/withdrawals",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] List withdrawal history",
		Middlewares: authMw,
	}, c.ListWithdrawalsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-wallet-withdrawal",
		Method:      "GET",
		Path:        "/v1/me/wallet/withdrawals/{payout_id}",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Get withdrawal details",
		Middlewares: authMw,
	}, c.GetWithdrawalHandler)
}
