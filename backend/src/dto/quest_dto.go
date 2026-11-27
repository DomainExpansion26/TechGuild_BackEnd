package dto

import "time"

type CreateQuestSkillInput struct {
	Name             string `json:"name" required:"true" minLength:"1" maxLength:"100" example:"Figma"`
	ProficiencyLevel string `json:"proficiency_level,omitempty" maxLength:"50" example:"expert"`
	IsCore           bool   `json:"is_core" example:"true"`
}

type CreateQuestLanguageInput struct {
	Name        string `json:"name" required:"true" minLength:"1" maxLength:"100" example:"English"`
	Proficiency string `json:"proficiency,omitempty" maxLength:"50" example:"Native/Bilingual"`
}

type CreateQuestBody struct {
	Title           string                     `json:"title" required:"true" minLength:"3" maxLength:"100" example:"Website Redesign for FinTech Startup"`
	Description     string                     `json:"description" required:"true" minLength:"10" maxLength:"5000" example:"Project focuses on modernizing the UI and streamlining the UX for our digital banking dashboard."`
	Category        string                     `json:"category" required:"true" minLength:"1" maxLength:"100" example:"UI/UX Design"`
	Industry        string                     `json:"industry,omitempty" maxLength:"100" example:"FinTech"`
	Complexity      string                     `json:"complexity,omitempty" maxLength:"50" example:"intermediate"`
	BudgetMinor     int64                      `json:"budget_minor" required:"true" minimum:"100000" doc:"Total quest budget in minor units (paise). ₹1,000 = 100000" example:"7500000"`
	TimelineDays    int                        `json:"timeline_days" required:"true" minimum:"1" example:"20"`
	QuestType       string                     `json:"quest_type,omitempty" maxLength:"50" example:"standard"`
	ExperienceLevel string                     `json:"experience_level,omitempty" maxLength:"50" example:"senior"`
	SoftSkills      []string                   `json:"soft_skills,omitempty" maxItems:"30" maxLength:"100"`
	Languages       []CreateQuestLanguageInput `json:"languages,omitempty" maxItems:"50"`
	Skills          []CreateQuestSkillInput    `json:"skills" required:"true" minItems:"1" maxItems:"50"`
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
	BudgetMinor     int64     `json:"budget_minor" example:"7500000" doc:"Amount in paise (minor units). 7500000 = ₹75,000.00 INR"`
	RequiredMinRank string    `json:"required_min_rank" example:"B"`
	CreatedAt       time.Time `json:"created_at" example:"2026-09-03T20:15:00Z"`
}

// CreateQuestOutput wraps the Huma response body with HTTP status 201.
type CreateQuestOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   CreateQuestResponse
}

// ---------- Publish Quest ----------

type PublishQuestBody struct {
	FundingMethod string `json:"funding_method,omitempty" maxLength:"50" doc:"DEPRECATED and ignored. Publishing has no funding prerequisite; escrow is funded per-milestone via the milestone fund endpoint." example:"escrow_deposit"`
	FundEscrowNow *bool  `json:"fund_escrow_now,omitempty" doc:"DEPRECATED and ignored. No funds are moved at publish time."`
}

type PublishQuestInput struct {
	ID   string           `path:"id" format:"uuid" doc:"Quest UUID"`
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
	Cursor         string  `query:"cursor" maxLength:"100" doc:"Opaque pagination cursor (pass next_cursor from the previous page)"`
	Limit          int     `query:"limit" minimum:"1" maximum:"50" doc:"Number of records to return (default 10, max 50)"`
	Search         string  `query:"search" maxLength:"200" doc:"Case-insensitive match on quest title or description"`
	Category       string  `query:"category" maxLength:"100" doc:"Filter by quest category"`
	Complexity     string  `query:"complexity" maxLength:"50" doc:"Filter by quest complexity (beginner, intermediate, advanced, enterprise)"`
	QuestType      string  `query:"quest_type" maxLength:"50" doc:"Filter by quest type (standard, emergency, long_term, multi_skill)"`
	Skills         string  `query:"skills" maxLength:"500" doc:"Comma-separated skill names; matches quests requiring ANY of them"`
	TimelineBucket string  `query:"timeline_bucket" enum:"under_7,7_15,15_30,30_plus" doc:"Filter by timeline bucket"`
	ClientRank     string  `query:"client_rank" maxLength:"10" doc:"Filter by the posting client's rank"`
	MinBudget      float64 `query:"min_budget" minimum:"0" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise. Filter by minimum budget."`
	MaxBudget      float64 `query:"max_budget" minimum:"0" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise. Filter by maximum budget."`
	Status         string  `query:"status" maxLength:"50" doc:"Filter by status (open, in_progress, completed, etc.)"`
	Sort           string  `query:"sort" enum:"newest,budget_desc,budget_asc,emergency_first" doc:"Sort order (default newest)"`
}

