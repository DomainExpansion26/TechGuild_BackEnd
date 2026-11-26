package dto

import "time"

// ------------------------------------------------------------
// Banner stages (derived, never stored)
// ------------------------------------------------------------

const (
	PartyTMDStageInitial            = "initial"
	PartyTMDStageMilestonesCreated  = "milestones_created"
	PartyTMDStageAwaitingClient     = "awaiting_client_approval"
	PartyTMDStageApprovedAssignTeam = "approved_assign_team"
	PartyTMDStageInProgress         = "in_progress"
)

const (
	PartyTMDRoleTeamLead   = "team_lead"
	PartyTMDRoleTeamMember = "team_member"
)

// ------------------------------------------------------------
// Dashboard
// ------------------------------------------------------------

type PartyTMDDashboardInput struct {
	QuestID string `query:"quest_id" format:"uuid" required:"true" doc:"Quest ID"`
}

type PartyTMDDashboardResponse struct {
	QuestID            string                     `json:"quest_id"`
	CallerRole         string                     `json:"caller_role" enum:"team_lead,team_member"`
	BannerStage        string                     `json:"banner_stage" enum:"initial,milestones_created,awaiting_client_approval,approved_assign_team,in_progress"`
	QuestTitle         string                     `json:"quest_title"`
	QuestStatus        string                     `json:"quest_status"`
	ClientName         string                     `json:"client_name"`
	KPIs               PartyTMDKPIs               `json:"kpis"`
	MilestoneBreakdown PartyTMDMilestoneBreakdown `json:"milestone_breakdown"`
	Milestones         []PartyTMDMilestoneItem    `json:"milestones"`
	RecentActivity     []PartyTMDActivityItem     `json:"recent_activity" maxItems:"20"`
}

type PartyTMDDashboardOutput struct {
	Body PartyTMDDashboardResponse
}

type PartyTMDKPIs struct {
	ActiveQuests       int `json:"active_quests"`
	TotalMilestones    int `json:"total_milestones"`
	AssignedMilestones int `json:"assigned_milestones" doc:"Team lead: milestones with >=1 assignee. Team member: milestones assigned to the caller."`
	TeamMembers        int `json:"team_members"`
	OverallProgress    int `json:"overall_progress" doc:"0-100, averaged across milestones"`
}

type PartyTMDMilestoneBreakdown struct {
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
}

// PartyTMDMilestoneItem is a full milestone card/row so both roles can render
// the table from a single dashboard call.
type PartyTMDMilestoneItem struct {
	ID                 string                   `json:"id"`
	OrderIndex         int                      `json:"order_index"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	AmountMinor        int64                    `json:"amount_minor"`
	Status             string                   `json:"status"`
	Progress           int                      `json:"progress" doc:"0-100"`
	StartDate          *time.Time               `json:"start_date,omitempty"`
	EndDate            *time.Time               `json:"end_date,omitempty"`
	AssignedMembers    []PartyTMDAssignmentItem `json:"assigned_members"`
	Tasks              []PartyTMDTaskItem       `json:"tasks"`
	AttachmentsCount   int                      `json:"attachments_count"`
	IsAssignedToCaller bool                     `json:"is_assigned_to_caller"`
}

type PartyTMDTaskItem struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	AssigneeID string     `json:"assignee_id,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
}

type PartyTMDActivityItem struct {
	ID          string    `json:"id"`
	ActionType  string    `json:"action_type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ActorID     string    `json:"actor_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// ------------------------------------------------------------
// Active quests (multi-quest switcher)
// ------------------------------------------------------------

type PartyTMDActiveQuestsInput struct{}

type PartyTMDActiveQuestsResponse struct {
	Quests []PartyTMDQuestItem `json:"quests"`
	Total  int                 `json:"total"`
}

type PartyTMDActiveQuestsOutput struct {
	Body PartyTMDActiveQuestsResponse
}

type PartyTMDQuestItem struct {
	QuestID        string `json:"quest_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	ClientName     string `json:"client_name"`
	MilestoneCount int    `json:"milestone_count"`
	Progress       int    `json:"progress"`
	CallerRole     string `json:"caller_role" enum:"team_lead,team_member"`
}

// ------------------------------------------------------------
// Members overview
// ------------------------------------------------------------

type PartyTMDMembersInput struct {
	QuestID string `query:"quest_id" format:"uuid" required:"true" doc:"Quest ID"`
}

type PartyTMDMembersResponse struct {
	TotalMembers       int                  `json:"total_members"`
	UnassignedMembers  int                  `json:"unassigned_members"`
	TotalMilestones    int                  `json:"total_milestones"`
	AssignedMilestones int                  `json:"assigned_milestones"`
	Members            []PartyTMDMemberItem `json:"members"`
}

type PartyTMDMembersOutput struct {
	Body PartyTMDMembersResponse
}

type PartyTMDMemberItem struct {
	MemberID           string                 `json:"member_id"`
	UserID             string                 `json:"user_id"`
	Name               string                 `json:"name"`
	Role               string                 `json:"role" enum:"team_lead,team_member"`
	Rank               string                 `json:"rank"`
	Skills             []string               `json:"skills"`
	AssignmentStatus   string                 `json:"assignment_status" enum:"assigned,unassigned"`
	AssignedMilestones []PartyTMDMilestoneRef `json:"assigned_milestones"`
}

type PartyTMDMilestoneRef struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// ------------------------------------------------------------
// Milestone assignment
// ------------------------------------------------------------

type AssignMilestoneBody struct {
	MemberIDs []string   `json:"member_ids" required:"true" minItems:"1" maxItems:"50" doc:"Active party member user IDs (solo quests: the lead)"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Note      string     `json:"note,omitempty" maxLength:"2000"`
}

type AssignMilestoneInput struct {
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        AssignMilestoneBody
}

type AssignMilestoneResponse struct {
	MilestoneID string                   `json:"milestone_id"`
	Assignments []PartyTMDAssignmentItem `json:"assignments"`
}

type AssignMilestoneOutput struct {
	Body AssignMilestoneResponse
}

type PartyTMDAssignmentItem struct {
	ID        string     `json:"id"`
	MemberID  string     `json:"member_id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Note      string     `json:"note,omitempty"`
}

type ListMilestoneAssignmentsInput struct {
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type ListMilestoneAssignmentsResponse struct {
	MilestoneID string                   `json:"milestone_id"`
	Assignments []PartyTMDAssignmentItem `json:"assignments"`
	Total       int                      `json:"total"`
}

type ListMilestoneAssignmentsOutput struct {
	Body ListMilestoneAssignmentsResponse
}
