package dto

import "time"

// ============ Apply / Bid ============

type ApplyQuestBody struct {
	CoverLetter       string  `json:"cover_letter,omitempty" maxLength:"5000" doc:"Optional. Empty is allowed for 1-click apply." example:"I have 5+ years in UI/UX and shipped multiple FinTech dashboards."`
	ProposedBudget    float64 `json:"proposed_budget,omitempty" minimum:"0" doc:"Amount in rupees (major units). 72000 = ₹72,000.00 INR. NOT paise. Omit or send 0 to accept the full quest budget." example:"72000"`
	Currency          string  `json:"currency,omitempty" maxLength:"10" example:"INR"`
	EstimatedDuration string  `json:"estimated_duration,omitempty" maxLength:"100" example:"18 days"`
	PartyID           string  `json:"party_id,omitempty" maxLength:"36" doc:"Optional: apply as agency (party UUID). Omit for individual."`
}

type ApplyQuestInput struct {
	ID   string         `path:"id" format:"uuid" doc:"Quest UUID"`
	Body ApplyQuestBody `json:"body"`
}

type ApplyQuestResponse struct {
	ApplicationID         string    `json:"application_id"`
	QuestID               string    `json:"quest_id"`
	Status                string    `json:"status"`
	ProposedBudgetINR     float64   `json:"proposed_budget_inr" example:"72000" doc:"Amount in rupees (major units). 72000 = ₹72,000.00 INR. NOT paise."`
	EstimatedNetPayoutINR float64   `json:"estimated_net_payout_inr" example:"68400" doc:"Amount in rupees (major units). 68400 = ₹68,400.00 INR. NOT paise. After 5% freelancer platform fee"`
	CreatedAt             time.Time `json:"created_at"`
	Message               string    `json:"message"`
}

type ApplyQuestOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   ApplyQuestResponse
}

// ============ List Applications (Client view) ============

type ListApplicationsInput struct {
	ID     string `path:"id" format:"uuid" doc:"Quest UUID"`
	Status string `query:"status" maxLength:"50" doc:"Filter by status (pending, accepted, rejected, withdrawn)"`
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
	ProposedBudgetINR float64   `json:"proposed_budget_inr" example:"72000" doc:"Amount in rupees (major units). 72000 = ₹72,000.00 INR. NOT paise."`
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
	ID            string `path:"id" format:"uuid" doc:"Quest UUID"`
	ApplicationID string `path:"application_id" format:"uuid" doc:"Application UUID"`
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
	Reason string `json:"reason,omitempty" maxLength:"1000" example:"Looking for more React experience"`
}

type RejectApplicationInput struct {
	ID            string                `path:"id" format:"uuid"`
	ApplicationID string                `path:"application_id" format:"uuid"`
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
	ID            string `path:"id" format:"uuid"`
	ApplicationID string `path:"application_id" format:"uuid"`
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
	Status string `query:"status" maxLength:"50" doc:"Filter by quest status (default: in_progress)"`
}

type MyWorkItem struct {
	QuestID        string     `json:"quest_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	BudgetINR      float64    `json:"budget_inr" example:"75000" doc:"Amount in rupees (major units). 75000 = ₹75,000.00 INR. NOT paise."`
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
