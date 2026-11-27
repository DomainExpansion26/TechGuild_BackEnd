package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

func ptr[T any](v T) *T { return &v }

func RegisterProfileRoutes(api huma.API, c *controllers.ProfileController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Public — slug availability check (static route, stays under /v1/profile/)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{},
		OperationID: "check-slug",
		Method:      "GET",
		Path:        "/v1/profile/check-slug",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Check slug availability",
	}, c.CheckSlugHandler)

	// Public — profile lookup by slug, moved to its own /v1/u/* namespace
	// so user-generated slugs can never collide with static /v1/profile/*
	// routes (e.g. individual, agency, client, avatar, logo, etc.)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{},
		OperationID: "get-public-profile",
		Method:      "GET",
		Path:        "/v1/u/{slug}",
		Tags:        []string{"Common / Settings & Webhooks"},
		Summary:     "[Common] Get public profile by slug",
	}, c.GetPublicProfileHandler)

	// Protected
	huma.Register(api, huma.Operation{
		Security:     []map[string][]string{{"BearerAuth": {}}},
		OperationID:  "upload-resume",
		Method:       "POST",
		Path:         "/v1/profile/upload-resume",
		Tags:         []string{"Freelancer / Profile & Verification"},
		Summary:      "[Freelancer] Upload resume (PDF)",
		Description:  "Uploads a resume as `multipart/form-data` with a single required `resume` file field. Accepted content type: PDF (application/pdf). Maximum file size: 2 MiB.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw}, c.UploadResumeHandler)

	huma.Register(api, huma.Operation{
		Security:     []map[string][]string{{"BearerAuth": {}}},
		OperationID:  "upload-avatar",
		Method:       "POST",
		Path:         "/v1/profile/avatar",
		Tags:         []string{"Common / Profile & Verification"},
		Summary:      "[Common] Upload avatar",
		Description:  "Uploads an avatar as `multipart/form-data` with a single required `avatar` file field. Accepted content types: image/jpeg, image/png, image/webp. Maximum file size: 2 MiB.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw}, c.UploadAvatarHandler)

	huma.Register(api, huma.Operation{
		Security:     []map[string][]string{{"BearerAuth": {}}},
		OperationID:  "upload-logo",
		Method:       "POST",
		Path:         "/v1/profile/logo",
		Tags:         []string{"Common / Profile & Verification"},
		Summary:      "[Common] Upload logo",
		Description:  "Uploads a logo as `multipart/form-data` with a single required `logo` file field. Accepted content types: image/jpeg, image/png, image/webp. Maximum file size: 2 MiB.",
		MaxBodyBytes: utils.MaxUploadBytes,
		Middlewares:  authMw}, c.UploadLogoHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-avatar",
		Method:      "DELETE",
		Path:        "/v1/profile/avatar",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Delete avatar",
		Middlewares: authMw}, c.DeleteAvatarHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-logo",
		Method:      "DELETE",
		Path:        "/v1/profile/logo",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Delete logo",
		Middlewares: authMw}, c.DeleteLogoHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-resume",
		Method:      "DELETE",
		Path:        "/v1/profile/resume",
		Tags:        []string{"Freelancer / Profile & Verification"},
		Summary:     "[Freelancer] Delete resume",
		Middlewares: authMw}, c.DeleteResumeHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-my-profile",
		Method:      "GET",
		Path:        "/v1/profile",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Get my profile",
		Middlewares: authMw}, c.GetMyProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-user-points",
		Method:      "GET",
		Path:        "/v1/profile/points",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Get user points",
		Middlewares: authMw}, c.GetUserPointsHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "export-profile",
		Method:      "POST",
		Path:        "/v1/profile/export",
		Tags:        []string{"Common / Profile & Verification"},
		Summary:     "[Common] Export profile data",
		Description: "No request body is required; only the path and Bearer authentication are used.",
		Middlewares: authMw}, c.ExportProfileHandler)

	// huma.Register(api, huma.Operation{
	// 	Security:    []map[string][]string{{"BearerAuth": {}}},
	// 	OperationID: "create-profile-deprecated",
	// 	Method:      "POST",
	// 	Path:        "/v1/profile",
	// 	Tags:        []string{"Profile"},
	// 	Summary:     "Deprecated: use POST /v1/profile/{type} instead",
	// 	Deprecated:  true,
	// 	Middlewares: authMw,
	// }, c.DeprecatedProfileCreateHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-individual-profile",
		Method:      "POST",
		Path:        "/v1/profile/individual",
		Tags:        []string{"Freelancer / Profile & Verification"},
		Summary:     "[Freelancer] Create individual profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateIndividualProfileRequest{
						Country:           ptr("India"),
						City:              ptr("Bengaluru"),
						TimeZone:          ptr("Asia/Kolkata"),
						Headline:          ptr("Lead Backend Engineer"),
						Bio:               ptr("Backend engineer with 8 years building Go services for fintech and logistics."),
						ExperienceLevel:   ptr("senior"),
						Availability:      ptr("full-time"),
						Skills:            ptr([]string{"Go", "React", "PostgreSQL"}),
						ToolsTechnologies: ptr([]string{"Docker", "Kubernetes", "Git"}),
						ServiceCategories: ptr([]string{"Web Development", "Backend Development"}),
						PortfolioURL:      ptr("https://cdn.techguild.in/portfolio/ananya-iyer"),
						GithubURL:         ptr("https://github.com/ananya-iyer"),
						LinkedinURL:       ptr("https://linkedin.com/in/ananya-iyer"),
						ResumeURL:         ptr("https://cdn.techguild.in/resumes/ananya-iyer.pdf"),
						DateOfBirth:       ptr("1990-01-15"),
						Gender:            ptr("female"),
						AvatarURL:         ptr("https://cdn.techguild.in/avatars/ananya-iyer.png"),
						PreferredLanguage: ptr("en"),
						Phone:             ptr("+919812345678"),
						TermsConfirmed:    ptr(true),
					},
				},
			},
		},
	}, c.CreateIndividualProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-agency-profile",
		Method:      "POST",
		Path:        "/v1/profile/agency",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Create agency profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateAgencyProfileRequest{
						AgencyName:      "Cygnet Software Solutions Pvt Ltd",
						LogoURL:         ptr("https://cdn.techguild.in/logos/cygnet.png"),
						Description:     ptr("Pune-based product engineering agency for fintech and logistics clients."),
						WebsiteURL:      ptr("https://cygnetsolutions.in"),
						ServicesOffered: ptr([]string{"Web Development", "Mobile App Development", "Cloud & DevOps"}),
						Industries:      ptr([]string{"Fintech", "Logistics", "E-commerce"}),
						TeamSize:        ptr("50-200"),
						Phone:           ptr("+919820012345"),
						ContactName:     ptr("Rohan Mehta"),
						Country:         ptr("India"),
						City:            ptr("Pune"),
						TimeZone:        ptr("Asia/Kolkata"),
					},
				},
			},
		},
	}, c.CreateAgencyProfileHandler)

	// client profile creation
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-client-profile",
		Method:      "POST",
		Path:        "/v1/profile/client",
		Tags:        []string{"Client / Profile & Verification"},
		Summary:     "[Client] Create client profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateClientProfileRequest{
						CompanyName:  "Sundaram Logistics Pvt Ltd",
						LogoURL:      ptr("https://cdn.techguild.in/logos/sundaram.png"),
						Industry:     ptr("logistics"),
						WebsiteURL:   ptr("https://sundaramlogistics.in"),
						ProjectTypes: ptr([]string{"Web Development", "Mobile App"}),
						BudgetRange:  ptr("INR 5,00,000 - 20,00,000"),
						TeamSize:     ptr("50-200"),
						Phone:        ptr("+919833012345"),
						Country:      ptr("India"),
						City:         ptr("Mumbai"),
						TimeZone:     ptr("Asia/Kolkata"),
					},
				},
			},
		},
	}, c.CreateClientProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-individual-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/individual",
		Tags:        []string{"Freelancer / Profile & Verification"},
		Summary:     "[Freelancer] Update individual profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateIndividualProfileRequest{
						City:         ptr("Pune"),
						Headline:     ptr("Lead Backend Engineer"),
						Availability: ptr("full-time"),
						Skills:       ptr([]string{"Go", "PostgreSQL", "Kubernetes"}),
					},
				},
			},
		},
	}, c.UpdateIndividualProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-agency-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/agency",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Update agency profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateAgencyProfileRequest{
						TeamSize:        ptr("200+"),
						ServicesOffered: ptr([]string{"Web Development", "Cloud & DevOps"}),
						Description:     ptr("Expanded to 240 engineers across Pune and Hyderabad."),
					},
				},
			},
		},
	}, c.UpdateAgencyProfileHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-client-profile",
		Method:      "PATCH",
		Path:        "/v1/profile/client",
		Tags:        []string{"Client / Profile & Verification"},
		Summary:     "[Client] Update client profile",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Required: false, // false only when EVERY body field is optional
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateClientProfileRequest{
						BudgetRange:  ptr("INR 10,00,000 - 35,00,000"),
						ProjectTypes: ptr([]string{"Web Development", "Data & Analytics"}),
					},
				},
			},
		},
	}, c.UpdateClientProfileHandler)
}
