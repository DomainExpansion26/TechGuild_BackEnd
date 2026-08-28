package dto

// create contract request and response

type CreateContractRequest struct {
	ProjectID     string `json:"project_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID string `json:"application_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`

	ContractAmount float64 `json:"contract_amount" binding:"required" example:"5000.00"`
	Currency       string  `json:"currency" example:"USD"`

	StartDate       string `json:"start_date" example:"2026-02-01T00:00:00Z"`
	ExpectedEndDate string `json:"expected_end_date" example:"2026-04-01T00:00:00Z"`
}

type CreateContractResponse struct {
	Message    string `json:"message" example:"Contract created successfully"`
	ContractID string `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

//sign contract request and response

type SignContractRequest struct {
	Signature string `json:"signature" example:"John Doe"`
}

type SignContractResponse struct {
	Message string `json:"message" example:"Contract signed successfully"`
}

//complete contract request and response

type CompleteContractRequest struct {
	CompletionNote string `json:"completion_note" example:"All deliverables completed and approved"`
}

type CompleteContractResponse struct {
	Message string `json:"message" example:"Contract completed successfully"`
}

//cancel contract request and response

type CancelContractRequest struct {
	Reason string `json:"reason" example:"Project requirements changed"`
}

type CancelContractResponse struct {
	Message string `json:"message" example:"Contract cancelled successfully"`
}

//contract response

type ContractResponse struct {
	ID            string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID     string `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID string `json:"application_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClientID      string `json:"client_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FreelancerID  string `json:"freelancer_id" example:"550e8400-e29b-41d4-a716-446655440000"`

	ContractAmount float64 `json:"contract_amount" example:"5000.00"`
	Currency       string  `json:"currency" example:"USD"`

	Status string `json:"status" example:"active"`

	SignedByClient     bool `json:"signed_by_client" example:"true"`
	SignedByFreelancer bool `json:"signed_by_freelancer" example:"true"`

	StartDate       string `json:"start_date,omitempty" example:"2026-02-01T00:00:00Z"`
	ExpectedEndDate string `json:"expected_end_date,omitempty" example:"2026-04-01T00:00:00Z"`
	CompletedAt     string `json:"completed_at,omitempty" example:"2026-03-30T00:00:00Z"`

	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

//contract list response

type ContractListResponse struct {
	Contracts []ContractResponse `json:"contracts"`
	Total     int                `json:"total" example:"1"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreateContractInput struct {
	Body CreateContractRequest
}
type CreateContractOutput struct {
	Body CreateContractResponse
}

type SignContractInput struct {
	ID   string `path:"id" doc:"Contract ID"`
	Body SignContractRequest
}
type SignContractOutput struct {
	Body SignContractResponse
}

type CompleteContractInput struct {
	ID   string `path:"id" doc:"Contract ID"`
	Body CompleteContractRequest
}
type CompleteContractOutput struct {
	Body CompleteContractResponse
}

type CancelContractInput struct {
	ID   string `path:"id" doc:"Contract ID"`
	Body CancelContractRequest
}
type CancelContractOutput struct {
	Body CancelContractResponse
}

type GetContractByIDInput struct {
	ID string `path:"id" doc:"Contract ID"`
}
type GetContractByIDOutput struct {
	Body ContractResponse
}

type GetClientContractsInput struct{}
type GetClientContractsOutput struct {
	Body ContractListResponse
}

type GetFreelancerContractsInput struct{}
type GetFreelancerContractsOutput struct {
	Body ContractListResponse
}
