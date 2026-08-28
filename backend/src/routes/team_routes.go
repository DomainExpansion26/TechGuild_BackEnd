package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterTeamRoutes(api huma.API) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	// Team CRUD
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-team",
		Method:      "POST",
		Path:        "/v1/teams",
		Tags:        []string{"Teams"},
		Summary:     "Create a team",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateTeamRequest{
						Name:        "TechGuild Devs",
						Slug:        "techguild-devs",
						Description: "A team of full-stack developers",
						LogoURL:     "https://storage.example.com/logos/team.png",
						BannerURL:   "https://storage.example.com/banners/team.png",
						IsHiring:    true,
					},
				},
			},
		},
	}, controllers.CreateTeamHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-team", Method: "GET", Path: "/v1/teams/{team_id}", Tags: []string{"Teams"}, Summary: "Get a team", Middlewares: authMw}, controllers.GetTeamHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "update-team",
		Method:      "PUT",
		Path:        "/v1/teams/{team_id}",
		Tags:        []string{"Teams"},
		Summary:     "Update a team",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateTeamRequest{
						Name:        "TechGuild Devs v2",
						Description: "Updated team description",
						LogoURL:     "https://storage.example.com/logos/team-v2.png",
						BannerURL:   "https://storage.example.com/banners/team-v2.png",
						IsHiring:    false,
					},
				},
			},
		},
	}, controllers.UpdateTeamHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-team", Method: "DELETE", Path: "/v1/teams/{team_id}", Tags: []string{"Teams"}, Summary: "Delete a team", Middlewares: authMw}, controllers.DeleteTeamHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "get-my-teams", Method: "GET", Path: "/v1/teams/my", Tags: []string{"Teams"}, Summary: "Get my teams", Middlewares: authMw}, controllers.GetMyTeamsHandler)

	// Members
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "invite-member",
		Method:      "POST",
		Path:        "/v1/teams/{team_id}/invite",
		Tags:        []string{"Teams"},
		Summary:     "Invite a member",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.InviteMemberRequest{
						UserID:  "550e8400-e29b-41d4-a716-446655440000",
						Message: "We'd love to have you on our team!",
					},
				},
			},
		},
	}, controllers.InviteMemberHandler)
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "accept-invitation", Method: "POST", Path: "/v1/teams/invitation/{invitation_id}/accept", Tags: []string{"Teams"}, Summary: "Accept invitation", Middlewares: authMw}, controllers.AcceptInvitationHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "reject-invitation",
		Method:      "POST",
		Path:        "/v1/teams/invitation/{invitation_id}/reject",
		Tags:        []string{"Teams"},
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
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "remove-member", Method: "DELETE", Path: "/v1/teams/{team_id}/member/{member_id}", Tags: []string{"Teams"}, Summary: "Remove member", Middlewares: authMw}, controllers.RemoveMemberHandler)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "leave-team",
		Method:      "POST",
		Path:        "/v1/teams/{team_id}/leave",
		Tags:        []string{"Teams"},
		Summary:     "Leave team",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LeaveTeamRequest{
						Reason: "Joining another team",
					},
				},
			},
		},
	}, controllers.LeaveTeamHandler)

	// Portfolio
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "create-portfolio",
		Method:      "POST",
		Path:        "/v1/teams/{team_id}/portfolio",
		Tags:        []string{"Teams"},
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
		Path:        "/v1/teams/portfolio/{portfolio_id}",
		Tags:        []string{"Teams"},
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
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-portfolio", Method: "DELETE", Path: "/v1/teams/portfolio/{portfolio_id}", Tags: []string{"Teams"}, Summary: "Delete portfolio", Middlewares: authMw}, controllers.DeletePortfolioHandler)

	// Skills
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "add-skill",
		Method:      "POST",
		Path:        "/v1/teams/{team_id}/skills",
		Tags:        []string{"Teams"},
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
		Path:        "/v1/teams/skills/{skill_id}",
		Tags:        []string{"Teams"},
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
	huma.Register(api, huma.Operation{Security: []map[string][]string{{"bearerAuth": {}}}, OperationID: "delete-skill", Method: "DELETE", Path: "/v1/teams/skills/{skill_id}", Tags: []string{"Teams"}, Summary: "Delete skill", Middlewares: authMw}, controllers.DeleteSkillHandler)
}

// ---------- Old Gin routes (not yet migrated) ----------

func TeamRoutes(router *gin.Engine) {
}
