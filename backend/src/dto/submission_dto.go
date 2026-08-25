package dto

// ---------- Requests ----------

type CreateSubmissionRequest struct {
	MilestoneID   string `json:"milestone_id" huma:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message       string `json:"message" example:"Work completed as per requirements"`
	SubmissionURL string `json:"submission_url" example:"https://github.com/user/repo/pull/123"`
	AttachmentURL string `json:"attachment_url" example:"https://storage.example.com/files/demo.zip"`
}

type UpdateSubmissionRequest struct {
	Message       string `json:"message" example:"Updated submission with fixes"`
	SubmissionURL string `json:"submission_url" example:"https://github.com/user/repo/pull/124"`
	AttachmentURL string `json:"attachment_url" example:"https://storage.example.com/files/demo-v2.zip"`
}

type ReviewSubmissionRequest struct {
	Status  string `json:"status" huma:"required,enum=approved;rejected" example:"approved"`
	Message string `json:"message" example:"Looks good, approved!"`
}

type ApproveSubmissionRequest struct {
	Message string `json:"message" example:"Looks good, approved!"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason" huma:"required" example:"Work does not meet the requirements"`
}

// ---------- Responses ----------

type CreateSubmissionResponse struct {
	Message      string `json:"message" example:"Submission created successfully"`
	SubmissionID string `json:"submission_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateSubmissionResponse struct {
	Message string `json:"message" example:"Submission updated successfully"`
}

type DeleteSubmissionResponse struct {
	Message string `json:"message" example:"Submission deleted successfully"`
}

type ReviewSubmissionResponse struct {
	Message string `json:"message" example:"Submission approved successfully"`
}

type SubmissionResponse struct {
	ID            string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MilestoneID   string `json:"milestone_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message       string `json:"message" example:"Work completed as per requirements"`
	SubmissionURL string `json:"submission_url" example:"https://github.com/user/repo/pull/123"`
	AttachmentURL string `json:"attachment_url" example:"https://storage.example.com/files/demo.zip"`
	Status        string `json:"status" example:"pending"`
	SubmittedAt   string `json:"submitted_at" example:"2026-01-15T00:00:00Z"`
	ReviewedAt    string `json:"reviewed_at,omitempty" example:"2026-01-20T00:00:00Z"`
	CreatedAt     string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt     string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type SubmissionListResponse struct {
	Submissions []SubmissionResponse `json:"submissions"`
	Total       int                  `json:"total" example:"1"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreateSubmissionInput struct {
	Body CreateSubmissionRequest
}
type CreateSubmissionOutput struct {
	Body CreateSubmissionResponse
}

type UpdateSubmissionInput struct {
	ID   string `path:"id" doc:"Submission ID"`
	Body UpdateSubmissionRequest
}
type UpdateSubmissionOutput struct {
	Body UpdateSubmissionResponse
}

type DeleteSubmissionInput struct {
	ID string `path:"id" doc:"Submission ID"`
}
type DeleteSubmissionOutput struct {
	Body DeleteSubmissionResponse
}

type ApproveSubmissionInput struct {
	ID   string `path:"id" doc:"Submission ID"`
	Body ApproveSubmissionRequest
}
type ApproveSubmissionOutput struct {
	Body ReviewSubmissionResponse
}

type RejectSubmissionInput struct {
	ID   string `path:"id" doc:"Submission ID"`
	Body RejectSubmissionRequest
}
type RejectSubmissionOutput struct {
	Body ReviewSubmissionResponse
}

type GetSubmissionByIDInput struct {
	ID string `path:"id" doc:"Submission ID"`
}
type GetSubmissionByIDOutput struct {
	Body SubmissionResponse
}

type GetMilestoneSubmissionsInput struct {
	MilestoneID string `path:"milestone_id" doc:"Milestone ID"`
}
type GetMilestoneSubmissionsOutput struct {
	Body SubmissionListResponse
}
