package dto

//apply project request and response

type ApplyProjectRequest struct {
	CoverLetter      string  `json:"cover_letter" binding:"required"`
	ProposedBudget   float64 `json:"proposed_budget" binding:"required"`
	Currency         string  `json:"currency"`
	EstimatedDuration string `json:"estimated_duration"`
}

type ApplyProjectResponse struct {
	Message       string `json:"message"`
	ApplicationID string `json:"application_id"`
}

//withdraw application response


type WithdrawApplicationResponse struct {
	Message string `json:"message"`
}

//accept application response


type AcceptApplicationResponse struct {
	Message string `json:"message"`
}

//reject application response

type RejectApplicationResponse struct {
	Message string `json:"message"`
}

//shortlist application response

type ShortlistApplicationResponse struct {
	Message string `json:"message"`
}

//project application response

type ProjectApplicationResponse struct {
	ID string `json:"id"`

	ProjectID string `json:"project_id"`

	ApplicantID string `json:"applicant_id"`

	CoverLetter string `json:"cover_letter"`

	ProposedBudget float64 `json:"proposed_budget"`

	Currency string `json:"currency"`

	EstimatedDuration string `json:"estimated_duration"`

	Status string `json:"status"`

	ClientMessage string `json:"client_message"`

	AppliedAt string `json:"applied_at"`

	ReviewedAt string `json:"reviewed_at,omitempty"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

//list of project applications response

type ProjectApplicationListResponse struct {
	Applications []ProjectApplicationResponse `json:"applications"`
	Total        int                          `json:"total"`
}
