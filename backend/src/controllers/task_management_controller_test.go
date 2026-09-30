package controllers_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

type mockServiceForController struct {
	clientID uuid.UUID
	questID  uuid.UUID
	msID     uuid.UUID
	taskID   uuid.UUID
}

func (m *mockServiceForController) GetMetrics(clientIDStr, questIDStr string) (*dto.TaskManagementMetricsResponse, error) {
	return &dto.TaskManagementMetricsResponse{
		ActiveQuests:      2,
		PendingReviews:    1,
		TasksInProgress:   5,
		UpcomingDeadlines: 2,
		CompletedQuests:   1,
	}, nil
}

func (m *mockServiceForController) GetActiveQuests(clientIDStr string) (*dto.ActiveQuestsListResponse, error) {
	return &dto.ActiveQuestsListResponse{
		TotalActive: 1,
		Quests: []dto.ActiveQuestSummaryItem{
			{
				ID:       m.questID.String(),
				Title:    "E-Commerce Website Redesign",
				Progress: 50,
				Status:   "in_progress",
			},
		},
	}, nil
}

func (m *mockServiceForController) GetQuestDashboard(clientIDStr, questIDStr string) (*dto.QuestDashboardResponse, error) {
	return &dto.QuestDashboardResponse{
		Quest: dto.QuestDashboardOverview{
			ID:              m.questID.String(),
			Title:           "E-Commerce Website Redesign",
			Status:          "in_progress",
			HasTasksCreated: true,
			CanApproveQuest: false,
			OverallProgress: 50,
		},
		MilestonesOverview: dto.MilestonesOverviewStats{
			CompletionPercentage: 50,
			TotalMilestones:      2,
			CompletedMilestones:  1,
			InProgressMilestones: 1,
			PendingMilestones:    0,
		},
	}, nil
}

func (m *mockServiceForController) GetQuestMilestones(clientIDStr, questIDStr string) (*dto.QuestMilestonesTableResponse, error) {
	return &dto.QuestMilestonesTableResponse{
		Milestones: []dto.QuestMilestoneRowItem{
			{
				SlNo:       1,
				ID:         m.msID.String(),
				Name:       "Milestone 1",
				TaskCount:  3,
				Progress:   100,
				Amount:     15000,
				Currency:   "INR",
				Status:     "approved",
				OrderIndex: 1,
			},
		},
		TotalAmount: 15000,
		Currency:    "INR",
	}, nil
}

func (m *mockServiceForController) AddMilestone(clientIDStr, questIDStr string, req dto.AddMilestoneRequestBody) (*dto.AddMilestoneResponse, error) {
	return &dto.AddMilestoneResponse{
		Message: "Milestone added successfully",
		Milestone: dto.QuestMilestoneRowItem{
			SlNo:   2,
			ID:     uuid.New().String(),
			Name:   req.Title,
			Amount: req.Amount,
			Status: "pending",
		},
		NewTotalBudget: 25000,
	}, nil
}

func (m *mockServiceForController) DeleteMilestone(clientIDStr, milestoneIDStr string) (*dto.DeleteMilestoneTMDResponse, error) {
	return &dto.DeleteMilestoneTMDResponse{
		Message:     "Milestone and associated tasks deleted successfully",
		MilestoneID: milestoneIDStr,
	}, nil
}

func (m *mockServiceForController) GetMilestoneTasks(clientIDStr, milestoneIDStr string) (*dto.MilestoneTasksResponse, error) {
	return &dto.MilestoneTasksResponse{
		MilestoneID:   milestoneIDStr,
		MilestoneName: "Milestone 1",
		Tasks: []dto.MilestoneTaskItem{
			{
				SlNo:   1,
				ID:     m.taskID.String(),
				Name:   "Navbar Wireframe",
				Status: "completed",
			},
		},
	}, nil
}

func (m *mockServiceForController) ApproveMilestone(clientIDStr, milestoneIDStr string, feedback string) (*dto.ApproveMilestoneTMDResponse, error) {
	return &dto.ApproveMilestoneTMDResponse{
		Message:              "Milestone approved successfully",
		MilestoneID:          milestoneIDStr,
		Status:               "approved",
		OverallQuestProgress: 100,
		CanApproveQuest:      true,
	}, nil
}

func (m *mockServiceForController) CancelMilestoneApproval(clientIDStr, milestoneIDStr string, reason string) (*dto.CancelMilestoneApprovalTMDResponse, error) {
	return &dto.CancelMilestoneApprovalTMDResponse{
		Message:              "Milestone approval cancelled successfully",
		MilestoneID:          milestoneIDStr,
		Status:               "in_progress",
		OverallQuestProgress: 50,
		CanApproveQuest:      false,
	}, nil
}

func (m *mockServiceForController) ApproveQuest(clientIDStr, questIDStr string, req dto.ApproveQuestTMDRequestBody) (*dto.ApproveQuestTMDResponse, error) {
	return &dto.ApproveQuestTMDResponse{
		Message:     "Quest approved and marked as completed",
		QuestID:     questIDStr,
		Status:      "completed",
		CompletedAt: time.Now(),
	}, nil
}

func (m *mockServiceForController) CancelQuestApproval(clientIDStr, questIDStr string, reason string) (*dto.CancelQuestApprovalTMDResponse, error) {
	return &dto.CancelQuestApprovalTMDResponse{
		Message: "Quest approval cancelled and reopened",
		QuestID: questIDStr,
		Status:  "in_progress",
	}, nil
}

