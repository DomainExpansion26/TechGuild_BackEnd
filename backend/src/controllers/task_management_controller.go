package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

type TaskManagementController struct {
	service services.TaskManagementService
}

func NewTaskManagementController() *TaskManagementController {
	return &TaskManagementController{
		service: services.NewTaskManagementService(),
	}
}

func NewTaskManagementControllerWithService(svc services.TaskManagementService) *TaskManagementController {
	return &TaskManagementController{service: svc}
}

var defaultTMDController = NewTaskManagementController()

// SetTaskManagementServiceForTesting allows unit tests to inject a mock service.
func SetTaskManagementServiceForTesting(svc services.TaskManagementService) {
	defaultTMDController.service = svc
}

// 1. GetTaskManagementMetricsHandler
func GetTaskManagementMetricsHandler(ctx context.Context, input *dto.TaskManagementMetricsInput) (*dto.TaskManagementMetricsOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetMetrics(clientID, input.QuestID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.TaskManagementMetricsOutput{Body: *res}, nil
}

// 2. GetActiveQuestsHandler
func GetActiveQuestsHandler(ctx context.Context, input *dto.GetActiveQuestsInput) (*dto.ActiveQuestsListOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetActiveQuests(clientID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.ActiveQuestsListOutput{Body: *res}, nil
}

// 3. GetQuestDashboardHandler
func GetQuestDashboardHandler(ctx context.Context, input *dto.GetQuestDashboardInput) (*dto.QuestDashboardOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetQuestDashboard(clientID, input.QuestID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.QuestDashboardOutput{Body: *res}, nil
}

// 4. GetQuestMilestonesTableHandler
func GetQuestMilestonesTableHandler(ctx context.Context, input *dto.GetQuestMilestonesTableInput) (*dto.QuestMilestonesTableOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetQuestMilestones(clientID, input.QuestID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.QuestMilestonesTableOutput{Body: *res}, nil
}

// 5. AddMilestoneHandler
func AddMilestoneHandler(ctx context.Context, input *dto.AddMilestoneInput) (*dto.AddMilestoneOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.AddMilestone(clientID, input.QuestID, input.Body)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.AddMilestoneOutput{
		Status: http.StatusCreated,
		Body:   *res,
	}, nil
}

// 6. DeleteMilestoneTMDHandler
func DeleteMilestoneTMDHandler(ctx context.Context, input *dto.DeleteMilestoneTMDInput) (*dto.DeleteMilestoneTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.DeleteMilestone(clientID, input.MilestoneID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.DeleteMilestoneTMDOutput{Body: *res}, nil
}

// 7. GetMilestoneTasksHandler
func GetMilestoneTasksHandler(ctx context.Context, input *dto.GetMilestoneTasksInput) (*dto.MilestoneTasksOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetMilestoneTasks(clientID, input.MilestoneID)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.MilestoneTasksOutput{Body: *res}, nil
}

// 8. ApproveMilestoneTMDHandler
func ApproveMilestoneTMDHandler(ctx context.Context, input *dto.ApproveMilestoneTMDInput) (*dto.ApproveMilestoneTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.ApproveMilestone(clientID, input.MilestoneID, input.Body.Feedback)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.ApproveMilestoneTMDOutput{Body: *res}, nil
}

// 9. CancelMilestoneApprovalTMDHandler
func CancelMilestoneApprovalTMDHandler(ctx context.Context, input *dto.CancelMilestoneApprovalTMDInput) (*dto.CancelMilestoneApprovalTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.CancelMilestoneApproval(clientID, input.MilestoneID, input.Body.Reason)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.CancelMilestoneApprovalTMDOutput{Body: *res}, nil
}

// 10. ApproveQuestTMDHandler
func ApproveQuestTMDHandler(ctx context.Context, input *dto.ApproveQuestTMDInput) (*dto.ApproveQuestTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.ApproveQuest(clientID, input.QuestID, input.Body)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.ApproveQuestTMDOutput{Body: *res}, nil
}

// 11. CancelQuestApprovalTMDHandler
func CancelQuestApprovalTMDHandler(ctx context.Context, input *dto.CancelQuestApprovalTMDInput) (*dto.CancelQuestApprovalTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.CancelQuestApproval(clientID, input.QuestID, input.Body.Reason)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.CancelQuestApprovalTMDOutput{Body: *res}, nil
}

// 12. GetQuestActivitiesTMDHandler
func GetQuestActivitiesTMDHandler(ctx context.Context, input *dto.GetQuestActivitiesTMDInput) (*dto.QuestActivitiesTMDOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.GetQuestActivities(clientID, input.QuestID, input.Limit, input.Offset)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.QuestActivitiesTMDOutput{Body: *res}, nil
}

// 13. CreateMilestoneTaskHandler
func CreateMilestoneTaskHandler(ctx context.Context, input *dto.CreateMilestoneTaskInput) (*dto.CreateMilestoneTaskOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.CreateMilestoneTask(userID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.CreateMilestoneTaskOutput{
		Status: http.StatusCreated,
		Body:   *res,
	}, nil
}

// 14. UpdateTaskStatusHandler
func UpdateTaskStatusHandler(ctx context.Context, input *dto.UpdateTaskStatusInput) (*dto.UpdateTaskStatusOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := defaultTMDController.service.UpdateTaskStatus(userID, input.TaskID, input.Body.Status)
	if err != nil {
		return nil, mapTMDError(err)
	}

	return &dto.UpdateTaskStatusOutput{Body: *res}, nil
}

func mapTMDError(err error) error {
	if errors.Is(err, services.ErrTMDUnauthorized) {
		return huma.Error403Forbidden(err.Error())
	}
	if errors.Is(err, services.ErrTMDQuestNotFound) ||
		errors.Is(err, services.ErrTMDMilestoneNotFound) ||
		errors.Is(err, services.ErrTMDTaskNotFound) {
		return huma.Error404NotFound(err.Error())
	}
	if errors.Is(err, services.ErrTMDMilestoneAlreadyDone) ||
		errors.Is(err, services.ErrTMDAllMilestonesNotDone) ||
		errors.Is(err, services.ErrTMDInvalidMilestoneStatus) ||
		errors.Is(err, services.ErrTMDQuestNotCompleted) ||
		errors.Is(err, services.ErrTMDInvalidRating) ||
		errors.Is(err, services.ErrTMDReasonRequired) ||
		errors.Is(err, services.ErrTMDInvalidStatus) {
		return huma.Error400BadRequest(err.Error())
	}
	return huma.Error400BadRequest(err.Error())
}
