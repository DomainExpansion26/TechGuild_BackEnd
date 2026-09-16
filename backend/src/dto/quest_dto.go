package dto

import "time"

// CreateQuestMilestoneInput represents an individual milestone deliverable in a quest creation request.
type CreateQuestMilestoneInput struct {
	Title       string  `json:"title" huma:"required,minLength=3,maxLength=255" example:"Research & Wireframes"`
	Description string  `json:"description,omitempty" example:"User research and low-fidelity wireframes"`
	Amount      float64 `json:"amount" huma:"required,minimum=0.01" example:"15000"`
	DueDays     int     `json:"due_days" huma:"required,minimum=1" example:"5"`
	OrderIndex  int     `json:"order_index" huma:"required,minimum=1" example:"1"`
}

// CreateQuestSkillInput represents a technical skill requirement in a quest creation request.
type CreateQuestSkillInput struct {
	Name             string `json:"name" huma:"required" example:"Figma"`
	ProficiencyLevel string `json:"proficiency_level,omitempty" example:"expert"`
	IsCore           bool   `json:"is_core" example:"true"`
}

// CreateQuestLanguageInput represents a preferred language requirement in a quest creation request.
type CreateQuestLanguageInput struct {
	Name        string `json:"name" huma:"required" example:"English"`
	Proficiency string `json:"proficiency,omitempty" example:"Native/Bilingual"`
}

// CreateQuestBody contains the payload for creating a quest draft.
type CreateQuestBody struct {
	Title           string                      `json:"title" huma:"required,minLength=3,maxLength=100" example:"Website Redesign for FinTech Startup"`
	Description     string                      `json:"description" huma:"required,minLength=10" example:"Project focuses on modernizing the UI and streamlining the UX for our digital banking dashboard."`
	Category        string                      `json:"category" huma:"required" example:"UI/UX Design"`
	Industry        string                      `json:"industry,omitempty" example:"FinTech"`
	Complexity      string                      `json:"complexity,omitempty" example:"intermediate"`
	BudgetINR       float64                     `json:"budget_inr" huma:"required,minimum=1000" doc:"Total quest budget in INR (minimum ₹1,000)" example:"75000"`
	TimelineDays    int                         `json:"timeline_days" huma:"required,minimum=1" example:"20"`
	QuestType       string                      `json:"quest_type,omitempty" example:"standard"`
	ExperienceLevel string                      `json:"experience_level,omitempty" example:"senior"`
	SoftSkills      []string                    `json:"soft_skills,omitempty"`
	Languages       []CreateQuestLanguageInput  `json:"languages,omitempty"`
	Skills          []CreateQuestSkillInput     `json:"skills" huma:"required,minItems=1"`
	Milestones      []CreateQuestMilestoneInput `json:"milestones" huma:"required,minItems=1"`
}

// CreateQuestInput wraps the Huma request body.
type CreateQuestInput struct {
	Body CreateQuestBody
}

// CreateQuestResponse defines the return structure upon successfully drafting a quest.
type CreateQuestResponse struct {
	QuestID         string    `json:"quest_id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	Title           string    `json:"title" example:"Website Redesign for FinTech Startup"`
	Status          string    `json:"status" example:"draft"`
	BudgetINR       float64   `json:"budget_inr" example:"75000"`
	RequiredMinRank string    `json:"required_min_rank" example:"B"`
	MilestoneCount  int       `json:"milestone_count" example:"4"`
	CreatedAt       time.Time `json:"created_at" example:"2026-09-03T20:15:00Z"`
}

// CreateQuestOutput wraps the Huma response body with HTTP status 201.
type CreateQuestOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   CreateQuestResponse
}
