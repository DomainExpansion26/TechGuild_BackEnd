package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

var (
	ErrTMDUnauthorized           = errors.New("unauthorized access to quest")
	ErrTMDQuestNotFound          = errors.New("quest not found")
	ErrTMDMilestoneNotFound      = errors.New("milestone not found")
	ErrTMDTaskNotFound           = errors.New("task not found")
	ErrTMDMilestoneAlreadyDone   = errors.New("cannot delete an already approved milestone")
	ErrTMDAllMilestonesNotDone   = errors.New("all milestones must be approved before approving the quest")
	ErrTMDInvalidMilestoneStatus = errors.New("milestone is not in an approved state to cancel")
	ErrTMDQuestNotCompleted      = errors.New("quest is not completed; cannot cancel approval")
	ErrTMDInvalidRating          = errors.New("rating must be between 1 and 5")
	ErrTMDReasonRequired         = errors.New("reason is mandatory and must be at least 5 characters")
	ErrTMDInvalidStatus          = errors.New("invalid task status; allowed values: todo, in_progress, completed")
)

type TaskManagementService interface {
	GetMetrics(clientIDStr, questIDStr string) (*dto.TaskManagementMetricsResponse, error)
	GetActiveQuests(clientIDStr string) (*dto.ActiveQuestsListResponse, error)
	GetQuestDashboard(clientIDStr, questIDStr string) (*dto.QuestDashboardResponse, error)
	GetQuestMilestones(clientIDStr, questIDStr string) (*dto.QuestMilestonesTableResponse, error)
	AddMilestone(clientIDStr, questIDStr string, req dto.AddMilestoneRequestBody) (*dto.AddMilestoneResponse, error)
	DeleteMilestone(clientIDStr, milestoneIDStr string) (*dto.DeleteMilestoneTMDResponse, error)
	GetMilestoneTasks(clientIDStr, milestoneIDStr string) (*dto.MilestoneTasksResponse, error)
	ApproveMilestone(clientIDStr, milestoneIDStr string, feedback string) (*dto.ApproveMilestoneTMDResponse, error)
	CancelMilestoneApproval(clientIDStr, milestoneIDStr string, reason string) (*dto.CancelMilestoneApprovalTMDResponse, error)
	ApproveQuest(clientIDStr, questIDStr string, req dto.ApproveQuestTMDRequestBody) (*dto.ApproveQuestTMDResponse, error)
	CancelQuestApproval(clientIDStr, questIDStr string, reason string) (*dto.CancelQuestApprovalTMDResponse, error)
	GetQuestActivities(clientIDStr, questIDStr string, limit, offset int) (*dto.QuestActivitiesTMDResponse, error)
	CreateMilestoneTask(clientIDStr, milestoneIDStr string, req dto.CreateMilestoneTaskRequestBody) (*dto.MilestoneTaskItem, error)
	UpdateTaskStatus(clientIDStr, taskIDStr, statusStr string) (*dto.MilestoneTaskItem, error)
}

type taskManagementService struct {
	repo repository.TaskManagementRepository
}

func NewTaskManagementService() TaskManagementService {
	return &taskManagementService{repo: repository.NewTaskManagementRepository()}
}

func NewTaskManagementServiceWithRepo(repo repository.TaskManagementRepository) TaskManagementService {
	return &taskManagementService{repo: repo}
}

