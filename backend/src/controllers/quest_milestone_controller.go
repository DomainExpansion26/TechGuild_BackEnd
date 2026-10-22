package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/repository"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

type QuestMilestoneController struct {
	svc services.QuestMilestoneService
}

func NewQuestMilestoneController(cfg *config.Config) *QuestMilestoneController {
	return &QuestMilestoneController{
		svc: services.NewQuestMilestoneService(cfg),
	}
}

// ============================================================
// 1. CREATE MILESTONE (freelancer)
// ============================================================

func (c *QuestMilestoneController) CreateMilestoneHandler(
	ctx context.Context,
	input *dto.CreateMilestoneInput,
) (*dto.CreateQuestMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.CreateMilestone(userID, input.QuestID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.CreateQuestMilestoneOutput{
		Status: 201,
		Body:   *resp,
	}, nil
}

// ============================================================
// 2. UPDATE MILESTONE (freelancer)
// ============================================================

func (c *QuestMilestoneController) UpdateMilestoneHandler(
	ctx context.Context,
	input *dto.UpdateQuestMilestoneInput,
) (*dto.UpdateQuestMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.UpdateMilestone(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.UpdateQuestMilestoneOutput{Body: *resp}, nil
}

// ============================================================
// 3. DELETE MILESTONE (freelancer)
// ============================================================

func (c *QuestMilestoneController) DeleteMilestoneHandler(
	ctx context.Context,
	input *dto.DeleteQuestMilestoneInput,
) (*dto.DeleteQuestMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.svc.DeleteMilestone(userID, input.QuestID, input.MilestoneID); err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.DeleteQuestMilestoneOutput{
		Body: dto.DeleteQuestMilestoneResponse{
			Message:     "Milestone deleted successfully",
			MilestoneID: input.MilestoneID,
		},
	}, nil
}

// ============================================================
// 4. SEND FOR REVIEW (freelancer)
// ============================================================

func (c *QuestMilestoneController) SendForReviewHandler(
	ctx context.Context,
	input *dto.SendForReviewInput,
) (*dto.SendForReviewOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.SendForReview(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.SendForReviewOutput{Body: *resp}, nil
}

// ============================================================
// 5. RESUBMIT AFTER REVISION (freelancer)
// ============================================================

func (c *QuestMilestoneController) ResubmitHandler(
	ctx context.Context,
	input *dto.SendForReviewInput,
) (*dto.SendForReviewOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ResubmitAfterRevision(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.SendForReviewOutput{Body: *resp}, nil
}

// ============================================================
// 6. APPROVE MILESTONE (client)
// ============================================================

func (c *QuestMilestoneController) ApproveMilestoneHandler(
	ctx context.Context,
	input *dto.ApproveMilestoneInput,
) (*dto.ApproveMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ApproveMilestone(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.ApproveMilestoneOutput{Body: *resp}, nil
}

// ============================================================
// 7. REQUEST REVISION (client)
// ============================================================

func (c *QuestMilestoneController) RequestRevisionHandler(
	ctx context.Context,
	input *dto.RequestMilestoneRevisionInput,
) (*dto.RequestMilestoneRevisionOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.RequestRevision(userID, input.QuestID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.RequestMilestoneRevisionOutput{Body: *resp}, nil
}

// ============================================================
// 8. GET MILESTONE (shared)
// ============================================================

func (c *QuestMilestoneController) GetMilestoneHandler(
	ctx context.Context,
	input *dto.GetQuestMilestoneInput,
) (*dto.GetQuestMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.GetMilestone(userID, input.QuestID, input.MilestoneID)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.GetQuestMilestoneOutput{Body: *resp}, nil
}

// ============================================================
// 9. LIST MILESTONES (shared)
// ============================================================

func (c *QuestMilestoneController) ListMilestonesHandler(
	ctx context.Context,
	input *dto.ListQuestMilestonesInput,
) (*dto.ListQuestMilestonesOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.ListMilestones(userID, input.QuestID)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.ListQuestMilestonesOutput{Body: *resp}, nil
}

// ============================================================
// 10. GET REVISIONS (shared)
// ============================================================

func (c *QuestMilestoneController) GetRevisionsHandler(
	ctx context.Context,
	input *dto.GetMilestoneRevisionsInput,
) (*dto.GetMilestoneRevisionsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	resp, err := c.svc.GetRevisions(userID, input.QuestID, input.MilestoneID)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.GetMilestoneRevisionsOutput{Body: *resp}, nil
}

// ============================================================
// Error mapper
// ============================================================

func mapQuestMilestoneError(err error) error {
	switch {
	case errors.Is(err, services.ErrQMSQuestNotFound),
		errors.Is(err, services.ErrQMSMilestoneNotFound),
		errors.Is(err, repository.ErrMilestoneNotFound):
		return huma.Error404NotFound(err.Error())

	case errors.Is(err, services.ErrQMSUnauthorized),
		errors.Is(err, services.ErrQMSNotAssignedFreelancer),
		errors.Is(err, services.ErrQMSNotQuestClient):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, services.ErrQMSInvalidState),
		errors.Is(err, services.ErrQMSCannotDelete),
		errors.Is(err, services.ErrQMSCannotEdit),
		errors.Is(err, services.ErrQMSQuestNotInProgress),
		errors.Is(err, services.ErrQMSAmountRequired),
		errors.Is(err, services.ErrQMSTitleRequired),
		errors.Is(err, services.ErrQMSRevisionNoteRequired),
		errors.Is(err, services.ErrInvalidUserID):
		return huma.Error400BadRequest(err.Error())

	default:
		return huma.Error500InternalServerError(err.Error())
	}
}

// ============================================================
// 11. SUBMIT ROADMAP (freelancer)
// ============================================================
func (c *QuestMilestoneController) SubmitRoadmapHandler(
	ctx context.Context,
	input *dto.SubmitRoadmapInput,
) (*dto.SubmitRoadmapOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	count, err := c.svc.SubmitRoadmap(userID, input.ID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.SubmitRoadmapOutput{
		Body: dto.RoadmapActionResponse{
			Message:                "Roadmap submitted for review successfully",
			UpdatedMilestonesCount: int(count),
		},
	}, nil
}

// ============================================================
// 12. APPROVE ROADMAP (client)
// ============================================================
func (c *QuestMilestoneController) ApproveRoadmapHandler(
	ctx context.Context,
	input *dto.ApproveRoadmapInput,
) (*dto.ApproveRoadmapOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	count, err := c.svc.ApproveRoadmap(userID, input.ID, input.Body)
	if err != nil {
		return nil, mapQuestMilestoneError(err)
	}

	return &dto.ApproveRoadmapOutput{
		Body: dto.RoadmapActionResponse{
			Message:                "Roadmap approved successfully",
			UpdatedMilestonesCount: int(count),
		},
	}, nil
}
