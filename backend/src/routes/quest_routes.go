package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

// RegisterQuestRoutes registers all quest-related routes with Huma.
func RegisterQuestRoutes(api huma.API, c *controllers.QuestController) {
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
						BudgetMinor:     7500000, // ₹75,000.00
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
		Description: "Validates quest is in draft status, sets quest status to open and records the publish timestamp. No payment or escrow movement happens here: escrow is funded per-milestone via POST /v1/quests/{id}/milestones/{milestone_id}/fund once a milestone is agreed.",
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

	// 4a. Saved quests — static path must be registered before /v1/quests/{id}.
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-saved-quests",
		Method:      "GET",
		Path:        "/v1/quests/saved",
		Tags:        tags,
		Summary:     "List bookmarked quests",
		Description: "Returns every quest the authenticated caller bookmarked, newest first.",
		Middlewares: generalAuthMw,
	}, c.ListSavedQuestsHandler)

	// 4b. Freelancer quest board summary — static path, registered before /v1/quests/{id}.
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-quest-board-summary",
		Method:      "GET",
		Path:        "/v1/quests/board/summary",
		Tags:        tags,
		Summary:     "Freelancer quest board summary",
		Description: "Returns the caller's application KPIs, latest applications and rank-eligible quests ranked by skill match.",
		Middlewares: generalAuthMw,
	}, c.BoardSummaryHandler)

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
		Description: "Validates quest is not already cancelled/completed, refunds every milestone still in a refundable state (work_in_progress, submitted_for_release) through the escrow refund flow, then marks the quest cancelled. Returns 409 for legacy quest-level escrow, which needs manual reconciliation.",
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
	}, c.RejectQuestHandler)

	// 6. Update Draft Quest (Client Only)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-quest",
		Method:      "PATCH",
		Path:        "/v1/quests/{id}",
		Tags:        tags,
		Summary:     "Update a draft quest",
		Description: "Partially updates a quest the caller owns. Only drafts are editable (409 otherwise). Changing the budget re-derives the required minimum rank. Providing skills replaces the whole skill set.",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateQuestBody{
						Title:       strPtr("Website Redesign for FinTech Startup v2"),
						BudgetMinor: int64Ptr(9000000),
					},
				},
			},
		},
	}, c.UpdateQuestHandler)

	// 7. Toggle Quest Bookmark
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "save-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/save",
		Tags:        tags,
		Summary:     "Bookmark / unbookmark a quest",
		Description: "Toggles the caller's bookmark for this quest and returns the resulting state.",
		Middlewares: generalAuthMw,
	}, c.SaveQuestHandler)
}

func strPtr(s string) *string { return &s }

func int64Ptr(v int64) *int64 { return &v }
