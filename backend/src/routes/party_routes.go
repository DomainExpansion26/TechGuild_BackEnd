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
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-party",
		Method:      "POST",
		Path:        "/v1/parties",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Create a party",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreatePartyRequest{
						Name:        "Bengaluru Go Guild",
						Slug:        "bengaluru-go-guild",
						Description: "A 12-engineer party focused on Go, gRPC and cloud-native delivery.",
						LogoURL:     "https://cdn.techguild.in/logos/bengaluru-go-guild.png",
						BannerURL:   "https://cdn.techguild.in/banners/bengaluru-go-guild.png",
						IsHiring:    true,
					},
				},
			},
		},
	}, controllers.CreatePartyHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Get a party",
		Middlewares: authMw},
		controllers.GetPartyHandler)

	//new
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-members",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/members",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get party members",
		Middlewares: authMw,
	}, controllers.GetPartyMembersHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-invitations",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/invitations",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get party invitations",
		Middlewares: authMw,
	}, controllers.GetPartyInvitationsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-portfolio",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/portfolio",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get party portfolio",
		Middlewares: authMw,
	}, controllers.GetPartyPortfolioHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-party-skills",
		Method:      "GET",
		Path:        "/v1/parties/{party_id}/skills",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get party skills",
		Middlewares: authMw,
	}, controllers.GetPartySkillsHandler)

	//continued
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-party",
		Method:      "PUT",
		Path:        "/v1/parties/{party_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Update a party",
		Description: "PUT replaces all mutable party fields. Send every field; omitted fields are written as empty values.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePartyRequest{
						Name:        "Bengaluru Go Guild",
						Description: "Now 18 engineers; added a DevOps pod.",
						LogoURL:     "https://cdn.techguild.in/logos/bengaluru-go-guild.png",
						BannerURL:   "https://cdn.techguild.in/banners/bengaluru-go-guild.png",
						IsHiring:    false,
					},
				},
			},
		},
	}, controllers.UpdatePartyHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-party", Method: "DELETE",
		Path:        "/v1/parties/{party_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Delete a party",
		Middlewares: authMw},
		controllers.DeletePartyHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-my-parties", Method: "GET",
		Path:        "/v1/parties/my",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Get my parties",
		Middlewares: authMw},
		controllers.GetMyPartiesHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-available-parties",
		Method:      "GET",
		Path:        "/v1/parties/available",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Get available parties",
		Middlewares: authMw},
		controllers.GetAvailablePartiesHandler)

	// Members
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "invite-member",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/invite",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Invite a member",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.InviteMemberRequest{
						UserID:  "9f8b1c2d-3e4f-4a5b-8c6d-7e8f9a0b1c2d",
						Message: "We'd love to have you join the Bengaluru Go Guild.",
					},
				},
			},
		},
	}, controllers.InviteMemberHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "accept-invitation",
		Method:      "POST",
		Path:        "/v1/parties/invitation/{invitation_id}/accept",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Accept invitation",
		Description: "No request body is required; the authenticated user accepts the invitation identified by the path parameter.",
		Middlewares: authMw},
		controllers.AcceptInvitationHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "reject-invitation",
		Method:      "POST",
		Path:        "/v1/parties/invitation/{invitation_id}/reject",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Reject invitation",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectInvitationRequest{
						Reason: "Committed to a client engagement through March; cannot take on party work.",
					},
				},
			},
		},
	}, controllers.RejectInvitationHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "remove-member",
		Method:      "DELETE",
		Path:        "/v1/parties/{party_id}/member/{member_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Remove member",
		Middlewares: authMw},
		controllers.RemoveMemberHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "leave-party",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/leave",
		Tags:        []string{"Freelancer / Task Management"},
		Summary:     "[Freelancer] Leave party",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LeavePartyRequest{
						Reason: "Moving to an in-house role; leaving the party on good terms.",
					},
				},
			},
		},
	}, controllers.LeavePartyHandler)

	// Portfolio
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-portfolio",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/portfolio",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Create portfolio",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreatePortfolioRequest{
						Title:       "UPI Reconciliation Dashboard",
						Description: "Real-time settlement dashboard for a Pune NBFC, built with Go, React and ClickHouse.",
						ImageURL:    "https://cdn.techguild.in/portfolio/upi-reconciliation.png",
						ProjectURL:  "https://cygnetsolutions.in/case-studies/upi-reconciliation",
						GithubURL:   "https://github.com/cygnet-software/upi-reconciliation",
					},
				},
			},
		},
	}, controllers.CreatePortfolioHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-portfolio",
		Method:      "PUT",
		Path:        "/v1/parties/portfolio/{portfolio_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Update portfolio",
		Description: "PUT replaces all mutable portfolio fields. Send every field; omitted fields are written as empty values.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdatePortfolioRequest{
						Title:       "UPI Reconciliation Dashboard v2",
						Description: "Added multi-bank settlement matching and anomaly alerts.",
						ImageURL:    "https://cdn.techguild.in/portfolio/upi-reconciliation-v2.png",
						ProjectURL:  "https://cygnetsolutions.in/case-studies/upi-reconciliation-v2",
						GithubURL:   "https://github.com/cygnet-software/upi-reconciliation-v2",
					},
				},
			},
		},
	}, controllers.UpdatePortfolioHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-portfolio",
		Method:      "DELETE",
		Path:        "/v1/parties/portfolio/{portfolio_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Delete portfolio",
		Middlewares: authMw},
		controllers.DeletePortfolioHandler)

	// Skills
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "add-skill",
		Method:      "POST",
		Path:        "/v1/parties/{party_id}/skills",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Add skill",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddSkillRequest{
						SkillName:       "Terraform",
						ExperienceLevel: "expert",
					},
				},
			},
		},
	}, controllers.AddSkillHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-skill",
		Method:      "PUT",
		Path:        "/v1/parties/skills/{skill_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Update skill",
		Description: "PUT replaces the skill entirely; send both `skill_name` and `experience_level`.",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AddSkillRequest{
						SkillName:       "Kubernetes",
						ExperienceLevel: "expert",
					},
				},
			},
		},
	}, controllers.UpdateSkillHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-skill",
		Method:      "DELETE",
		Path:        "/v1/parties/skills/{skill_id}",
		Tags:        []string{"Agency / Profile & Party"},
		Summary:     "[Agency] Delete skill",
		Middlewares: authMw},
		controllers.DeleteSkillHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func PartyRoutes(router *gin.Engine) {
}
