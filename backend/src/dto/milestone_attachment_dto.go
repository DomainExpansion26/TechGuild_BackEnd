package dto

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// UploadMilestoneAttachmentForm is the multipart form for a deliverable file.
type UploadMilestoneAttachmentForm struct {
	File huma.FormFile `form:"file" required:"true" doc:"PDF or image up to 2 MiB"`
}

type UploadMilestoneAttachmentInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	RawBody     huma.MultipartFormFiles[UploadMilestoneAttachmentForm]
}

type UploadMilestoneAttachmentOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   MilestoneAttachmentResponse
}

type MilestoneAttachmentResponse struct {
	ID            string    `json:"id"`
	MilestoneID   string    `json:"milestone_id"`
	QuestID       string    `json:"quest_id"`
	FileName      string    `json:"file_name"`
	FileURL       string    `json:"file_url"`
	FileType      string    `json:"file_type" enum:"pdf,image"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	UploadedByID  string    `json:"uploaded_by_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type ListMilestoneAttachmentsInput struct {
	QuestID     string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
}

type ListMilestoneAttachmentsResponse struct {
	Attachments []MilestoneAttachmentResponse `json:"attachments"`
	Total       int                           `json:"total"`
}

type ListMilestoneAttachmentsOutput struct {
	Body ListMilestoneAttachmentsResponse
}

type DeleteMilestoneAttachmentInput struct {
	QuestID      string `path:"id" format:"uuid" doc:"Quest ID"`
	MilestoneID  string `path:"milestone_id" format:"uuid" doc:"Milestone ID"`
	AttachmentID string `path:"attachment_id" format:"uuid" doc:"Attachment ID"`
}

type DeleteMilestoneAttachmentResponse struct {
	Message      string `json:"message"`
	AttachmentID string `json:"attachment_id"`
}

type DeleteMilestoneAttachmentOutput struct {
	Body DeleteMilestoneAttachmentResponse
}
