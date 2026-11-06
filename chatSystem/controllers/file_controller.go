package controllers

import (
	"context"
	"io"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
)

type FileController struct {
	fileService services.FileService
}

func NewFileController(fileService services.FileService) *FileController {
	return &FileController{
		fileService: fileService,
	}
}

// UploadFileHandler handles POST /v1/communication/channels/{id}/files via multipart form
func (c *FileController) UploadFileHandler(ctx context.Context, input *dto.UploadFileInput) (*dto.UploadFileOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	form := input.RawBody.Data()
	if form == nil {
		return nil, huma.Error400BadRequest("No file uploaded. Please provide a file in form field 'file'")
	}

	file := form.File
	if file.Filename == "" {
		return nil, huma.Error400BadRequest("No file uploaded. Please provide a file in form field 'file'")
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to read uploaded file: " + err.Error())
	}

	res, err := c.fileService.UploadFileContent(ctx, userUUID, input.ID, file.Filename, content, file.Size, file.ContentType)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == utils.ErrFileTypeNotAllowed || err == utils.ErrFileSizeExceeded || err == utils.ErrEmptyFile {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UploadFileOutput{Body: *res}, nil
}

// UploadFileDirectHandler handles POST /v1/communication/channels/{id}/files when JSON payload is provided
func (c *FileController) UploadFileDirectHandler(ctx context.Context, input *dto.UploadFileDirectInput) (*dto.UploadFileOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.fileService.UploadFileDirect(ctx, userUUID, input.ID, input.Body)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == utils.ErrFileTypeNotAllowed || err == utils.ErrFileSizeExceeded || err == utils.ErrEmptyFile {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UploadFileOutput{Body: *res}, nil
}
