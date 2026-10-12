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

// ---------- Publish Quest ----------

type PublishQuestBody struct {
	FundingMethod string `json:"funding_method,omitempty" doc:"Client funding method (e.g., bank_transfer, escrow_deposit)" example:"escrow_deposit"`
	FundEscrowNow *bool  `json:"fund_escrow_now,omitempty" doc:"Whether to deposit funds into escrow upon publish (defaults to true)"`
}

type PublishQuestInput struct {
	ID   string           `path:"id" doc:"Quest UUID"`
	Body PublishQuestBody `json:"body,omitempty"`
}

type PublishQuestResponse struct {
	QuestID        string    `json:"quest_id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	Title          string    `json:"title" example:"Website Redesign for FinTech Startup"`
	Status         string    `json:"status" example:"open"`
	PublishedAt    time.Time `json:"published_at" example:"2026-09-03T20:20:00Z"`
	IsEscrowFunded bool      `json:"is_escrow_funded" example:"true"`
	Message        string    `json:"message" example:"Quest published successfully and open for acceptance"`
}

type PublishQuestOutput struct {
	Status int `json:"-" huma:"status:200"`
	Body   PublishQuestResponse
}

// ---------- List Quests (Tier-Locked) ----------

type ListQuestsInput struct {
	Cursor     string   `query:"cursor" doc:"Cursor for pagination (UUID or ISO-8601 timestamp)"`
	Limit      int      `query:"limit" doc:"Number of records to return (default 10, max 50)"`
	Category   string   `query:"category" doc:"Filter by quest category"`
	Complexity string   `query:"complexity" doc:"Filter by quest complexity (beginner, intermediate, advanced, enterprise)"`
	MinBudget  *float64 `query:"min_budget" doc:"Filter by minimum budget INR"`
	MaxBudget  *float64 `query:"max_budget" doc:"Filter by maximum budget INR"`
	Status     string   `query:"status" doc:"Filter by status (open, in_progress, completed, etc.)"`
}

type QuestListItem struct {
	ID              string     `json:"id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	Title           string     `json:"title" example:"Website Redesign for FinTech Startup"`
	Description     string     `json:"description" example:"Redesign of dashboard"`
	Category        string     `json:"category" example:"UI/UX Design"`
	Complexity      string     `json:"complexity" example:"intermediate"`
	BudgetINR       float64    `json:"budget_inr" example:"75000"`
	TimelineDays    int        `json:"timeline_days" example:"20"`
	RequiredMinRank string     `json:"required_min_rank" example:"B"`
	Status          string     `json:"status" example:"open"`
	ClientID        string     `json:"client_id" example:"11111111-2222-3333-4444-555555555555"`
	ClientName      string     `json:"client_name" example:"Acme Corp"`
	Skills          []string   `json:"skills" example:"[\"Figma\", \"React\"]"`
	MilestoneCount  int        `json:"milestone_count" example:"4"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ListQuestsResponse struct {
	Quests     []QuestListItem `json:"quests"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
	Total      int             `json:"total"`
}

type ListQuestsOutput struct {
	Body ListQuestsResponse
}

// ---------- Get Quest Details ----------

type GetQuestDetailsInput struct {
	ID string `path:"id" doc:"Quest UUID"`
}

type QuestDetailMilestone struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Amount      float64 `json:"amount"`
	DueDays     int     `json:"due_days"`
	OrderIndex  int     `json:"order_index"`
	Status      string  `json:"status"`
}

type QuestDetailSkill struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ProficiencyLevel string `json:"proficiency_level"`
	IsCore           bool   `json:"is_core"`
}

type QuestDetailsResponse struct {
	ID              string                 `json:"id"`
	ClientID        string                 `json:"client_id"`
	ClientName      string                 `json:"client_name"`
	ClientRank      string                 `json:"client_rank"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Industry        string                 `json:"industry,omitempty"`
	Complexity      string                 `json:"complexity"`
	BudgetINR       float64                `json:"budget_inr"`
	TimelineDays    int                    `json:"timeline_days"`
	QuestType       string                 `json:"quest_type"`
	RequiredMinRank string                 `json:"required_min_rank"`
	ExperienceLevel string                 `json:"experience_level,omitempty"`
	SoftSkills      []string               `json:"soft_skills,omitempty"`
	Status          string                 `json:"status"`
	PublishedAt     *time.Time             `json:"published_at,omitempty"`
	IsEscrowFunded  bool                   `json:"is_escrow_funded"`
	AcceptedPartyID *string                `json:"accepted_party_id,omitempty"`
	Milestones      []QuestDetailMilestone `json:"milestones"`
	Skills          []QuestDetailSkill     `json:"skills"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type GetQuestDetailsOutput struct {
	Body QuestDetailsResponse
}

// ---------- Reject / Cancel Quest ----------

type RejectQuestBody struct {
	Reason string `json:"reason,omitempty" example:"Budget re-allocated"`
}

type RejectQuestInput struct {
	ID   string          `path:"id" doc:"Quest UUID"`
	Body RejectQuestBody `json:"body,omitempty"`
}

type RejectQuestResponse struct {
	QuestID            string  `json:"quest_id"`
	Status             string  `json:"status"`
	EscrowRefunded     bool    `json:"escrow_refunded"`
	EscrowRefundAmount float64 `json:"escrow_refund_amount"`
	Message            string  `json:"message"`
}

type RejectQuestOutput struct {
	Body RejectQuestResponse
}

// ---------- Accept Quest ----------

type AcceptQuestBody struct {
	PartyID string `json:"party_id,omitempty" doc:"Optional party UUID if Party Lead manages multiple parties"`
}

type AcceptQuestInput struct {
	ID   string          `path:"id" doc:"Quest UUID"`
	Body AcceptQuestBody `json:"body,omitempty"`
}

type AcceptQuestResponse struct {
	QuestID       string    `json:"quest_id"`
	PartyID       string    `json:"party_id"`
	PartyName     string    `json:"party_name"`
	Status        string    `json:"status"`
	ChatChannelID string    `json:"chat_channel_id"`
	TasksCreated  int       `json:"tasks_created"`
	AcceptedAt    time.Time `json:"accepted_at"`
	Message       string    `json:"message"`
}

type AcceptQuestOutput struct {
	Status int `json:"-" huma:"status:200"`
	Body   AcceptQuestResponse
}

