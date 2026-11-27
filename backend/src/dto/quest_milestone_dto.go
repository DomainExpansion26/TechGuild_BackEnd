package dto

import "time"

// ============================================================
// FREELANCER — Create Milestone
// ============================================================

type MilestoneTaskInput struct {
	Title       string     `json:"title" required:"true" minLength:"2" maxLength:"255" example:"Navbar Wireframe"`
	Description string     `json:"description,omitempty" maxLength:"5000" example:"Design responsive navigation component"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	AssigneeID  *string    `json:"assignee_id,omitempty" format:"uuid" doc:"Lead or active party member (defaults to the lead)"`
}

type CreateQuestMilestoneBody struct {
	Title       string               `json:"title" required:"true" minLength:"2" maxLength:"255" example:"Research & Wireframes"`
	Description string               `json:"description,omitempty" maxLength:"5000" example:"User research and low-fidelity wireframes"`
	Amount      float64              `json:"amount" required:"true" minimum:"0.01" example:"10000" doc:"Amount in rupees (major units). 10000 = ₹10,000.00 INR. NOT paise."`
	StartDate   *time.Time           `json:"start_date,omitempty"`
	DueDate     *time.Time           `json:"due_date,omitempty"`
	DueDays     int                  `json:"due_days,omitempty" minimum:"1" example:"7"`
	Tasks       []MilestoneTaskInput `json:"tasks,omitempty" maxItems:"50" doc:"Optional initial tasks"`
}

type CreateMilestoneInput struct {
	QuestID string `path:"id" format:"uuid" doc:"Quest ID"`
	Body    CreateQuestMilestoneBody
}

type CreateQuestMilestoneOutput struct {
	Status int                     `json:"-" huma:"status:201"`
	Body   MilestoneDetailResponse `json:"body"`
}

// ============================================================
// FREELANCER — Update Milestone
// ============================================================

type UpdateQuestMilestoneBody struct {
	Title       *string    `json:"title,omitempty" minLength:"2" maxLength:"255"`
	Description *string    `json:"description,omitempty" maxLength:"5000"`
	Amount      *float64   `json:"amount,omitempty" minimum:"0.01" example:"28000" doc:"Amount in rupees (major units). 28000 = ₹28,000.00 INR. NOT paise."`
	StartDate   *time.Time `json:"start_date,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	DueDays     *int       `json:"due_days,omitempty" minimum:"1"`
	// Additive task patch: omit a task to leave it untouched.
	Tasks         []MilestoneTaskPatchInput `json:"tasks,omitempty" maxItems:"50" doc:"Create (no id) or partially update (with id) tasks"`
	DeleteTaskIDs []string                  `json:"delete_task_ids,omitempty" maxItems:"50" doc:"Task IDs to delete; must belong to this milestone"`
}