// 1. GetMetrics
func (s *taskManagementService) GetMetrics(clientIDStr, questIDStr string) (*dto.TaskManagementMetricsResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	var questUUID *uuid.UUID
	if strings.TrimSpace(questIDStr) != "" {
		parsed, err := uuid.Parse(questIDStr)
		if err == nil {
			questUUID = &parsed
		}
	}

	res, err := s.repo.GetMetrics(clientUUID, questUUID)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 2. GetActiveQuests (Switcher / Multi-Quest Banner)
func (s *taskManagementService) GetActiveQuests(clientIDStr string) (*dto.ActiveQuestsListResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	quests, err := s.repo.GetClientActiveQuests(clientUUID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ActiveQuestSummaryItem, 0, len(quests))
	for _, q := range quests {
		progress := calculateQuestProgress(q.Milestones)

		var freelancer *dto.FreelancerSummary
		if q.AssignedFreelancer != nil {
			freelancer = &dto.FreelancerSummary{
				ID:   q.AssignedFreelancer.ID.String(),
				Name: q.AssignedFreelancer.FirstName + " " + q.AssignedFreelancer.LastName,
				Role: "Freelancer",
			}
		}

		items = append(items, dto.ActiveQuestSummaryItem{
			ID:         q.ID.String(),
			Title:      q.Title,
			Category:   q.Category,
			Budget:     q.BudgetINR,
			Currency:   "INR",
			Freelancer: freelancer,
			Progress:   progress,
			Status:     string(q.Status),
		})
	}

	return &dto.ActiveQuestsListResponse{
		TotalActive: len(items),
		Quests:      items,
	}, nil
}

// 3. GetQuestDashboard
func (s *taskManagementService) GetQuestDashboard(clientIDStr, questIDStr string) (*dto.QuestDashboardResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	// Calculate stats
	totalMilestones := len(quest.Milestones)
	completedMilestones := 0
	inProgressMilestones := 0
	pendingMilestones := 0
	hasTasksCreated := false

	for _, m := range quest.Milestones {
		if len(m.Tasks) > 0 {
			hasTasksCreated = true
		}
		switch m.Status {
		case models.MilestoneApproved:
			completedMilestones++
		case models.MilestoneInProgress, models.MilestoneSubmitted:
			inProgressMilestones++
		default:
			pendingMilestones++
		}
	}

	completionPercentage := 0
	if totalMilestones > 0 {
		completionPercentage = int(math.Round((float64(completedMilestones) / float64(totalMilestones)) * 100))
	}

	canApproveQuest := totalMilestones > 0 &&
		completedMilestones == totalMilestones &&
		quest.Status == models.QuestStatusInProgress

	currentStep := 1
	if hasTasksCreated {
		currentStep = 2
	}
	if completionPercentage > 0 {
		currentStep = 3
	}
	if quest.Status == models.QuestStatusCompleted {
		currentStep = 4
	}

	var freelancer *dto.FreelancerSummary
	if quest.AssignedFreelancer != nil {
		freelancer = &dto.FreelancerSummary{
			ID:   quest.AssignedFreelancer.ID.String(),
			Name: quest.AssignedFreelancer.FirstName + " " + quest.AssignedFreelancer.LastName,
			Role: "Freelancer",
		}
	}

	// Recent activities preview (up to 4 items)
	activities, _ := s.repo.GetRecentActivities(questUUID, 4)
	activityItems := make([]dto.RecentActivityItem, 0, len(activities))
	for _, a := range activities {
		activityItems = append(activityItems, dto.RecentActivityItem{
			ID:          a.ID.String(),
			ActionType:  string(a.ActionType),
			Title:       a.Title,
			Description: a.Description,
			CreatedAt:   a.CreatedAt,
		})
	}

	return &dto.QuestDashboardResponse{
		Quest: dto.QuestDashboardOverview{
			ID:              quest.ID.String(),
			Title:           quest.Title,
			Category:        quest.Category,
			Budget:          quest.BudgetINR,
			Currency:        "INR",
			Status:          string(quest.Status),
			Freelancer:      freelancer,
			CurrentStep:     currentStep,
			HasTasksCreated: hasTasksCreated,
			CanApproveQuest: canApproveQuest,
			OverallProgress: completionPercentage,
		},
		MilestonesOverview: dto.MilestonesOverviewStats{
			CompletionPercentage: completionPercentage,
			TotalMilestones:      totalMilestones,
			CompletedMilestones:  completedMilestones,
			InProgressMilestones: inProgressMilestones,
			PendingMilestones:    pendingMilestones,
		},
		RecentActivitiesPreview: activityItems,
	}, nil
}

// 4. GetQuestMilestones
func (s *taskManagementService) GetQuestMilestones(clientIDStr, questIDStr string) (*dto.QuestMilestonesTableResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	rows := make([]dto.QuestMilestoneRowItem, 0, len(quest.Milestones))
	var totalAmount float64

	for i, m := range quest.Milestones {
		totalAmount += m.Amount
		progress := calculateMilestoneProgress(m)

		var startDateStr *string
		if m.StartDate != nil {
			str := m.StartDate.Format("2006-01-02")
			startDateStr = &str
		}

		var dueDateStr *string
		if m.DueDate != nil {
			str := m.DueDate.Format("2006-01-02")
			dueDateStr = &str
		}

		rows = append(rows, dto.QuestMilestoneRowItem{
			SlNo:        i + 1,
			ID:          m.ID.String(),
			Name:        m.Title,
			Description: m.Description,
			TaskCount:   len(m.Tasks),
			Progress:    progress,
			StartDate:   startDateStr,
			DueDate:     dueDateStr,
			DueDays:     m.DueDays,
			Amount:      m.Amount,
			Currency:    "INR",
			Status:      string(m.Status),
			OrderIndex:  m.OrderIndex,
		})
	}

	return &dto.QuestMilestonesTableResponse{
		Milestones:  rows,
		TotalAmount: totalAmount,
		Currency:    "INR",
	}, nil
}

// 5. AddMilestone
func (s *taskManagementService) AddMilestone(clientIDStr, questIDStr string, req dto.AddMilestoneRequestBody) (*dto.AddMilestoneResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 2 {
		return nil, errors.New("milestone title must be at least 2 characters")
	}

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	dueDays := req.DueDays
	if dueDays <= 0 {
		if req.StartDate != nil && req.DueDate != nil && req.DueDate.After(*req.StartDate) {
			dueDays = int(math.Ceil(req.DueDate.Sub(*req.StartDate).Hours() / 24))
		} else {
			dueDays = 7
		}
	}

	milestone := &models.QuestMilestone{
		QuestID:     quest.ID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Amount:      req.Amount,
		DueDays:     dueDays,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
		OrderIndex:  len(quest.Milestones) + 1,
		Status:      models.MilestonePending,
	}

	if err := s.repo.CreateMilestone(quest, milestone); err != nil {
		return nil, fmt.Errorf("failed to create milestone: %w", err)
	}

	// Log activity
	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityMilestoneCreated,
		Title:       fmt.Sprintf("Milestone '%s' Added", milestone.Title),
		Description: fmt.Sprintf("Client added new milestone for ₹%.2f", milestone.Amount),
	})

	var startStr *string
	if milestone.StartDate != nil {
		s := milestone.StartDate.Format("2006-01-02")
		startStr = &s
	}
	var dueStr *string
	if milestone.DueDate != nil {
		s := milestone.DueDate.Format("2006-01-02")
		dueStr = &s
	}

	newBudget := quest.BudgetINR

	return &dto.AddMilestoneResponse{
		Message: "Milestone added successfully",
		Milestone: dto.QuestMilestoneRowItem{
			SlNo:        milestone.OrderIndex,
			ID:          milestone.ID.String(),
			Name:        milestone.Title,
			Description: milestone.Description,
			TaskCount:   0,
			Progress:    0,
			StartDate:   startStr,
			DueDate:     dueStr,
			DueDays:     milestone.DueDays,
			Amount:      milestone.Amount,
			Currency:    "INR",
			Status:      string(milestone.Status),
			OrderIndex:  milestone.OrderIndex,
		},
		NewTotalBudget: newBudget,
	}, nil
}