func (m *mockServiceForController) GetQuestActivities(clientIDStr, questIDStr string, limit, offset int) (*dto.QuestActivitiesTMDResponse, error) {
	return &dto.QuestActivitiesTMDResponse{
		QuestID: questIDStr,
		Total:   1,
		Activities: []dto.RecentActivityItem{
			{
				ID:         uuid.New().String(),
				ActionType: "milestone_approved",
				Title:      "Milestone 1 Approved",
				CreatedAt:  time.Now(),
			},
		},
	}, nil
}

func (m *mockServiceForController) CreateMilestoneTask(clientIDStr, milestoneIDStr string, req dto.CreateMilestoneTaskRequestBody) (*dto.MilestoneTaskItem, error) {
	return &dto.MilestoneTaskItem{
		SlNo:   1,
		ID:     uuid.New().String(),
		Name:   req.Title,
		Status: "todo",
	}, nil
}

func (m *mockServiceForController) UpdateTaskStatus(clientIDStr, taskIDStr, statusStr string) (*dto.MilestoneTaskItem, error) {
	return &dto.MilestoneTaskItem{
		SlNo:   1,
		ID:     taskIDStr,
		Name:   "Task",
		Status: statusStr,
	}, nil
}

func TestTaskManagementControllerHandlers(t *testing.T) {
	clientID := uuid.New()
	questID := uuid.New()
	msID := uuid.New()
	taskID := uuid.New()

	mockSvc := &mockServiceForController{
		clientID: clientID,
		questID:  questID,
		msID:     msID,
		taskID:   taskID,
	}
	controllers.SetTaskManagementServiceForTesting(mockSvc)

	// Create authenticated context with UserIDKey
	authCtx := context.WithValue(context.Background(), middleware.UserIDKey, clientID.String())
	unauthCtx := context.Background()

	// 1. Unauthorized check
	_, err := controllers.GetTaskManagementMetricsHandler(unauthCtx, &dto.TaskManagementMetricsInput{})
	if err == nil {
		t.Fatalf("expected unauthorized error for context without UserID")
	}

	// 2. Metrics Handler
	metrics, err := controllers.GetTaskManagementMetricsHandler(authCtx, &dto.TaskManagementMetricsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Body.ActiveQuests != 2 {
		t.Errorf("expected 2 active quests, got %d", metrics.Body.ActiveQuests)
	}

	// 3. Active Quests Handler
	activeQuests, err := controllers.GetActiveQuestsHandler(authCtx, &dto.GetActiveQuestsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if activeQuests.Body.TotalActive != 1 {
		t.Errorf("expected 1 active quest, got %d", activeQuests.Body.TotalActive)
	}

	// 4. Quest Dashboard Handler
	dash, err := controllers.GetQuestDashboardHandler(authCtx, &dto.GetQuestDashboardInput{QuestID: questID.String()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dash.Body.Quest.Title != "E-Commerce Website Redesign" {
		t.Errorf("expected title, got %s", dash.Body.Quest.Title)
	}

	// 5. Add Milestone Handler
	addRes, err := controllers.AddMilestoneHandler(authCtx, &dto.AddMilestoneInput{
		QuestID: questID.String(),
		Body: dto.AddMilestoneRequestBody{
			Title:  "Backend API",
			Amount: 10000,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addRes.Body.NewTotalBudget != 25000 {
		t.Errorf("expected 25000, got %f", addRes.Body.NewTotalBudget)
	}

	// 6. Approve Milestone Handler
	apprRes, err := controllers.ApproveMilestoneTMDHandler(authCtx, &dto.ApproveMilestoneTMDInput{
		MilestoneID: msID.String(),
		Body:        dto.ApproveMilestoneTMDRequestBody{Feedback: "Approved"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apprRes.Body.Status != "approved" {
		t.Errorf("expected approved status, got %s", apprRes.Body.Status)
	}

	// 7. Cancel Milestone Approval Handler
	cancelApprRes, err := controllers.CancelMilestoneApprovalTMDHandler(authCtx, &dto.CancelMilestoneApprovalTMDInput{
		MilestoneID: msID.String(),
		Body:        dto.CancelMilestoneApprovalTMDRequestBody{Reason: "Need changes"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelApprRes.Body.Status != "in_progress" {
		t.Errorf("expected in_progress, got %s", cancelApprRes.Body.Status)
	}

	// 8. Approve Quest Handler
	questApprRes, err := controllers.ApproveQuestTMDHandler(authCtx, &dto.ApproveQuestTMDInput{
		QuestID: questID.String(),
		Body: dto.ApproveQuestTMDRequestBody{
			Rating:   5,
			Feedback: "Excellent work",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if questApprRes.Body.Status != "completed" {
		t.Errorf("expected completed, got %s", questApprRes.Body.Status)
	}

	// 9. Cancel Quest Approval Handler
	cancelQuestRes, err := controllers.CancelQuestApprovalTMDHandler(authCtx, &dto.CancelQuestApprovalTMDInput{
		QuestID: questID.String(),
		Body:    dto.CancelQuestApprovalTMDRequestBody{Reason: "Defect found"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelQuestRes.Body.Status != "in_progress" {
		t.Errorf("expected in_progress, got %s", cancelQuestRes.Body.Status)
	}

	// 10. Delete Milestone Handler
	delRes, err := controllers.DeleteMilestoneTMDHandler(authCtx, &dto.DeleteMilestoneTMDInput{
		MilestoneID: msID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delRes.Body.MilestoneID != msID.String() {
		t.Errorf("expected %s, got %s", msID.String(), delRes.Body.MilestoneID)
	}

	_ = mockSvc
}
