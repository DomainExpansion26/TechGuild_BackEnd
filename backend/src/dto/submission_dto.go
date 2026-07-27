package dto

// Create Submission

type CreateSubmissionRequest struct {
	MilestoneID string `json:"milestone_id" binding:"required"`

	Message string `json:"message"`

	SubmissionURL string `json:"submission_url"`

	AttachmentURL string `json:"attachment_url"`
}

type CreateSubmissionResponse struct {
	Message      string `json:"message"`
	SubmissionID string `json:"submission_id"`
}
// Review Submission


type ReviewSubmissionRequest struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`

	Message string `json:"message"`
}

type ReviewSubmissionResponse struct {
	Message string `json:"message"`
}

//submission response

type SubmissionResponse struct {
	ID string `json:"id"`

	MilestoneID string `json:"milestone_id"`

	Message string `json:"message"`

	SubmissionURL string `json:"submission_url"`

	AttachmentURL string `json:"attachment_url"`

	Status string `json:"status"`

	SubmittedAt string `json:"submitted_at"`

	ReviewedAt string `json:"reviewed_at,omitempty"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

//submission list response

type SubmissionListResponse struct {
	Submissions []SubmissionResponse `json:"submissions"`
	Total       int                  `json:"total"`
}