// 6. DeleteMilestone
func (s *taskManagementService) DeleteMilestone(clientIDStr, milestoneIDStr string) (*dto.DeleteMilestoneTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrTMDMilestoneNotFound
	}

	milestone, quest, err := s.repo.GetMilestoneWithQuest(milestoneUUID)
	if err != nil || milestone == nil || quest == nil {
		return nil, ErrTMDMilestoneNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, ErrTMDUnauthorized
	}

	if milestone.Status == models.MilestoneApproved {
		return nil, ErrTMDMilestoneAlreadyDone
	}

	if err := s.repo.DeleteMilestone(milestone.ID, quest.ID, milestone.Amount); err != nil {
		return nil, fmt.Errorf("failed to delete milestone: %w", err)
	}

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:    quest.ID,
		ActorID:    clientUUID,
		ActionType: models.ActivityMilestoneDeleted,
		Title:      fmt.Sprintf("Milestone '%s' Deleted", milestone.Title),
		Description: fmt.Sprintf("Client removed milestone and its tasks"),
	})

	return &dto.DeleteMilestoneTMDResponse{
		Message:     "Milestone and associated tasks deleted successfully",
		MilestoneID: milestone.ID.String(),
	}, nil
}

// 7. GetMilestoneTasks
func (s *taskManagementService) GetMilestoneTasks(clientIDStr, milestoneIDStr string) (*dto.MilestoneTasksResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrTMDMilestoneNotFound
	}

	milestone, quest, err := s.repo.GetMilestoneWithQuest(milestoneUUID)
	if err != nil || milestone == nil || quest == nil {
		return nil, ErrTMDMilestoneNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, ErrTMDUnauthorized
	}

	mWithTasks, err := s.repo.GetMilestoneByID(milestoneUUID)
	if err != nil {
		return nil, err
	}

	taskItems := make([]dto.MilestoneTaskItem, 0, len(mWithTasks.Tasks))
	for i, t := range mWithTasks.Tasks {
		assigneeName := "Unassigned"
		assigneeAvatar := ""
		if t.Assignee != nil {
			assigneeName = t.Assignee.FirstName + " " + t.Assignee.LastName
			if assigneeName == " " {
				assigneeName = t.Assignee.Email
			}
		}

		var startStr *string
		if t.StartDate != nil {
			s := t.StartDate.Format("2006-01-02")
			startStr = &s
		}
		var endStr *string
		if t.EndDate != nil {
			s := t.EndDate.Format("2006-01-02")
			endStr = &s
		}

		taskItems = append(taskItems, dto.MilestoneTaskItem{
			SlNo:           i + 1,
			ID:             t.ID.String(),
			Name:           t.Title,
			Description:    t.Description,
			Assignee:       assigneeName,
			AssigneeAvatar: assigneeAvatar,
			StartDate:      startStr,
			EndDate:        endStr,
			Status:         string(t.Status),
		})
	}

	return &dto.MilestoneTasksResponse{
		MilestoneID:   milestone.ID.String(),
		MilestoneName: milestone.Title,
		Tasks:         taskItems,
	}, nil
}

