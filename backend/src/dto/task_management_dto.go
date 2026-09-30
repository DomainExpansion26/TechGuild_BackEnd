package dto

import "time"

// FreelancerSummary provides client-facing details of the assigned talent.
type FreelancerSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role,omitempty"`
}

// ---------- 1. Metrics ----------

type TaskManagementMetricsInput struct {
	QuestID string `query:"quest_id" doc:"Optional Quest ID to scope task and deadline counts"`
}

type TaskManagementMetricsResponse struct {
	ActiveQuests      int `json:"active_quests"`
	PendingReviews    int `json:"pending_reviews"`
	TasksInProgress   int `json:"tasks_in_progress"`
	UpcomingDeadlines int `json:"upcoming_deadlines"`
	CompletedQuests   int `json:"completed_quests"`
}

type TaskManagementMetricsOutput struct {
	Body TaskManagementMetricsResponse
}

// ---------- 2. Active Quests List (Switcher / Banner) ----------

type GetActiveQuestsInput struct{}

type ActiveQuestSummaryItem struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Category   string             `json:"category"`
	Budget     float64            `json:"budget"`
	Currency   string             `json:"currency"`
	Freelancer *FreelancerSummary `json:"freelancer,omitempty"`
	Progress   int                `json:"progress"`
	Status     string             `json:"status"`
}

type ActiveQuestsListResponse struct {
	TotalActive int                      `json:"total_active"`
	Quests      []ActiveQuestSummaryItem `json:"quests"`
}

type ActiveQuestsListOutput struct {
	Body ActiveQuestsListResponse
}

// ---------- 3. Quest Dashboard Overview & Donut Chart ----------

type GetQuestDashboardInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
}

type QuestDashboardOverview struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Category        string             `json:"category"`
	Budget          float64            `json:"budget"`
	Currency        string             `json:"currency"`
	Status          string             `json:"status"`
	Freelancer      *FreelancerSummary `json:"freelancer,omitempty"`
	CurrentStep     int                `json:"current_step"`
	HasTasksCreated bool               `json:"has_tasks_created"`
	CanApproveQuest bool               `json:"can_approve_quest"`
	OverallProgress int                `json:"overall_progress"`
}

type MilestonesOverviewStats struct {
	CompletionPercentage int `json:"completion_percentage"`
	TotalMilestones      int `json:"total_milestones"`
	CompletedMilestones  int `json:"completed_milestones"`
	InProgressMilestones int `json:"in_progress_milestones"`
	PendingMilestones    int `json:"pending_milestones"`
}

type RecentActivityItem struct {
	ID          string    `json:"id"`
	ActionType  string    `json:"action_type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type QuestDashboardResponse struct {
	Quest                   QuestDashboardOverview  `json:"quest"`
	MilestonesOverview      MilestonesOverviewStats `json:"milestones_overview"`
	RecentActivitiesPreview []RecentActivityItem    `json:"recent_activities_preview"`
}

type QuestDashboardOutput struct {
	Body QuestDashboardResponse
}

// ---------- 4. Milestones Table ----------

type GetQuestMilestonesTableInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
}

type QuestMilestoneRowItem struct {
	SlNo        int     `json:"sl_no"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	TaskCount   int     `json:"task_count"`
	Progress    int     `json:"progress"`
	StartDate   *string `json:"start_date,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	DueDays     int     `json:"due_days"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	OrderIndex  int     `json:"order_index"`
}

type QuestMilestonesTableResponse struct {
	Milestones  []QuestMilestoneRowItem `json:"milestones"`
	TotalAmount float64                 `json:"total_amount"`
	Currency    string                  `json:"currency"`
}

type QuestMilestonesTableOutput struct {
	Body QuestMilestonesTableResponse
}

// ---------- 5. Add Milestone Modal ----------

