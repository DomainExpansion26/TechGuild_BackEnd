package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterWalletRoutes(api huma.API, c *controllers.WalletController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"BearerAuth": {}}}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-wallet",
		Method:      "GET",
		Path:        "/v1/me/wallet",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] Get authenticated user's wallet",
		Middlewares: authMw,
	}, c.GetWalletHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-wallet-transactions",
		Method:      "GET",
		Path:        "/v1/me/wallet/transactions",
		Tags:        []string{"Freelancer / Wallet & Payouts"},
		Summary:     "[Freelancer] List wallet transactions",
		Middlewares: authMw,
	}, c.ListTransactionsHandler)
}
