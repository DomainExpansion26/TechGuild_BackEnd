package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterProfileRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Public
	huma.Register(api, huma.Operation{OperationID: "check-slug", Method: "GET", Path: "/v1/profile/check-slug", Tags: []string{"Profile"}, Security: []map[string][]string{}}, controllers.CheckSlugHandler)
	huma.Register(api, huma.Operation{OperationID: "get-public-profile", Method: "GET", Path: "/v1/profile/{slug}", Tags: []string{"Profile"}, Security: []map[string][]string{}}, controllers.GetPublicProfileHandler)

	// Protected
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "upload-resume", Method: "POST", Path: "/v1/profile/upload-resume", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.UploadResumeHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "upload-avatar", Method: "POST", Path: "/v1/profile/avatar", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.UploadAvatarHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "upload-logo", Method: "POST", Path: "/v1/profile/logo", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.UploadLogoHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-avatar", Method: "DELETE", Path: "/v1/profile/avatar", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.DeleteAvatarHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-logo", Method: "DELETE", Path: "/v1/profile/logo", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.DeleteLogoHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-resume", Method: "DELETE", Path: "/v1/profile/resume", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.DeleteResumeHandler)

	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "create-update-profile", Method: "POST", Path: "/v1/profile", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.CreateOrUpdateProfileHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-my-profile", Method: "GET", Path: "/v1/profile", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.GetMyProfileHandler)
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

	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-user-points", Method: "GET", Path: "/v1/profile/points", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.GetUserPointsHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "export-profile", Method: "POST", Path: "/v1/profile/export", Tags: []string{"Profile"}, Middlewares: authMw}, controllers.ExportProfileHandler)

	// Individual/Agency/Client JSON-only variants
	examplePhone := "+1234567890"
	exampleDOB := "1990-01-15"

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-update-individual-profile",
		Method:      "POST",
		Path:        "/v1/profile/individual",
		Tags:        []string{"Profile"},
		Summary:     "Create or update individual profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateIndividualProfileRequest{
						Phone:             &examplePhone,
						DateOfBirth:       &exampleDOB,
						Gender:            "male",
						AvatarURL:         "https://storage.example.com/avatars/user.png",
						Bio:               "Full-stack developer with 5 years of experience",
						Country:           "India",
						State:             "Maharashtra",
						City:              "Mumbai",
						Headline:          "Senior Full-Stack Developer",
						PreferredLanguage: "en",
						TimeZone:          "Asia/Kolkata",
						CountryCode:       "+91",
						ExperienceLevel:   "senior",
						Availability:      "full-time",
						Skills:            []string{"Go", "React", "PostgreSQL"},
						ToolsTechnologies: []string{"Docker", "Kubernetes", "Git"},
						ServiceCategories: []string{"Web Development", "Backend Development"},
						PortfolioURL:      "https://portfolio.example.com/johndoe",
						GithubURL:         "https://github.com/johndoe",
						LinkedinURL:       "https://linkedin.com/in/johndoe",
						ResumeURL:         "https://storage.example.com/resumes/johndoe.pdf",
						TermsConfirmed:    true,
						ProfileVisibility: "public",
					},
				},
			},
		},
	}, controllers.CreateOrUpdateIndividualProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-update-agency-profile",
		Method:      "POST",
		Path:        "/v1/profile/agency",
		Tags:        []string{"Profile"},
		Summary:     "Create or update agency profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateAgencyProfileRequest{
						AgencyName:      "Acme Digital Agency",
						LogoURL:         "https://storage.example.com/logos/acme.png",
						Description:     "Digital agency specializing in web and mobile development",
						WebsiteURL:      "https://acme.example.com",
						ServicesOffered: []string{"Web Development", "Mobile App Development"},
						Industries:      []string{"Technology", "E-commerce"},
						TeamSize:        "10-50",
						ContactName:     "Jane Smith",
						Phone:           &examplePhone,
						RegistrationNo:  "U74110MH2020PTC123456",
						Country:         "India",
						State:           "Maharashtra",
						City:            "Mumbai",
						TimeZone:        "Asia/Kolkata",
						CountryCode:     "+91",
					},
				},
			},
		},
	}, controllers.CreateOrUpdateAgencyProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-update-client-profile",
		Method:      "POST",
		Path:        "/v1/profile/client",
		Tags:        []string{"Profile"},
		Summary:     "Create or update client profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateClientProfileRequest{
						CompanyName:  "TechStart Inc",
						LogoURL:      "https://storage.example.com/logos/techstart.png",
						Industry:     "technology",
						WebsiteURL:   "https://techstart.example.com",
						ProjectTypes: []string{"Web Development", "Mobile App"},
						BudgetRange:  "$5,000 - $20,000",
						TeamSize:     "5-10",
						ContactName:  "Alice Johnson",
						Phone:        &examplePhone,
						Country:      "United States",
						State:        "California",
						City:         "San Francisco",
						TimeZone:     "America/Los_Angeles",
						CountryCode:  "+1",
					},
				},
			},
		},
	}, controllers.CreateOrUpdateClientProfileHandler)
}

// ---------- Old Gin routes — Settings (not yet migrated) ----------

func ProfileRoutes(router *gin.Engine) {
	settingsGroup := router.Group("/v1/settings")
	settingsGroup.Use(middleware.AuthMiddleware())
	{
		settingsGroup.PATCH("/account", controllers.UpdateAccountSettings)
		settingsGroup.PATCH("/notifications", controllers.UpdateNotifications)
		settingsGroup.PATCH("/privacy", controllers.UpdatePrivacySettings)
	}
}
