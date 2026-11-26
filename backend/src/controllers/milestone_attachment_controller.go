package controllers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

// UploadMilestoneAttachmentHandler accepts a multipart file (PDF or image) and
// attaches it to a milestone.
func UploadMilestoneAttachmentHandler(ctx context.Context, input *dto.UploadMilestoneAttachmentInput) (*dto.UploadMilestoneAttachmentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.File

	res, err := services.NewMilestoneAttachmentService().Upload(
		userID, input.QuestID, input.MilestoneID, file, file.Filename, file.Size,
	)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}

	out := &dto.UploadMilestoneAttachmentOutput{}
	out.Status = 201
	out.Body = *res
	return out, nil
}

func ListMilestoneAttachmentsHandler(ctx context.Context, input *dto.ListMilestoneAttachmentsInput) (*dto.ListMilestoneAttachmentsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := services.NewMilestoneAttachmentService().List(userID, input.QuestID, input.MilestoneID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.ListMilestoneAttachmentsOutput{Body: *res}, nil
}

func DeleteMilestoneAttachmentHandler(ctx context.Context, input *dto.DeleteMilestoneAttachmentInput) (*dto.DeleteMilestoneAttachmentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := services.NewMilestoneAttachmentService().Delete(userID, input.QuestID, input.MilestoneID, input.AttachmentID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.DeleteMilestoneAttachmentOutput{Body: *res}, nil
}
