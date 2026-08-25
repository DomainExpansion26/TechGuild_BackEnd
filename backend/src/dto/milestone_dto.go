package dto

// create milestone request and response

type CreateMilestoneRequest struct {
	ContractID  string  `json:"contract_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title       string  `json:"title" binding:"required" example:"Design Phase"`
	Description string  `json:"description" example:"Complete UI/UX design for the homepage"`
	Amount      float64 `json:"amount" binding:"required" example:"1500.00"`
	DueDate     string  `json:"due_date" example:"2026-03-01T00:00:00Z"`
}

type CreateMilestoneResponse struct {
	Message     string `json:"message" example:"Milestone created successfully"`
	MilestoneID string `json:"milestone_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

//update milestone request and response

type UpdateMilestoneRequest struct {
	Title       string  `json:"title" example:"Design Phase v2"`
	Description string  `json:"description" example:"Updated UI/UX design with feedback"`
	Amount      float64 `json:"amount" example:"2000.00"`
	DueDate     string  `json:"due_date" example:"2026-03-15T00:00:00Z"`
}

type UpdateMilestoneResponse struct {
	Message string `json:"message" example:"Milestone updated successfully"`
}

//delete milestone response

type DeleteMilestoneResponse struct {
	Message string `json:"message" example:"Milestone deleted successfully"`
}

//submit milestone request and response

type SubmitMilestoneRequest struct {
	Notes          string `json:"notes" example:"Completed design files, Figma link attached"`
	DeliverableURL string `json:"deliverable_url" example:"https://drive.google.com/file/d/abc123"`
}

type SubmitMilestoneResponse struct {
	Message string `json:"message" example:"Milestone submitted for review"`
}

//approve milestone request and response

type ApproveMilestoneRequest struct {
	Note string `json:"note" example:"Great work, approved!"`
}

type ApproveMilestoneResponse struct {
	Message string `json:"message" example:"Milestone approved successfully"`
}

//reject milestone request and response

type RejectMilestoneRequest struct {
	Reason string `json:"reason" huma:"required" example:"Logo colors do not match brand guidelines"`
}

type RejectMilestoneResponse struct {
	Message string `json:"message" example:"Milestone rejected"`
}

//mark milestone paid request and response

type MarkMilestonePaidRequest struct {
	PaymentReference string `json:"payment_reference" example:"TXN-2026-001234"`
}

type MarkMilestonePaidResponse struct {
	Message string `json:"message" example:"Milestone marked as paid"`
}

//milestone response

type MilestoneResponse struct {
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`

	ContractID string `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`

	Title string `json:"title" example:"Design Phase"`

	Description string `json:"description" example:"Complete UI/UX design for the homepage"`

	Amount float64 `json:"amount" example:"1500.00"`

	DueDate string `json:"due_date,omitempty" example:"2026-03-01T00:00:00Z"`

	Status string `json:"status" example:"pending"`

	CompletedAt string `json:"completed_at,omitempty" example:"2026-02-28T00:00:00Z"`

	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`

	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

//milestone list response

type MilestoneListResponse struct {
	Milestones []MilestoneResponse `json:"milestones"`
	Total      int                 `json:"total" example:"1"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreateMilestoneInput struct {
	Body CreateMilestoneRequest
}
type CreateMilestoneOutput struct {
	Body CreateMilestoneResponse
}

type UpdateMilestoneInput struct {
	ID   string `path:"id" doc:"Milestone ID"`
	Body UpdateMilestoneRequest
}
type UpdateMilestoneOutput struct {
	Body UpdateMilestoneResponse
}

type DeleteMilestoneInput struct {
	ID string `path:"id" doc:"Milestone ID"`
}
type DeleteMilestoneOutput struct {
	Body DeleteMilestoneResponse
}

type SubmitMilestoneInput struct {
	ID   string `path:"id" doc:"Milestone ID"`
	Body SubmitMilestoneRequest
}
type SubmitMilestoneOutput struct {
	Body SubmitMilestoneResponse
}

type ApproveMilestoneInput struct {
	ID   string `path:"id" doc:"Milestone ID"`
	Body ApproveMilestoneRequest
}
type ApproveMilestoneOutput struct {
	Body ApproveMilestoneResponse
}

type RejectMilestoneInput struct {
	ID   string `path:"id" doc:"Milestone ID"`
	Body RejectMilestoneRequest
}
type RejectMilestoneOutput struct {
	Body RejectMilestoneResponse
}

type MarkMilestonePaidInput struct {
	ID   string `path:"id" doc:"Milestone ID"`
	Body MarkMilestonePaidRequest
}
type MarkMilestonePaidOutput struct {
	Body MarkMilestonePaidResponse
}

type GetMilestoneByIDInput struct {
	ID string `path:"id" doc:"Milestone ID"`
}
type GetMilestoneByIDOutput struct {
	Body MilestoneResponse
}

type GetContractMilestonesInput struct {
	ContractID string `path:"contract_id" doc:"Contract ID"`
}
type GetContractMilestonesOutput struct {
	Body MilestoneListResponse
}
