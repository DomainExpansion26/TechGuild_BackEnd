package dto

import (
	"github.com/danielgtaylor/huma/v2"
)

// UploadFileForm represents a multipart form file upload
type UploadFileForm struct {
	File huma.FormFile `form:"file" required:"true" doc:"The file to upload (allowed: .pdf, .xml, .doc, .docx, .md)"`
}

type UploadFileInput struct {
	ID      string                                `path:"id" doc:"Channel ID" required:"true"`
	RawBody huma.MultipartFormFiles[UploadFileForm]
}

// UploadFileDirectRequest provides JSON payload alternative for presigned/direct upload flow
type UploadFileDirectRequest struct {
	FileName    string `json:"file_name" required:"true" doc:"Name of the file including extension (.pdf, .xml, .doc, .docx, .md)" example:"specification.pdf"`
	FileSize    int64  `json:"file_size" required:"true" doc:"Size of the file in bytes" example:"1048576"`
	ContentType string `json:"content_type,omitempty" doc:"MIME content type" example:"application/pdf"`
	FileURL     string `json:"file_url,omitempty" doc:"Direct URL if client pre-uploaded to R2 via presigned URL"`
}

type UploadFileDirectInput struct {
	ID   string                  `path:"id" doc:"Channel ID" required:"true"`
	Body UploadFileDirectRequest `required:"true"`
}

// UploadFileResponse returned when a file is uploaded to the channel
type UploadFileResponse struct {
	MessageID   string          `json:"message_id"`
	FileName    string          `json:"file_name"`
	FileURL     string          `json:"file_url"`
	FileSize    int64           `json:"file_size"`
	ContentType string          `json:"content_type"`
	Message     MessageResponse `json:"message"`
}

type UploadFileOutput struct {
	Body UploadFileResponse
}

// PresignedURLRequest for requesting an R2 presigned upload URL
type PresignedURLRequest struct {
	FileName    string `json:"file_name" required:"true" example:"document.pdf"`
	FileSize    int64  `json:"file_size" required:"true" example:"204800"`
	ContentType string `json:"content_type,omitempty" example:"application/pdf"`
}

type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	ExpiresIn int    `json:"expires_in_seconds" example:"900"`
}

