package dto

import "time"

// ============ Apply / Bid ============

type ApplyQuestBody struct {
	CoverLetter       string  `json:"cover_letter" huma:"required,minLength=20,maxLength=5000" example:"I have 5+ years in UI/UX and shipped multiple FinTech dashboards."`
	ProposedBudget    float64 `json:"proposed_budget" huma:"required,minimum=0.01" doc:"Proposed budget in INR" example:"72000"`
	Currency          string  `json:"currency,omitempty" example:"INR"`
	EstimatedDuration string  `json:"estimated_duration,omitempty" example:"18 days"`
	PartyID           string  `json:"party_id,omitempty" doc:"Optional: apply as agency (party UUID). Omit for individual."`
}

type ApplyQuestInput struct {
	ID   string         `path:"id" doc:"Quest UUID"`
	Body ApplyQuestBody `json:"body"`
}

type ApplyQuestResponse struct {
	ApplicationID         string    `json:"application_id"`
	QuestID               string    `json:"quest_id"`
	Status                string    `json:"status"`
	ProposedBudgetINR     float64   `json:"proposed_budget_inr"`
	EstimatedNetPayoutINR float64   `json:"estimated_net_payout_inr" doc:"After 5% freelancer platform fee"`
	CreatedAt             time.Time `json:"created_at"`
	Message               string    `json:"message"`
}

type ApplyQuestOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   ApplyQuestResponse
}

// ============ List Applications (Client view) ============

type ListApplicationsInput struct {
	ID     string `path:"id" doc:"Quest UUID"`
	Status string `query:"status" doc:"Filter by status (pending, accepted, rejected, withdrawn)"`
}

type ApplicationItem struct {
	ID                string    `json:"id"`
	FreelancerID      string    `json:"freelancer_id"`
	FreelancerName    string    `json:"freelancer_name"`
	FreelancerRank    string    `json:"freelancer_rank"`
	PartyID           *string   `json:"party_id,omitempty"`
	PartyName         *string   `json:"party_name,omitempty"`
	IsAgency          bool      `json:"is_agency"`
	CoverLetter       string    `json:"cover_letter"`
	ProposedBudgetINR float64   `json:"proposed_budget_inr"`
	Currency          string    `json:"currency"`
	EstimatedDuration string    `json:"estimated_duration"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

type ListApplicationsResponse struct {
	Applications []ApplicationItem `json:"applications"`
	Total        int               `json:"total"`
}

type ListApplicationsOutput struct {
	Body ListApplicationsResponse
}

// ============ Accept Application (Client picks winner) ============

type AcceptApplicationInput struct {
	ID            string `path:"id" doc:"Quest UUID"`
	ApplicationID string `path:"application_id" doc:"Application UUID"`
}

type AcceptApplicationResponse struct {
	ApplicationID     string    `json:"application_id"`
	QuestID           string    `json:"quest_id"`
	FreelancerID      string    `json:"freelancer_id"`
	FreelancerName    string    `json:"freelancer_name"`
	PartyID           *string   `json:"party_id,omitempty"`
	PartyName         *string   `json:"party_name,omitempty"`
	IsAgency          bool      `json:"is_agency"`
	ApplicationStatus string    `json:"application_status"`
	QuestStatus       string    `json:"quest_status"`
	ChatChannelID     string    `json:"chat_channel_id"`
	AutoRejectedCount int       `json:"auto_rejected_count"`
	AcceptedAt        time.Time `json:"accepted_at"`
	Message           string    `json:"message"`
}

type AcceptApplicationOutput struct {
	Status int `json:"-" huma:"status:200"`
	Body   AcceptApplicationResponse
}

// ============ Reject Application ============

type RejectApplicationBody struct {
	Reason string `json:"reason,omitempty" example:"Looking for more React experience"`
}

type RejectApplicationInput struct {
	ID            string                `path:"id"`
	ApplicationID string                `path:"application_id"`
	Body          RejectApplicationBody `json:"body,omitempty"`
}

type RejectApplicationResponse struct {
	ApplicationID string `json:"application_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

type RejectApplicationOutput struct {
	Body RejectApplicationResponse
}

// ============ Withdraw Application (Freelancer) ============

type WithdrawApplicationInput struct {
	ID            string `path:"id"`
	ApplicationID string `path:"application_id"`
}

type WithdrawApplicationResponse struct {
	ApplicationID string `json:"application_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

type WithdrawApplicationOutput struct {
	Body WithdrawApplicationResponse
}

// ============ My Work (Freelancer task-management) ============

type MyWorkInput struct {
	Status string `query:"status" doc:"Filter by quest status (default: in_progress)"`
}

type MyWorkItem struct {
	QuestID        string     `json:"quest_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	BudgetINR      float64    `json:"budget_inr"`
	Status         string     `json:"status"`
	ClientID       string     `json:"client_id"`
	ClientName     string     `json:"client_name"`
	IsAgency       bool       `json:"is_agency"`
	ChatChannelID  *string    `json:"chat_channel_id,omitempty"`
	MilestoneCount int        `json:"milestone_count"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
}

type MyWorkResponse struct {
	Quests []MyWorkItem `json:"quests"`
	Total  int          `json:"total"`
}

type MyWorkOutput struct {
	Body MyWorkResponse
}