// MilestoneTaskPatchInput patches one task. Only non-nil fields are written.
type MilestoneTaskPatchInput struct {
	ID          *string    `json:"id,omitempty" format:"uuid" doc:"Omit to create a new task"`
	Title       *string    `json:"title,omitempty" minLength:"2" maxLength:"255"`
	Description *string    `json:"description,omitempty" maxLength:"5000"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	AssigneeID  *string    `json:"assignee_id,omitempty" format:"uuid"`
	Status      *string    `json:"status,omitempty" enum:"todo,in_progress,completed"`
}

type UpdateQuestMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        UpdateQuestMilestoneBody
}

type UpdateQuestMilestoneOutput struct {
	Body MilestoneDetailResponse
}

// ============================================================
// FREELANCER — Delete Milestone
// ============================================================

type DeleteQuestMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type DeleteQuestMilestoneResponse struct {
	Message     string `json:"message" example:"Milestone deleted successfully"`
	MilestoneID string `json:"milestone_id"`
}

type DeleteQuestMilestoneOutput struct {
	Body DeleteQuestMilestoneResponse
}

// ============================================================
// FREELANCER — Send / Resubmit for Review
// ============================================================

type SendForReviewBody struct {
	Note string `json:"note,omitempty" maxLength:"2000" example:"Please review the deliverables"`
}

type SendForReviewInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        SendForReviewBody
}

type SendForReviewOutput struct {
	Body MilestoneDetailResponse
}

// ============================================================
// CLIENT — Approve / Request Revision
// ============================================================

type ApproveMilestoneBody struct {
	Feedback string `json:"feedback,omitempty" maxLength:"2000" example:"Looks great, approved!"`
}

type ApproveMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        ApproveMilestoneBody
}

type ApproveMilestoneOutput struct {
	Body MilestoneDetailResponse
}

type RequestMilestoneRevisionBody struct {
	Note          string `json:"note" required:"true" minLength:"5" maxLength:"2000" example:"Please adjust the color scheme to match brand guidelines"`
	AttachmentURL string `json:"attachment_url,omitempty" format:"uri" maxLength:"2048" example:"https://cdn.techguild.in/reviews/milestone-1-revision-notes.pdf"`
}

type RequestMilestoneRevisionInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	Body        RequestMilestoneRevisionBody
}

type RequestMilestoneRevisionOutput struct {
	Body MilestoneDetailResponse
}

// ============================================================
// READ — Get / List
// ============================================================

type GetQuestMilestoneInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type GetQuestMilestoneOutput struct {
	Body MilestoneDetailResponse
}

type ListQuestMilestonesInput struct {
	QuestID string `path:"id" format:"uuid" doc:"Quest ID"`
}

type ListQuestMilestonesResponse struct {
	Milestones []MilestoneSummaryItem `json:"milestones"`
	Total      int                    `json:"total"`
	Currency   string                 `json:"currency" example:"INR"`
}

type ListQuestMilestonesOutput struct {
	Body ListQuestMilestonesResponse
}

// ============================================================
// REVISIONS — History
// ============================================================

type GetMilestoneRevisionsInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type RevisionItem struct {
	ID                  string    `json:"id"`
	RequestedByID       string    `json:"requested_by_id"`
	RequestedByName     string    `json:"requested_by_name"`
	RequestedByRole     string    `json:"requested_by_role" example:"client" doc:"client or freelancer"`
	TitleSnapshot       string    `json:"title_snapshot,omitempty"`
	DescriptionSnapshot string    `json:"description_snapshot,omitempty"`
	AmountMinorSnapshot int64     `json:"amount_minor_snapshot,omitempty" example:"2500000" doc:"Amount in paise (minor units). 2500000 = ₹25,000.00 INR"`
	Note                string    `json:"note,omitempty"`
	AttachmentURL       string    `json:"attachment_url,omitempty"`
	RoundNumber         int       `json:"round_number"`
	CreatedAt           time.Time `json:"created_at"`
}

type GetMilestoneRevisionsResponse struct {
	MilestoneID string         `json:"milestone_id"`
	Total       int            `json:"total"`
	Revisions   []RevisionItem `json:"revisions"`
}

type GetMilestoneRevisionsOutput struct {
	Body GetMilestoneRevisionsResponse
}

// ============================================================
// SHARED — Response Models
// ============================================================

type MilestoneTaskSummary struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status" example:"todo"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

type MilestoneSummaryItem struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	AmountMinor   int64      `json:"amount_minor" example:"2500000" doc:"Amount in paise (minor units). 2500000 = ₹25,000.00 INR"`
	Amount        float64    `json:"amount" example:"25000" doc:"Amount in rupees (major units). 25000 = ₹25,000.00 INR. NOT paise."`
	Currency      string     `json:"currency" example:"INR"`
	DueDays       int        `json:"due_days"`
	OrderIndex    int        `json:"order_index"`
	Status        string     `json:"status"`
	TaskCount     int        `json:"task_count"`
	RevisionCount int        `json:"revision_count"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type MilestoneDetailResponse struct {
	ID          string `json:"id"`
	QuestID     string `json:"quest_id"`
	CreatedByID string `json:"created_by_id"`

	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	AmountMinor int64   `json:"amount_minor" example:"2500000" doc:"Amount in paise (minor units). 2500000 = ₹25,000.00 INR"`
	Amount      float64 `json:"amount" example:"25000" doc:"Amount in rupees (major units). 25000 = ₹25,000.00 INR. NOT paise."`
	Currency    string  `json:"currency" example:"INR"`
	DueDays     int     `json:"due_days"`
	OrderIndex  int     `json:"order_index"`

	Status         string `json:"status"`
	RevisionCount  int    `json:"revision_count"`
	ClientFeedback string `json:"client_feedback,omitempty"`

	StartDate  *time.Time `json:"start_date,omitempty"`
	DueDate    *time.Time `json:"due_date,omitempty"`
	AgreedAt   *time.Time `json:"agreed_at,omitempty"`
	FundedAt   *time.Time `json:"funded_at,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`

	Tasks     []MilestoneTaskSummary `json:"tasks"`
	Revisions []RevisionItem         `json:"revisions,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------- Roadmap Bulk Actions ----------

type SubmitRoadmapBody struct {
	Note string `json:"note,omitempty" maxLength:"2000" example:"Here is the complete project roadmap. Please review."`
}

type SubmitRoadmapInput struct {
	ID   string            `path:"id" format:"uuid" doc:"Quest UUID"`
	Body SubmitRoadmapBody `json:"body,omitempty"`
}

type ApproveRoadmapBody struct {
	Feedback string `json:"feedback,omitempty" maxLength:"2000" example:"Roadmap looks perfect. Let's start with Milestone 1."`
}

type ApproveRoadmapInput struct {
	ID   string             `path:"id" format:"uuid" doc:"Quest UUID"`
	Body ApproveRoadmapBody `json:"body,omitempty"`
}

type RoadmapActionResponse struct {
	Message                string `json:"message"`
	UpdatedMilestonesCount int    `json:"updated_milestones_count"`
}

type SubmitRoadmapOutput struct {
	Body RoadmapActionResponse
}

type ApproveRoadmapOutput struct {
	Body RoadmapActionResponse
}
