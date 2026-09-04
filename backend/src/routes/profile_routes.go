package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func ptr[T any](v T) *T { return &v }

func RegisterProfileRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Public — slug availability check (static route, stays under /v1/profile/)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{},
		OperationID: "check-slug",
		Method:      "GET",
		Path:        "/v1/profile/check-slug",
		Tags:        []string{"Profile"},
	}, controllers.CheckSlugHandler)

	// Public — profile lookup by slug, moved to its own /v1/u/* namespace
	// so user-generated slugs can never collide with static /v1/profile/*
	// routes (e.g. individual, agency, client, avatar, logo, etc.)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{},
		OperationID: "get-public-profile",
		Method:      "GET",
		Path:        "/v1/u/{slug}",
		Tags:        []string{"Profile"},
	}, controllers.GetPublicProfileHandler)

	// Protected
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "upload-resume",
		Method:      "POST",
		Path:        "/v1/profile/upload-resume",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.UploadResumeHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "upload-avatar",
		Method:      "POST",
		Path:        "/v1/profile/avatar",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.UploadAvatarHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "upload-logo",
		Method:      "POST",
		Path:        "/v1/profile/logo",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.UploadLogoHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-avatar",
		Method:      "DELETE",
		Path:        "/v1/profile/avatar",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.DeleteAvatarHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-logo",
		Method:      "DELETE",
		Path:        "/v1/profile/logo",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.DeleteLogoHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-resume",
		Method:      "DELETE",
		Path:        "/v1/profile/resume",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.DeleteResumeHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-my-profile",
		Method:      "GET",
		Path:        "/v1/profile",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.GetMyProfileHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-profile-account",
		Method:      "DELETE",
		Path:        "/v1/profile",
		Tags:        []string{"Profile"},
		Summary:     "Delete profile account",
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
	}, controllers.DeleteProfileAccountHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-user-points",
		Method:      "GET",
		Path:        "/v1/profile/points",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.GetUserPointsHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "export-profile",
		Method:      "POST",
		Path:        "/v1/profile/export",
		Tags:        []string{"Profile"},
		Middlewares: authMw}, controllers.ExportProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-profile-deprecated",
		Method:      "POST",
		Path:        "/v1/profile",
		Tags:        []string{"Profile"},
		Summary:     "Deprecated: use POST /v1/profile/{type} instead",
		Deprecated:  true,
		Middlewares: authMw,
	}, controllers.DeprecatedProfileCreateHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-individual-profile",
		Method:      "POST",
		Path:        "/v1/profile/individual",
		Tags:        []string{"Profile"},
		Summary:     "Create individual profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateIndividualProfileRequest{
						Country:           ptr("India"),
						City:              ptr("Mumbai"),
						TimeZone:          ptr("Asia/Kolkata"),
						Headline:          ptr("Senior Full-Stack Developer"),
						Bio:               ptr("Full-stack developer with 5 years of experience"),
						ExperienceLevel:   ptr("senior"),
						Availability:      ptr("full-time"),
						Skills:            ptr([]string{"Go", "React", "PostgreSQL"}),
						ToolsTechnologies: ptr([]string{"Docker", "Kubernetes", "Git"}),
						ServiceCategories: ptr([]string{"Web Development", "Backend Development"}),
						PortfolioURL:      ptr("https://portfolio.example.com/johndoe"),
						GithubURL:         ptr("https://github.com/johndoe"),
						LinkedinURL:       ptr("https://linkedin.com/in/johndoe"),
						ResumeURL:         ptr("https://storage.example.com/resumes/johndoe.pdf"),
						DateOfBirth:       ptr("1990-01-15"),
						Gender:            ptr("male"),
						AvatarURL:         ptr("https://storage.example.com/avatars/user.png"),
						PreferredLanguage: ptr("en"),
						Phone:             ptr("+919876543210"),
						TermsConfirmed:    ptr(true),
					},
				},
			},
		},
	}, controllers.CreateIndividualProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-agency-profile",
		Method:      "POST",
		Path:        "/v1/profile/agency",
		Tags:        []string{"Profile"},
		Summary:     "Create agency profile (wizard step-save)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateAgencyProfileRequest{
						AgencyName:      "Acme Digital Agency",
						LogoURL:         ptr("https://storage.example.com/logos/acme.png"),
						Description:     ptr("Digital agency specializing in web and mobile development"),
						WebsiteURL:      ptr("https://acme.example.com"),
						ServicesOffered: ptr([]string{"Web Development", "Mobile App Development"}),
						Industries:      ptr([]string{"Technology", "E-commerce"}),
						TeamSize:        ptr("10-50"),
						Phone:           ptr("+919876543210"),
						ContactName:     ptr("Jane Smith"),
						Country:         ptr("India"),
						City:            ptr("Mumbai"),
						TimeZone:        ptr("Asia/Kolkata"),
					},
				},
			},
		},
	}, controllers.CreateAgencyProfileHandler)

	// client profile creation
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-client-profile",
		Method:      "POST",
		Path:        "/v1/profile/client",
		Tags:        []string{"Profile"},
		Summary:     "Create client profile (wizard step-save)",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateClientProfileRequest{
						CompanyName:  "TechStart Inc",
						LogoURL:      ptr("https://storage.example.com/logos/techstart.png"),
						Industry:     ptr("technology"),
						WebsiteURL:   ptr("https://techstart.example.com"),
						ProjectTypes: ptr([]string{"Web Development", "Mobile App"}),
						BudgetRange:  ptr("$5,000 - $20,000"),
						TeamSize:     ptr("5-10"),
						Phone:        ptr("+919876543210"),
						Country:      ptr("United States"),
						City:         ptr("San Francisco"),
						TimeZone:     ptr("America/Los_Angeles"),
					},
				},
			},
		},
	}, controllers.CreateClientProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-individual-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/individual",
		Tags:        []string{"Profile"},
		Summary:     "Update individual profile",
		Middlewares: authMw,
	}, controllers.UpdateIndividualProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-agency-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/agency",
		Tags:        []string{"Profile"},
		Summary:     "Update agency profile",
		Middlewares: authMw,
	}, controllers.UpdateAgencyProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-client-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/client",
		Tags:        []string{"Profile"},
		Summary:     "Update client profile",
		Middlewares: authMw,
	}, controllers.UpdateClientProfileHandler)
}

// ---------- Settings routes (migrated to Huma) ----------

func RegisterSettingsRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-account-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/account",
		Tags:        []string{"Settings"},
		Summary:     "Update account settings",
		Middlewares: authMw,
	}, controllers.UpdateAccountSettingsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-notifications",
		Method:      "PATCH",
		Path:        "/v1/settings/notifications",
		Tags:        []string{"Settings"},
		Summary:     "Update notification settings",
		Middlewares: authMw,
	}, controllers.UpdateNotificationsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-privacy-settings",
		Method:      "PATCH",
		Path:        "/v1/settings/privacy",
		Tags:        []string{"Settings"},
		Summary:     "Update privacy settings",
		Middlewares: authMw,
	}, controllers.UpdatePrivacySettingsHandler)
}
