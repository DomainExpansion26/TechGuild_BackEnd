package dto

// create milestone request and response

type CreateMilestoneRequest struct {
	ContractID  string  `json:"contract_id" binding:"required"`
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description"`
	Amount        float64 `json:"amount" binding:"required"`
	DueDate       string  `json:"due_date"`
}

type CreateMilestoneResponse struct {
	Message     string `json:"message"`
	MilestoneID string `json:"milestone_id"`
}

//update milestone request and response

type UpdateMilestoneRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	DueDate     string  `json:"due_date"`
}

type UpdateMilestoneResponse struct {
	Message string `json:"message"`
}

//delete milestone response

type DeleteMilestoneResponse struct {
	Message string `json:"message"`
}

//submit milestone response

type SubmitMilestoneResponse struct {
	Message string `json:"message"`
}

//approve milestone response

type ApproveMilestoneResponse struct {
	Message string `json:"message"`
}

//reject milestone response

type RejectMilestoneResponse struct {
	Message string `json:"message"`
}

//milestone response

type MilestoneResponse struct {
	ID string `json:"id"`

	ContractID string `json:"contract_id"`

	Title string `json:"title"`

	Description string `json:"description"`

	Amount float64 `json:"amount"`

	DueDate string `json:"due_date,omitempty"`

	Status string `json:"status"`

	CompletedAt string `json:"completed_at,omitempty"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

//milestone list response

type MilestoneListResponse struct {
	Milestones []MilestoneResponse `json:"milestones"`
	Total      int                 `json:"total"`
}