// 8. ApproveMilestone
func (s *taskManagementService) ApproveMilestone(clientIDStr, milestoneIDStr string, feedback string) (*dto.ApproveMilestoneTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrTMDMilestoneNotFound
	}

	milestone, quest, err := s.repo.GetMilestoneWithQuest(milestoneUUID)
	if err != nil || milestone == nil || quest == nil {
		return nil, ErrTMDMilestoneNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, ErrTMDUnauthorized
	}

	now := time.Now()
	if err := s.repo.UpdateMilestoneStatus(milestone.ID, models.MilestoneApproved, &now); err != nil {
		return nil, fmt.Errorf("failed to approve milestone: %w", err)
	}

	// Log approval
	_ = s.repo.LogMilestoneApproval(&models.MilestoneApprovalLog{
		MilestoneID: milestone.ID,
		ClientID:    clientUUID,
		Action:      "approve",
		Reason:      feedback,
	})

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityMilestoneApproved,
		Title:       fmt.Sprintf("Milestone '%s' Approved", milestone.Title),
		Description: fmt.Sprintf("Client approved milestone deliverables. %s", feedback),
	})

	// Recalculate quest completion
	updatedQuest, _ := s.repo.GetQuestByIDAndClient(quest.ID, clientUUID)
	progress := 0
	canApprove := false
	if updatedQuest != nil {
		progress = calculateQuestProgress(updatedQuest.Milestones)
		allApproved := true
		for _, m := range updatedQuest.Milestones {
			if m.Status != models.MilestoneApproved {
				allApproved = false
				break
			}
		}
		canApprove = len(updatedQuest.Milestones) > 0 && allApproved && updatedQuest.Status == models.QuestStatusInProgress
	}

	return &dto.ApproveMilestoneTMDResponse{
		Message:              "Milestone approved successfully",
		MilestoneID:          milestone.ID.String(),
		Status:               string(models.MilestoneApproved),
		OverallQuestProgress: progress,
		CanApproveQuest:      canApprove,
	}, nil
}

