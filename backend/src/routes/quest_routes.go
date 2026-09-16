package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

// RegisterQuestRoutes registers all quest-related routes with Huma.
func RegisterQuestRoutes(api huma.API) {
	authMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
		middleware.ClientMiddlewareHuma(api),
	}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-quest",
		Method:      "POST",
		Path:        "/v1/quests",
		Tags:        []string{"Quests"},
		Summary:     "Create a quest draft",
		Description: "Allows authenticated and verified clients to create a quest draft with milestones and skill requirements. The minimum required rank is auto-computed based on the budget progression matrix (minimum budget ₹1,000).",
		Middlewares: authMw,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateQuestBody{
						Title:           "Website Redesign for FinTech Startup",
						Description:     "Project focuses on modernizing the user interface and streamlining the user experience for our flagship digital banking dashboard.",
						Category:        "UI/UX Design",
						Industry:        "FinTech",
						Complexity:      "intermediate",
						BudgetINR:       75000,
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
						Milestones: []dto.CreateQuestMilestoneInput{
							{Title: "Research & Wireframes", Description: "User research and low-fidelity wireframes", Amount: 15000, DueDays: 5, OrderIndex: 1},
							{Title: "High-Fidelity UI Design", Description: "Complete UI design system and high-fidelity mockups", Amount: 25000, DueDays: 5, OrderIndex: 2},
							{Title: "Prototype & Handoff", Description: "Interactive Figma prototype and developer handoff", Amount: 20000, DueDays: 5, OrderIndex: 3},
							{Title: "Final Review & Delivery", Description: "Final revisions and file handoff", Amount: 15000, DueDays: 5, OrderIndex: 4},
						},
					},
				},
			},
		},
	}, controllers.CreateQuestHandler)
}
