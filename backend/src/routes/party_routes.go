package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterPartyRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Party CRUD
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-party",
		Method:      "POST",
		Path:        "/v1/parties",
		Tags:        []string{"Parties"},
		Summary:     "Create a party",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreatePartyRequest{
						Name:        "TechGuild Devs",
						Slug:        "techguild-devs",
						Description: "A party of full-stack developers",
						LogoURL:     "https://storage.example.com/logos/party.png",
						BannerURL:   "https://storage.example.com/banners/party.png",
						IsHiring:    true,
					},
				},
			},
		},
	}, controllers.CreatePartyHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-party",
		Method: "GET",
		Path: "/v1/parties/{party_id}",
		Tags: []string{"Parties"},
		Summary: "Get a party",
		Middlewares: authMw},
		controllers.GetPartyHandler)
	
	//new	
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-party-members",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/members",
		Tags:        []string{"Parties"},
		Summary:     "Get party members",
		Middlewares: authMw,
	}, controllers.GetPartyMembersHandler)	

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-party-invitations",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/invitations",
		Tags:        []string{"Parties"},
		Summary:     "Get party invitations",
		Middlewares: authMw,
	}, controllers.GetPartyInvitationsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-party-portfolio",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/portfolio",
		Tags:        []string{"Parties"},
		Summary:     "Get party portfolio",
		Middlewares: authMw,
	}, controllers.GetPartyPortfolioHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-party-skills",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/skills",
		Tags:        []string{"Parties"},
		Summary:     "Get party skills",
		Middlewares: authMw,
	}, controllers.GetPartySkillsHandler)

	//continued	
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-party",
		Method:      "PUT",
		Path:        "/v1/parties/{party_id}",
		Tags:        []string{"Parties"},
		Summary:     "Update a party",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePartyRequest{
						Name:        "TechGuild Devs v2",
						Description: "Updated party description",
						LogoURL:     "https://storage.example.com/logos/party-v2.png",
						BannerURL:   "https://storage.example.com/banners/party-v2.png",
						IsHiring:    false,
					},
				},
			},
		},
	}, controllers.UpdatePartyHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-party", Method: "DELETE", 
		Path: "/v1/parties/{party_id}", 
		Tags: []string{"Parties"}, 
		Summary: "Delete a party", 
		Middlewares: authMw}, 
		controllers.DeletePartyHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}}, 
		OperationID: "get-my-parties", Method: "GET", 
		Path: "/v1/parties/my",
		Tags: []string{"Parties"}, 
		Summary: "Get my parties", 
		Middlewares: authMw}, 
		controllers.GetMyPartiesHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-available-parties",
		Method:      "GET",
		Path:        "/v1/parties/available",
		Tags:        []string{"Parties"},
		Summary:     "Get parties available to freelancer",
		Middlewares: authMw,},
		controllers.GetAvailablePartiesHandler)


	// Members
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "invite-member",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/invite",
		Tags:        []string{"Parties"},
		Summary:     "Invite a member",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.InviteMemberRequest{
						UserID:  "550e8400-e29b-41d4-a716-446655440000",
						Message: "We'd love to have you on our party!",
					},
				},
			},
		},
	}, controllers.InviteMemberHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}}, 
		OperationID: "accept-invitation", 
		Method: "POST", 
		Path: "/v1/parties/invitation/{invitation_id}/accept", 
		Tags: []string{"Parties"}, 
		Summary: "Accept invitation", 
		Middlewares: authMw}, 
		controllers.AcceptInvitationHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-invitation",
		Method:      "POST",
		Path:        "/v1/parties/invitation/{invitation_id}/reject",
		Tags:        []string{"Parties"},
		Summary:     "Reject invitation",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectInvitationRequest{
						Reason: "Currently busy with other projects",
					},
				},
			},
		},
	}, controllers.RejectInvitationHandler)
	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}}, 
		OperationID: "remove-member", 
		Method: "DELETE", 
		Path: "/v1/parties/{party_id}/member/{member_id}", 
		Tags: []string{"Parties"}, 
		Summary: "Remove member", 
		Middlewares: authMw}, 
		controllers.RemoveMemberHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "leave-party",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/leave",
		Tags:        []string{"Parties"},
		Summary:     "Leave party",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LeavePartyRequest{
						Reason: "Joining another party",
					},
				},
			},
		},
	}, controllers.LeavePartyHandler)

	// Portfolio
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-portfolio",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/portfolio",
		Tags:        []string{"Parties"},
		Summary:     "Create portfolio",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreatePortfolioRequest{
						Title:       "E-commerce Platform",
						Description: "Built a full-stack e-commerce platform with React and Go",
						ImageURL:    "https://storage.example.com/portfolio/project.png",
						ProjectURL:  "https://example.com/project",
						GithubURL:   "https://github.com/user/project",
					},
				},
			},
		},
	}, controllers.CreatePortfolioHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-portfolio",
		Method:      "PUT",
		Path:        "/v1/parties/portfolio/{portfolio_id}",
		Tags:        []string{"Parties"},
		Summary:     "Update portfolio",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePortfolioRequest{
						Title:       "E-commerce Platform v2",
						Description: "Updated project with payment integration",
						ImageURL:    "https://storage.example.com/portfolio/project-v2.png",
						ProjectURL:  "https://example.com/project-v2",
						GithubURL:   "https://github.com/user/project-v2",
					},
				},
			},
		},
	}, controllers.UpdatePortfolioHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-portfolio",
		Method: "DELETE", 
		Path: "/v1/parties/portfolio/{portfolio_id}", 
		Tags: []string{"Parties"}, 
		Summary: "Delete portfolio", 
		Middlewares: authMw}, 
		controllers.DeletePortfolioHandler)

	// Skills
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "add-skill",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/skills",
		Tags:        []string{"Parties"},
		Summary:     "Add skill",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddSkillRequest{
						SkillName:       "Go",
						ExperienceLevel: "intermediate",
					},
				},
			},
		},
	}, controllers.AddSkillHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-skill",
		Method:      "PUT",
		Path:        "/v1/parties/skills/{skill_id}",
		Tags:        []string{"Parties"},
		Summary:     "Update skill",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddSkillRequest{
						SkillName:       "Go",
						ExperienceLevel: "expert",
					},
				},
			},
		},
	}, controllers.UpdateSkillHandler)

	huma.Register(api, huma.Operation{
		Security: []map[string][]string{{"bearerAuth": {}}},
		OperationID: "delete-skill", 
		Method: "DELETE", 
		Path: "/v1/parties/skills/{skill_id}", 
		Tags: []string{"Parties"}, 
		Summary: "Delete skill", 
		Middlewares: authMw}, 
		controllers.DeleteSkillHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func PartyRoutes(router *gin.Engine) {
}