// 9. CancelMilestoneApproval
func (s *taskManagementService) CancelMilestoneApproval(clientIDStr, milestoneIDStr string, reason string) (*dto.CancelMilestoneApprovalTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	reason = strings.TrimSpace(reason)
	if len(reason) < 5 {
		return nil, ErrTMDReasonRequired
	}

	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrTMDMilestoneNotFound
	}

	milestone, quest, err := s.repo.GetMilestoneWithQuest(milestoneUUID)
	if err != nil || milestone == nil || quest == nil {
		return nil, ErrTMDMilestoneNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, ErrTMDUnauthorized
	}

	if milestone.Status != models.MilestoneApproved {
		return nil, ErrTMDInvalidMilestoneStatus
	}

	if err := s.repo.UpdateMilestoneStatus(milestone.ID, models.MilestoneInProgress, nil); err != nil {
		return nil, fmt.Errorf("failed to cancel approval: %w", err)
	}

	_ = s.repo.LogMilestoneApproval(&models.MilestoneApprovalLog{
		MilestoneID: milestone.ID,
		ClientID:    clientUUID,
		Action:      "cancel",
		Reason:      reason,
	})

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityMilestoneApprovalCancelled,
		Title:       fmt.Sprintf("Approval Cancelled for '%s'", milestone.Title),
		Description: fmt.Sprintf("Reason: %s", reason),
	})

	updatedQuest, _ := s.repo.GetQuestByIDAndClient(quest.ID, clientUUID)
	progress := 0
	if updatedQuest != nil {
		progress = calculateQuestProgress(updatedQuest.Milestones)
	}

	return &dto.CancelMilestoneApprovalTMDResponse{
		Message:              "Milestone approval cancelled successfully",
		MilestoneID:          milestone.ID.String(),
		Status:               string(models.MilestoneInProgress),
		OverallQuestProgress: progress,
		CanApproveQuest:      false,
	}, nil
}

// 10. ApproveQuest
func (s *taskManagementService) ApproveQuest(clientIDStr, questIDStr string, req dto.ApproveQuestTMDRequestBody) (*dto.ApproveQuestTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	if req.Rating < 1 || req.Rating > 5 {
		return nil, ErrTMDInvalidRating
	}

	feedback := strings.TrimSpace(req.Feedback)
	if len(feedback) < 5 {
		return nil, errors.New("feedback must be at least 5 characters")
	}

	if len(quest.Milestones) == 0 {
		return nil, errors.New("cannot approve a quest with zero milestones")
	}

	for _, m := range quest.Milestones {
		if m.Status != models.MilestoneApproved {
			return nil, ErrTMDAllMilestonesNotDone
		}
	}

	now := time.Now()
	if err := s.repo.UpdateQuestStatus(quest.ID, models.QuestStatusCompleted); err != nil {
		return nil, fmt.Errorf("failed to complete quest: %w", err)
	}

	_ = s.repo.CreateQuestReview(&models.QuestReview{
		QuestID:  quest.ID,
		ClientID: clientUUID,
		Rating:   req.Rating,
		Feedback: feedback,
	})

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityQuestApproved,
		Title:       fmt.Sprintf("Quest '%s' Completed & Approved", quest.Title),
		Description: fmt.Sprintf("Client gave %d stars: %s", req.Rating, feedback),
	})

	return &dto.ApproveQuestTMDResponse{
		Message:     "Quest approved and marked as completed",
		QuestID:     quest.ID.String(),
		Status:      string(models.QuestStatusCompleted),
		CompletedAt: now,
	}, nil
}