type QuestListItem struct {
	ID              string     `json:"id" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	Title           string     `json:"title" example:"Website Redesign for FinTech Startup"`
	Description     string     `json:"description" example:"Redesign of dashboard"`
	Category        string     `json:"category" example:"UI/UX Design"`
	Complexity      string     `json:"complexity" example:"intermediate"`
	QuestType       string     `json:"quest_type" example:"standard"`
	BudgetINR       float64    `json:"budget_inr" example:"75000" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise."`
	TimelineDays    int        `json:"timeline_days" example:"20"`
	RequiredMinRank string     `json:"required_min_rank" example:"B"`
	Status          string     `json:"status" example:"open"`
	ClientID        string     `json:"client_id" example:"6f1d9b3a-8c2e-4a57-9b1f-3d6c0e8a5b24"`
	ClientName      string     `json:"client_name" example:"Acme Corp"`
	ClientRank      string     `json:"client_rank" example:"A"`
	Skills          []string   `json:"skills" example:"[\"Figma\", \"React\"]"`
	MilestoneCount  int        `json:"milestone_count" example:"4"`
	ApplicantsCount int        `json:"applicants_count" example:"12"`
	ViewsCount      int64      `json:"views_count" example:"340"`
	IsSaved         bool       `json:"is_saved" example:"false"`
	MatchPercentage int        `json:"match_percentage" example:"66"`
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
	ID string `path:"id" format:"uuid" doc:"Quest UUID"`
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
	BudgetINR       float64                `json:"budget_inr" example:"75000" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise."`
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
	ApplicantsCount int                    `json:"applicants_count"`
	ViewsCount      int64                  `json:"views_count"`
	IsSaved         bool                   `json:"is_saved"`
	MatchPercentage int                    `json:"match_percentage"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type GetQuestDetailsOutput struct {
	Body QuestDetailsResponse
}

// ---------- Reject / Cancel Quest ----------

type RejectQuestBody struct {
	Reason string `json:"reason,omitempty" maxLength:"1000" example:"Budget re-allocated"`
}

type RejectQuestInput struct {
	ID   string          `path:"id" format:"uuid" doc:"Quest UUID"`
	Body RejectQuestBody `json:"body,omitempty"`
}

type RejectQuestResponse struct {
	QuestID            string  `json:"quest_id"`
	Status             string  `json:"status"`
	EscrowRefunded     bool    `json:"escrow_refunded"`
	EscrowRefundAmount float64 `json:"escrow_refund_amount" example:"75000" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise."`
	Message            string  `json:"message"`
}

type RejectQuestOutput struct {
	Body RejectQuestResponse
}

// ---------- Update Quest (draft only) ----------

type UpdateQuestBody struct {
	Title           *string                     `json:"title,omitempty" minLength:"3" maxLength:"100"`
	Description     *string                     `json:"description,omitempty" minLength:"10" maxLength:"5000"`
	Category        *string                     `json:"category,omitempty" minLength:"1" maxLength:"100"`
	Industry        *string                     `json:"industry,omitempty" maxLength:"100"`
	Complexity      *string                     `json:"complexity,omitempty" maxLength:"50"`
	BudgetMinor     *int64                      `json:"budget_minor,omitempty" minimum:"100000" example:"9000000" doc:"Amount in paise (minor units). 9000000 = ₹90,000.00 INR"`
	TimelineDays    *int                        `json:"timeline_days,omitempty" minimum:"1"`
	QuestType       *string                     `json:"quest_type,omitempty" maxLength:"50"`
	ExperienceLevel *string                     `json:"experience_level,omitempty" maxLength:"50"`
	SoftSkills      *[]string                   `json:"soft_skills,omitempty" maxItems:"30"`
	Languages       *[]CreateQuestLanguageInput `json:"languages,omitempty" maxItems:"50"`
	Skills          *[]CreateQuestSkillInput    `json:"skills,omitempty" maxItems:"50" doc:"When present, replaces the quest's skill list"`
}

type UpdateQuestInput struct {
	ID   string `path:"id" format:"uuid" doc:"Quest UUID"`
	Body UpdateQuestBody
}

type UpdateQuestResponse struct {
	QuestID         string    `json:"quest_id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	BudgetMinor     int64     `json:"budget_minor" example:"9000000" doc:"Amount in paise (minor units). 9000000 = ₹90,000.00 INR"`
	RequiredMinRank string    `json:"required_min_rank"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpdateQuestOutput struct {
	Body UpdateQuestResponse
}

// ---------- Save / Bookmark Quest ----------

type SaveQuestInput struct {
	ID string `path:"id" format:"uuid" doc:"Quest UUID"`
}

type SaveQuestResponse struct {
	QuestID string `json:"quest_id"`
	Saved   bool   `json:"saved"`
}

type SaveQuestOutput struct {
	Body SaveQuestResponse
}

type ListSavedQuestsInput struct{}

type ListSavedQuestsOutput struct {
	Body ListQuestsResponse
}

// ---------- Freelancer Quest Board Summary ----------

type QuestBoardSummaryInput struct{}

type QuestBoardKPIs struct {
	AppliedCount      int `json:"applied_count"`
	UnderReviewCount  int `json:"under_review_count"`
	ShortlistedCount  int `json:"shortlisted_count"`
	ActiveQuestsCount int `json:"active_quests_count"`
}

type QuestBoardApplicationItem struct {
	QuestID    string    `json:"quest_id"`
	QuestTitle string    `json:"quest_title"`
	BudgetINR  float64   `json:"budget_inr" example:"75000" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise."`
	Status     string    `json:"status"`
	AppliedAt  time.Time `json:"applied_at"`
}

type QuestBoardSummaryResponse struct {
	KPIs               QuestBoardKPIs              `json:"kpis"`
	RecentApplications []QuestBoardApplicationItem `json:"recent_applications"`
	RecommendedQuests  []QuestListItem             `json:"recommended_quests"`
}

type QuestBoardSummaryOutput struct {
	Body QuestBoardSummaryResponse
}
