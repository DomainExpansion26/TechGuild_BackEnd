package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterWalletWithdrawalRoutes(api huma.API, c *controllers.WalletWithdrawalController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	bearer := []map[string][]string{{"bearerAuth": {}}}
	tags := []string{"Wallet Withdrawal"}

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "request-wallet-withdrawal",
		Method:      "POST",
		Path:        "/v1/me/wallet/withdraw",
		Tags:        tags,
		Summary:     "Request withdrawal from wallet to bank/UPI",
		Middlewares: authMw,
	}, c.RequestWithdrawalHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "list-wallet-withdrawals",
		Method:      "GET",
		Path:        "/v1/me/wallet/withdrawals",
		Tags:        tags,
		Summary:     "List wallet withdrawal history",
		Middlewares: authMw,
	}, c.ListWithdrawalsHandler)

	huma.Register(api, huma.Operation{
		Security:    bearer,
		OperationID: "get-wallet-withdrawal",
		Method:      "GET",
		Path:        "/v1/me/wallet/withdrawals/{payout_id}",
		Tags:        tags,
		Summary:     "Get withdrawal details",
		Middlewares: authMw,
	}, c.GetWithdrawalHandler)
}