// 11. CancelQuestApproval
func (s *taskManagementService) CancelQuestApproval(clientIDStr, questIDStr string, reason string) (*dto.CancelQuestApprovalTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	reason = strings.TrimSpace(reason)
	if len(reason) < 5 {
		return nil, ErrTMDReasonRequired
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	if quest.Status != models.QuestStatusCompleted {
		return nil, ErrTMDQuestNotCompleted
	}

	if err := s.repo.UpdateQuestStatus(quest.ID, models.QuestStatusInProgress); err != nil {
		return nil, fmt.Errorf("failed to revert quest status: %w", err)
	}

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityQuestApprovalCancelled,
		Title:       fmt.Sprintf("Quest Approval Cancelled for '%s'", quest.Title),
		Description: fmt.Sprintf("Reason: %s", reason),
	})

	return &dto.CancelQuestApprovalTMDResponse{
		Message: "Quest approval cancelled and reopened",
		QuestID: quest.ID.String(),
		Status:  string(models.QuestStatusInProgress),
	}, nil
}

// 12. GetQuestActivities
func (s *taskManagementService) GetQuestActivities(clientIDStr, questIDStr string, limit, offset int) (*dto.QuestActivitiesTMDResponse, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrTMDQuestNotFound
	}

	quest, err := s.repo.GetQuestByIDAndClient(questUUID, clientUUID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	activities, total, err := s.repo.GetActivitiesByQuestID(questUUID, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]dto.RecentActivityItem, 0, len(activities))
	for _, a := range activities {
		items = append(items, dto.RecentActivityItem{
			ID:          a.ID.String(),
			ActionType:  string(a.ActionType),
			Title:       a.Title,
			Description: a.Description,
			CreatedAt:   a.CreatedAt,
		})
	}

	return &dto.QuestActivitiesTMDResponse{
		QuestID:    quest.ID.String(),
		Total:      total,
		Activities: items,
	}, nil
}

