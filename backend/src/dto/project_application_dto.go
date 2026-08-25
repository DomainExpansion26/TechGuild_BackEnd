package dto

// ---------- Requests ----------

type ApplyProjectRequest struct {
	CoverLetter       string  `json:"cover_letter" huma:"required" example:"I am interested in this project and have relevant experience in web development."`
	ProposedBudget    float64 `json:"proposed_budget" huma:"required" example:"1500.00"`
	Currency          string  `json:"currency" example:"USD"`
	EstimatedDuration string  `json:"estimated_duration" example:"2 weeks"`
}

// ---------- Responses ----------

type ApplyProjectResponse struct {
	Message       string `json:"message" example:"Application submitted successfully"`
	ApplicationID string `json:"application_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type WithdrawApplicationResponse struct {
	Message string `json:"message" example:"Application withdrawn successfully"`
}

type AcceptApplicationResponse struct {
	Message string `json:"message" example:"Application accepted successfully"`
}

type AcceptApplicationRequest struct {
	Message string `json:"message" example:"Great profile, let's discuss further."`
}

type RejectApplicationResponse struct {
	Message string `json:"message" example:"Application rejected"`
}

type RejectApplicationRequest struct {
	Reason string `json:"reason" huma:"required" example:"Profile does not match project requirements"`
}

type ShortlistApplicationResponse struct {
	Message string `json:"message" example:"Application shortlisted"`
}

type ShortlistApplicationRequest struct {
	Message string `json:"message" example:"You are shortlisted, we will reach out soon."`
}

type ProjectApplicationResponse struct {
	ID                string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID         string  `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicantID       string  `json:"applicant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CoverLetter       string  `json:"cover_letter" example:"I am interested in this project and have relevant experience in web development."`
	ProposedBudget    float64 `json:"proposed_budget" example:"1500.00"`
	Currency          string  `json:"currency" example:"USD"`
	EstimatedDuration string  `json:"estimated_duration" example:"2 weeks"`
	Status            string  `json:"status" example:"pending"`
	ClientMessage     string  `json:"client_message" example:"Great profile, let's discuss further."`
	AppliedAt         string  `json:"applied_at" example:"2026-01-15T00:00:00Z"`
	ReviewedAt        string  `json:"reviewed_at,omitempty" example:"2026-01-20T00:00:00Z"`
	CreatedAt         string  `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt         string  `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type ProjectApplicationListResponse struct {
	Applications []ProjectApplicationResponse `json:"applications"`
	Total        int                          `json:"total" example:"1"`
}

// ---------- Huma Input/Output wrapper structs ----------

type ApplyProjectInput struct {
	ProjectID string `path:"project_id" doc:"Project ID"`
	Body      ApplyProjectRequest
}
type ApplyProjectOutput struct {
	Body ApplyProjectResponse
}

type WithdrawApplicationInput struct {
	ID string `path:"application_id" doc:"Application ID"`
}
type WithdrawApplicationOutput struct {
	Body WithdrawApplicationResponse
}

type AcceptApplicationInput struct {
	ID   string `path:"application_id" doc:"Application ID"`
	Body AcceptApplicationRequest
}
type AcceptApplicationOutput struct {
	Body AcceptApplicationResponse
}

type RejectApplicationInput struct {
	ID   string `path:"application_id" doc:"Application ID"`
	Body RejectApplicationRequest
}
type RejectApplicationOutput struct {
	Body RejectApplicationResponse
}

type ShortlistApplicationInput struct {
	ID   string `path:"application_id" doc:"Application ID"`
	Body ShortlistApplicationRequest
}
type ShortlistApplicationOutput struct {
	Body ShortlistApplicationResponse
}

type GetMyApplicationsInput struct{}
type GetMyApplicationsOutput struct {
	Body ProjectApplicationListResponse
}

type GetProjectApplicationsInput struct {
	ProjectID string `path:"project_id" doc:"Project ID"`
}
type GetProjectApplicationsOutput struct {
	Body ProjectApplicationListResponse
}
