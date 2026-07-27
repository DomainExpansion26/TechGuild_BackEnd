package dto

// create contract request and response

type CreateContractRequest struct {
	ProjectID     string  `json:"project_id" binding:"required"`
	ApplicationID string  `json:"application_id" binding:"required"`

	ContractAmount float64 `json:"contract_amount" binding:"required"`
	Currency       string  `json:"currency"`

	StartDate       string `json:"start_date"`
	ExpectedEndDate string `json:"expected_end_date"`
}

type CreateContractResponse struct {
	Message    string `json:"message"`
	ContractID string `json:"contract_id"`
}

//sign contract response

type SignContractResponse struct {
	Message string `json:"message"`
}

//cancel contract response

type CancelContractResponse struct {
	Message string `json:"message"`
}

//complete contract response

type CompleteContractResponse struct {
	Message string `json:"message"`
}

//contract response

type ContractResponse struct {
	ID string `json:"id"`

	ProjectID string `json:"project_id"`

	ApplicationID string `json:"application_id"`

	ClientID string `json:"client_id"`

	FreelancerID string `json:"freelancer_id"`

	ContractAmount float64 `json:"contract_amount"`

	Currency string `json:"currency"`

	StartDate string `json:"start_date,omitempty"`

	ExpectedEndDate string `json:"expected_end_date,omitempty"`

	Status string `json:"status"`

	SignedByClient bool `json:"signed_by_client"`

	SignedByFreelancer bool `json:"signed_by_freelancer"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

//contract list response

type ContractListResponse struct {
	Contracts []ContractResponse `json:"contracts"`
	Total     int                `json:"total"`
}