// 13. CreateMilestoneTask
func (s *taskManagementService) CreateMilestoneTask(clientIDStr, milestoneIDStr string, req dto.CreateMilestoneTaskRequestBody) (*dto.MilestoneTaskItem, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrTMDMilestoneNotFound
	}

	milestone, quest, err := s.repo.GetMilestoneWithQuest(milestoneUUID)
	if err != nil || milestone == nil || quest == nil {
		return nil, ErrTMDMilestoneNotFound
	}

	// Verify either client owns it or assigned freelancer creates it
	isClient := quest.ClientID == clientUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == clientUUID
	if !isClient && !isFreelancer {
		return nil, ErrTMDUnauthorized
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 2 {
		return nil, errors.New("task title must be at least 2 characters")
	}

	status := models.TaskStatusToDo
	if req.Status != "" {
		status = models.TaskStatus(req.Status)
		if status != models.TaskStatusToDo && status != models.TaskStatusInProgress && status != models.TaskStatusCompleted {
			return nil, ErrTMDInvalidStatus
		}
	}

	var assigneeUUID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		parsed, err := uuid.Parse(*req.AssigneeID)
		if err == nil {
			assigneeUUID = &parsed
		}
	} else if quest.AssignedFreelancerID != nil {
		assigneeUUID = quest.AssignedFreelancerID
	}

	task := &models.Task{
		MilestoneID: milestone.ID,
		QuestID:     quest.ID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		AssigneeID:  assigneeUUID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Status:      status,
	}

	if err := s.repo.CreateTask(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// If milestone was pending, move to in_progress
	if milestone.Status == models.MilestonePending {
		_ = s.repo.UpdateMilestoneStatus(milestone.ID, models.MilestoneInProgress, nil)
	}

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		TaskID:      &task.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityTaskCreated,
		Title:       fmt.Sprintf("Task '%s' Created", task.Title),
		Description: fmt.Sprintf("New task added to milestone '%s'", milestone.Title),
	})

	var startStr *string
	if task.StartDate != nil {
		s := task.StartDate.Format("2006-01-02")
		startStr = &s
	}
	var endStr *string
	if task.EndDate != nil {
		s := task.EndDate.Format("2006-01-02")
		endStr = &s
	}

	return &dto.MilestoneTaskItem{
		SlNo:        1,
		ID:          task.ID.String(),
		Name:        task.Title,
		Description: task.Description,
		Assignee:    "Assigned",
		StartDate:   startStr,
		EndDate:     endStr,
		Status:      string(task.Status),
	}, nil
}

// 14. UpdateTaskStatus
func (s *taskManagementService) UpdateTaskStatus(clientIDStr, taskIDStr, statusStr string) (*dto.MilestoneTaskItem, error) {
	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return nil, ErrTMDTaskNotFound
	}

	task, err := s.repo.GetTaskByID(taskUUID)
	if err != nil || task == nil {
		return nil, ErrTMDTaskNotFound
	}

	status := models.TaskStatus(statusStr)
	if status != models.TaskStatusToDo && status != models.TaskStatusInProgress && status != models.TaskStatusCompleted {
		return nil, ErrTMDInvalidStatus
	}

	if err := s.repo.UpdateTaskStatus(task.ID, status); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	task.Status = status

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     task.QuestID,
		MilestoneID: &task.MilestoneID,
		TaskID:      &task.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityTaskStatusChanged,
		Title:       fmt.Sprintf("Task '%s' %s", task.Title, status),
		Description: fmt.Sprintf("Status changed to %s", status),
	})

	var startStr *string
	if task.StartDate != nil {
		s := task.StartDate.Format("2006-01-02")
		startStr = &s
	}
	var endStr *string
	if task.EndDate != nil {
		s := task.EndDate.Format("2006-01-02")
		endStr = &s
	}

	return &dto.MilestoneTaskItem{
		SlNo:        1,
		ID:          task.ID.String(),
		Name:        task.Title,
		Description: task.Description,
		Assignee:    "Assigned",
		StartDate:   startStr,
		EndDate:     endStr,
		Status:      string(task.Status),
	}, nil
}

// Helper: Calculate progress percentage across milestones
func calculateQuestProgress(milestones []models.QuestMilestone) int {
	if len(milestones) == 0 {
		return 0
	}
	completed := 0
	for _, m := range milestones {
		if m.Status == models.MilestoneApproved {
			completed++
		}
	}
	return int(math.Round((float64(completed) / float64(len(milestones))) * 100))
}

// Helper: Calculate individual milestone progress
func calculateMilestoneProgress(m models.QuestMilestone) int {
	if m.Status == models.MilestoneApproved {
		return 100
	}
	if len(m.Tasks) > 0 {
		done := 0
		for _, t := range m.Tasks {
			if t.Status == models.TaskStatusCompleted {
				done++
			}
		}
		return int(math.Round((float64(done) / float64(len(m.Tasks))) * 100))
	}
	switch m.Status {
	case models.MilestoneSubmitted:
		return 90
	case models.MilestoneInProgress:
		return 25
	default:
		return 0
	}
}