type AddMilestoneRequestBody struct {
	Title       string     `json:"title" huma:"required,minLength=2,maxLength=255" example:"Responsive Design & Mobile Optimization"`
	Description string     `json:"description" example:"Complete responsive layouts for tablet and mobile."`
	Amount      float64    `json:"amount" huma:"required,minimum=0.01" example:"10000"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	DueDays     int        `json:"due_days,omitempty" example:"7"`
}

type AddMilestoneInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
	Body    AddMilestoneRequestBody
}

type AddMilestoneResponse struct {
	Message        string                `json:"message"`
	Milestone      QuestMilestoneRowItem `json:"milestone"`
	NewTotalBudget float64               `json:"new_total_budget"`
}

type AddMilestoneOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   AddMilestoneResponse
}

// ---------- 6. Delete Milestone Modal ----------

type DeleteMilestoneTMDInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
}

type DeleteMilestoneTMDResponse struct {
	Message     string `json:"message"`
	MilestoneID string `json:"milestone_id"`
}

type DeleteMilestoneTMDOutput struct {
	Body DeleteMilestoneTMDResponse
}

// ---------- 7. View Tasks Modal ----------

type GetMilestoneTasksInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
}

type MilestoneTaskItem struct {
	SlNo           int     `json:"sl_no"`
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Assignee       string  `json:"assignee"`
	AssigneeAvatar string  `json:"assignee_avatar,omitempty"`
	StartDate      *string `json:"start_date,omitempty"`
	EndDate        *string `json:"end_date,omitempty"`
	Status         string  `json:"status"`
}

type MilestoneTasksResponse struct {
	MilestoneID   string              `json:"milestone_id"`
	MilestoneName string              `json:"milestone_name"`
	Tasks         []MilestoneTaskItem `json:"tasks"`
}

type MilestoneTasksOutput struct {
	Body MilestoneTasksResponse
}

// ---------- 8. Approve Milestone Modal ----------

type ApproveMilestoneTMDRequestBody struct {
	Feedback string `json:"feedback,omitempty" example:"Deliverables approved and tested"`
}

type ApproveMilestoneTMDInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        ApproveMilestoneTMDRequestBody
}

type ApproveMilestoneTMDResponse struct {
	Message              string `json:"message"`
	MilestoneID          string `json:"milestone_id"`
	Status               string `json:"status"`
	OverallQuestProgress int    `json:"overall_quest_progress"`
	CanApproveQuest      bool   `json:"can_approve_quest"`
}

type ApproveMilestoneTMDOutput struct {
	Body ApproveMilestoneTMDResponse
}

// ---------- 9. Cancel Milestone Approval Modal ----------

type CancelMilestoneApprovalTMDRequestBody struct {
	Reason string `json:"reason" huma:"required,minLength=5" example:"Revisions required on mobile layouts"`
}

type CancelMilestoneApprovalTMDInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        CancelMilestoneApprovalTMDRequestBody
}

type CancelMilestoneApprovalTMDResponse struct {
	Message              string `json:"message"`
	MilestoneID          string `json:"milestone_id"`
	Status               string `json:"status"`
	OverallQuestProgress int    `json:"overall_quest_progress"`
	CanApproveQuest      bool   `json:"can_approve_quest"`
}

type CancelMilestoneApprovalTMDOutput struct {
	Body CancelMilestoneApprovalTMDResponse
}

// ---------- 10. Approve Quest Modal ----------

type ApproveQuestTMDRequestBody struct {
	Rating   int    `json:"rating" huma:"required,minimum=1,maximum=5" example:"5"`
	Feedback string `json:"feedback" huma:"required,minLength=5" example:"Outstanding work delivered on time"`
}

type ApproveQuestTMDInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
	Body    ApproveQuestTMDRequestBody
}

type ApproveQuestTMDResponse struct {
	Message     string    `json:"message"`
	QuestID     string    `json:"quest_id"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

type ApproveQuestTMDOutput struct {
	Body ApproveQuestTMDResponse
}

// ---------- 11. Cancel Quest Approval Modal ----------

type CancelQuestApprovalTMDRequestBody struct {
	Reason string `json:"reason" huma:"required,minLength=5" example:"Post-delivery integration defect requires patch"`
}

type CancelQuestApprovalTMDInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
	Body    CancelQuestApprovalTMDRequestBody
}

type CancelQuestApprovalTMDResponse struct {
	Message string `json:"message"`
	QuestID string `json:"quest_id"`
	Status  string `json:"status"`
}

type CancelQuestApprovalTMDOutput struct {
	Body CancelQuestApprovalTMDResponse
}

// ---------- 12. Recent Activities Drawer ----------

type GetQuestActivitiesTMDInput struct {
	QuestID string `path:"quest_id" doc:"Quest ID"`
	Limit   int    `query:"limit" doc:"Maximum items" default:"20"`
	Offset  int    `query:"offset" doc:"Offset items" default:"0"`
}

type QuestActivitiesTMDResponse struct {
	QuestID    string               `json:"quest_id"`
	Total      int64                `json:"total"`
	Activities []RecentActivityItem `json:"activities"`
}

type QuestActivitiesTMDOutput struct {
	Body QuestActivitiesTMDResponse
}

// ---------- 13. Task Operations (Create Task & Status Update) ----------

type CreateMilestoneTaskRequestBody struct {
	Title       string     `json:"title" huma:"required,minLength=2,maxLength=255" example:"Navbar Wireframe"`
	Description string     `json:"description,omitempty" example:"Design responsive navigation component"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	AssigneeID  *string    `json:"assignee_id,omitempty"`
	Status      string     `json:"status,omitempty" example:"todo"`
}

type CreateMilestoneTaskInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
	Body        CreateMilestoneTaskRequestBody
}

type CreateMilestoneTaskOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   MilestoneTaskItem
}

type UpdateTaskStatusRequestBody struct {
	Status string `json:"status" huma:"required" example:"in_progress"`
}

type UpdateTaskStatusInput struct {
	TaskID string `path:"task_id" doc:"Task ID"`
	Body   UpdateTaskStatusRequestBody
}

type UpdateTaskStatusOutput struct {
	Body MilestoneTaskItem
}
