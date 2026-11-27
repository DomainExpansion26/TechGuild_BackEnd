package routes

import (
	"os"

	"github.com/danielgtaylor/huma/v2"
)

// APIConfig returns the Huma configuration shared by the server and by the
// OpenAPI/Postman contract tests. It is pure apart from reading
// SERVER_PUBLIC_URL, so the spec can be rendered without a database.
func APIConfig() huma.Config {
	apiConfig := huma.DefaultConfig("TechGuild Backend API", "1.0.0")
	apiConfig.Info.Description = "Backend API for TechGuild Platform."

	publicURL := os.Getenv("SERVER_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8080"
	}
	apiConfig.Servers = []*huma.Server{{URL: publicURL}}

	apiConfig.Components = &huma.Components{
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"BearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
	}
	apiConfig.Security = []map[string][]string{{"BearerAuth": {}}}

	// x-tagGroups drives the collapsible parent folders in Zudoku and the
	// two-level folder tree Postman builds when importing /openapi.json.
	apiConfig.Extensions = map[string]any{
		"x-tagGroups": []map[string]any{
			{
				"name": "Client",
				"tags": []string{
					"Client / Quest Board",
					"Client / Milestones & Escrow",
					"Client / Task Management",
					"Client / Profile & Verification",
					"Client / Billing & Payments",
				},
			},
			{
				"name": "Freelancer",
				"tags": []string{
					"Freelancer / Profile & Verification",
					"Freelancer / Quest Board",
					"Freelancer / Milestones & Deliverables",
					"Freelancer / Task Management",
					"Freelancer / Wallet & Payouts",
				},
			},
			{
				"name": "Agency",
				"tags": []string{"Agency / Profile & Party"},
			},
			{
				"name": "Common",
				"tags": []string{
					"Common / Authentication",
					"Common / Quest Discovery",
					"Common / Milestones Read",
					"Common / Profile & Verification",
					"Common / Guildhall",
					"Common / Settings & Webhooks",
				},
			},
			{
				"name": "Admin",
				"tags": []string{"Admin / Verification"},
			},
		},
	}

	return apiConfig
}
