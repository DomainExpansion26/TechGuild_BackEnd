package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterSettingsRoutes(api huma.API, c *controllers.SettingsController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// fetch account settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-account-settings",
		Method:      "GET",
		Path:        "/v1/settings/account",
		Tags:        []string{"Settings"},
		Summary:     "Get account settings",
		Middlewares: authMw,
	}, c.GetAccountSettingsHandler)

	// edit account settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-account-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/account",
		Tags:        []string{"Settings"},
		Summary:     "Update account settings",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateAccountRequest{
						Password:        "test@123",
						NewPassword:     "newpass@123",
						ConfirmPassword: "newpass@123",
					},
				},
			},
		},
	}, c.UpdateAccountSettingsHandler)

	// fetch notification settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-notifications",
		Method:      "GET",
		Path:        "/v1/settings/notifications",
		Tags:        []string{"Settings"},
		Summary:     "Get notification settings",
		Middlewares: authMw,
	}, c.GetNotificationsHandler)

	// edit notification settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-notifications",
		Method:      "PATCH",
		Path:        "/v1/settings/notifications",
		Tags:        []string{"Settings"},
		Summary:     "Update notification settings",
		Middlewares: authMw,
	}, c.UpdateNotificationsHandler)

	// fetch privacy settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-privacy-settings",
		Method:      "GET",
		Path:        "/v1/settings/privacy",
		Tags:        []string{"Settings"},
		Summary:     "Get privacy settings",
		Middlewares: authMw,
	}, c.GetPrivacySettingsHandler)

	// edit privacy settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-privacy-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/privacy",
		Tags:        []string{"Settings"},
		Summary:     "Update privacy settings",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePrivacyRequest{
						ProfileVisibility: "public",
					},
				},
			},
		},
	}, c.UpdatePrivacySettingsHandler)

	// fetch billing settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-billing-settings",
		Method:      "GET",
		Path:        "/v1/settings/billing",
		Tags:        []string{"Settings"},
		Summary:     "Get billing settings",
		Middlewares: authMw,
	}, c.GetBillingSettingsHandler)

	// setup payout method info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-payout-method",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payout-method",
		Tags:        []string{"Settings"},
		Summary:     "Update billing settings",
		Middlewares: authMw,
	}, c.UpdatePayoutMethodHandler)

	// edit payout schedule info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-payout-schedule",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payout-schedule",
		Tags:        []string{"Settings"},
		Summary:     "Update billing settings",
		Middlewares: authMw,
	}, c.UpdatePayoutScheduleHandler)

	// deactivate account
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "deactivate-account",
		Method:      "POST",
		Path:        "/v1/settings/account/deactivate",
		Tags:        []string{"Settings"},
		Summary:     "Deactivate account",
		Middlewares: authMw,
	}, c.DeactivateAccountHandler)

	// delete account
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-settings-account",
		Method:      "DELETE",
		Path:        "/v1/settings/account",
		Tags:        []string{"Settings"},
		Summary:     "Delete account",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.DeleteAccountRequest{
						Password: "test@123",
					},
				},
			},
		},
	}, c.DeleteAccountHandler)

	// fetch session info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-sessions",
		Method:      "GET",
		Path:        "/v1/settings/sessions",
		Tags:        []string{"Settings"},
		Summary:     "Get session info",
		Middlewares: authMw,
	}, c.GetSessionsHandler)

	// revoke specific session
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "revoke-session",
		Method:      "DELETE",
		Path:        "/v1/settings/sessions/{id}",
		Tags:        []string{"Settings"},
		Summary:     "Revoke specific session",
		Middlewares: authMw,
	}, c.RevokeSessionHandler)

	// sign out of all other sessions
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "sign-out-other-sessions",
		Method:      "DELETE",
		Path:        "/v1/settings/sessions/others",
		Tags:        []string{"Settings"},
		Summary:     "Sign out of all other sessions",
		Middlewares: authMw,
	}, c.SignOutOtherSessionsHandler)
}
