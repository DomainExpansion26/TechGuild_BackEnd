package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

// RegisterQuestRoutes registers all quest-related routes with Huma.
func RegisterQuestRoutes(api huma.API) {
	clientAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
		middleware.ClientMiddlewareHuma(api),
	}

	generalAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}

	tags := []string{"Quests"}

	// 1. Task P2-3.2: Create Quest Draft (Client Only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-quest",
		Method:      "POST",
		Path:        "/v1/quests",
		Tags:        tags,
		Summary:     "Create a quest draft",
		Description: "Allows authenticated and verified clients to create a quest draft with milestones and skill requirements. The minimum required rank is auto-computed based on the budget progression matrix (minimum budget ₹1,000).",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateQuestBody{
						Title:           "Website Redesign for FinTech Startup",
						Description:     "Project focuses on modernizing the user interface and streamlining the user experience for our flagship digital banking dashboard.",
						Category:        "UI/UX Design",
						Industry:        "FinTech",
						Complexity:      "intermediate",
						BudgetMinor:     75000,
						TimelineDays:    20,
						QuestType:       "standard",
						ExperienceLevel: "senior",
						SoftSkills:      []string{"Leadership", "Communication"},
						Languages: []dto.CreateQuestLanguageInput{
							{Name: "English", Proficiency: "Native/Bilingual"},
						},
						Skills: []dto.CreateQuestSkillInput{
							{Name: "Figma", ProficiencyLevel: "expert", IsCore: true},
							{Name: "React", ProficiencyLevel: "intermediate", IsCore: true},
							{Name: "Tailwind CSS", ProficiencyLevel: "intermediate", IsCore: false},
						},
					},
				},
			},
		},
	}, controllers.CreateQuestHandler)

	// 2.Publish Quest (Client Only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "publish-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/publish",
		Tags:        tags,
		Summary:     "Publish quest (open for acceptance)",
		Description: "Validates quest is in draft status, verifies client funding method, sets quest status to open, records visibility timestamps, and deposits escrow funds.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.PublishQuestBody{
						FundingMethod: "escrow_deposit",
					},
				},
			},
		},
	}, controllers.PublishQuestHandler)

	// 3.List Quests (Tier-Locked Asymmetric Visibility)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-quests",
		Method:      "GET",
		Path:        "/v1/quests",
		Tags:        tags,
		Summary:     "List quests (tier-locked per requester)",
		Description: "Returns public marketplace quests with asymmetric tier visibility (requester rank >= quest required_min_rank). Supports cursor-based pagination and filtering.",
		Middlewares: generalAuthMw,
	}, controllers.ListQuestsHandler)

	// 4.Get Quest Details
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-quest-details",
		Method:      "GET",
		Path:        "/v1/quests/{id}",
		Tags:        tags,
		Summary:     "Get quest details",
		Description: "Returns full quest details including milestones and required skills, if visible to the requester according to tier hierarchy and ownership.",
		Middlewares: generalAuthMw,
	}, controllers.GetQuestDetailsHandler)

	// 5.Reject / Cancel Quest (Client Only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "reject-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/reject",
		Tags:        tags,
		Summary:     "Reject / cancel quest (client only)",
		Description: "Validates quest is open or in_progress, marks quest cancelled, and triggers escrow refund if funds were held.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectQuestBody{
						Reason: "Project requirements changed",
					},
				},
			},
		},
	}, controllers.RejectQuestHandler)


	// 6.Quest Acceptance API (Party Lead)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "accept-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/accept",
		Tags:        tags,
		Summary:     "Accept quest on behalf of party",
		Description: "Party Lead accepts a quest on behalf of an active, verified party. Runs all 8 validation checks (Party Lead authorization, party rank >= quest required_min_rank, capacity < 5, skill match >= 60%, no client conflict, premium subscription check, escrow funded check) and atomically updates quest to in_progress, sets accepted_party_id, auto-creates quest chat channel, and auto-creates task templates.",
		Middlewares: generalAuthMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.AcceptQuestBody{
						PartyID: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
					},
				},
			},
		},
	}, controllers.AcceptQuestHandler)
}
