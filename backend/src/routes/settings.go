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
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-account-settings",
		Method:      "GET",
		Path:        "/v1/settings/account",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get account settings",
		Middlewares: authMw,
	}, c.GetAccountSettingsHandler)

	// edit account settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-account-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/account",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Update account settings",
		Description: "Changes the account password. Requires the current password plus the new password twice for confirmation.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateAccountRequest{
						Password:        "Pune@2026x",
						NewPassword:     "Hyderabad@2026z",
						ConfirmPassword: "Hyderabad@2026z",
					},
				},
			},
		},
	}, c.UpdateAccountSettingsHandler)

	// fetch notification settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-notifications",
		Method:      "GET",
		Path:        "/v1/settings/notifications",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get notification settings",
		Middlewares: authMw,
	}, c.GetNotificationsHandler)

	// edit notification settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-notifications",
		Method:      "PATCH",
		Path:        "/v1/settings/notifications",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Update notification settings",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateNotificationsRequest{
						Preferences: dto.NotificationPreference{
							Email: true,
							Push:  true,
						},
					},
				},
			},
		},
	}, c.UpdateNotificationsHandler)

	// fetch privacy settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-privacy-settings",
		Method:      "GET",
		Path:        "/v1/settings/privacy",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get privacy settings",
		Middlewares: authMw,
	}, c.GetPrivacySettingsHandler)

	// edit privacy settings info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-privacy-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/privacy",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Update privacy settings",
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
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-billing-settings",
		Method:      "GET",
		Path:        "/v1/settings/billing",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get billing settings",
		Middlewares: authMw,
	}, c.GetBillingSettingsHandler)

	// setup payout method info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-payout-method",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payout-method",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] DEPRECATED — use Payout Methods API",
		Description: "Deprecated. Returns 501 Not Implemented.",
		Deprecated:  true,
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePayoutMethodRequest{
						PayoutMethod: "paypal",
					},
				},
			},
		},
	}, c.UpdatePayoutMethodHandler)

	// edit payout schedule info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-payout-schedule",
		Method:      "PATCH",
		Path:        "/v1/settings/billing/payout-schedule",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] DEPRECATED — use Payout Methods API",
		Description: "Deprecated. Returns 501 Not Implemented.",
		Deprecated:  true,
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePayoutScheduleRequest{
						PayoutSchedule: "weekly",
					},
				},
			},
		},
	}, c.UpdatePayoutScheduleHandler)

	// deactivate account
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "deactivate-account",
		Method:      "POST",
		Path:        "/v1/settings/account/deactivate",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Deactivate account",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.DeactivateAccountRequest{
						Password:     "Pune@2026x",
						Confirmation: true,
					},
				},
			},
		},
	}, c.DeactivateAccountHandler)

	// delete account
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-settings-account",
		Method:      "DELETE",
		Path:        "/v1/settings/account",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Delete account",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.DeleteAccountRequest{
						Password: "Pune@2026x",
					},
				},
			},
		},
	}, c.DeleteAccountHandler)

	// fetch session info
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-sessions",
		Method:      "GET",
		Path:        "/v1/settings/sessions",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get session info",
		Middlewares: authMw,
	}, c.GetSessionsHandler)

	// revoke specific session
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "revoke-session",
		Method:      "DELETE",
		Path:        "/v1/settings/sessions/{id}",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Revoke specific session",
		Middlewares: authMw,
	}, c.RevokeSessionHandler)

	// sign out of all other sessions
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "sign-out-other-sessions",
		Method:      "DELETE",
		Path:        "/v1/settings/sessions/others",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Sign out of all other sessions",
		Middlewares: authMw,
	}, c.SignOutOtherSessionsHandler)
}
