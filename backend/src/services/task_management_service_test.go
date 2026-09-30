package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/services"
)

// mockTaskManagementRepository implements repository.TaskManagementRepository in-memory for testing.
type mockTaskManagementRepository struct {
	quests      map[uuid.UUID]*models.Quest
	milestones  map[uuid.UUID]*models.QuestMilestone
	tasks       map[uuid.UUID]*models.Task
	activities  []models.QuestActivity
	reviews     map[uuid.UUID]*models.QuestReview
	approvalLog []models.MilestoneApprovalLog
}

func newMockRepo() *mockTaskManagementRepository {
	return &mockTaskManagementRepository{
		quests:     make(map[uuid.UUID]*models.Quest),
		milestones: make(map[uuid.UUID]*models.QuestMilestone),
		tasks:      make(map[uuid.UUID]*models.Task),
		reviews:    make(map[uuid.UUID]*models.QuestReview),
	}
}

func (m *mockRepoWrapper) GetClientActiveQuests(clientID uuid.UUID) ([]models.Quest, error) {
	var list []models.Quest
	for _, q := range m.quests {
		if q.ClientID == clientID && (q.Status == models.QuestStatusInProgress || q.Status == models.QuestStatusCompleted) {
			list = append(list, *q)
		}
	}
	return list, nil
}

func (m *mockRepoWrapper) GetQuestByIDAndClient(questID, clientID uuid.UUID) (*models.Quest, error) {
	q, ok := m.quests[questID]
	if !ok || q.ClientID != clientID {
		return nil, errors.New("not found")
	}
	return q, nil
}

func (m *mockRepoWrapper) GetMilestoneByID(milestoneID uuid.UUID) (*models.QuestMilestone, error) {
	milestone, ok := m.milestones[milestoneID]
	if !ok {
		return nil, errors.New("milestone not found")
	}
	return milestone, nil
}

func (m *mockRepoWrapper) GetMilestoneWithQuest(milestoneID uuid.UUID) (*models.QuestMilestone, *models.Quest, error) {
	milestone, ok := m.milestones[milestoneID]
	if !ok {
		return nil, nil, errors.New("milestone not found")
	}
	quest, ok := m.quests[milestone.QuestID]
	if !ok {
		return nil, nil, errors.New("quest not found")
	}
	return milestone, quest, nil
}

func (m *mockRepoWrapper) CreateMilestone(quest *models.Quest, milestone *models.QuestMilestone) error {
	if milestone.ID == uuid.Nil {
		milestone.ID = uuid.New()
	}
	m.milestones[milestone.ID] = milestone
	quest.Milestones = append(quest.Milestones, *milestone)
	quest.BudgetINR += milestone.Amount
	return nil
}

func (m *mockRepoWrapper) DeleteMilestone(milestoneID uuid.UUID, questID uuid.UUID, amount float64) error {
	delete(m.milestones, milestoneID)
	if q, ok := m.quests[questID]; ok {
		var remaining []models.QuestMilestone
		for _, ms := range q.Milestones {
			if ms.ID != milestoneID {
				remaining = append(remaining, ms)
			}
		}
		q.Milestones = remaining
		q.BudgetINR -= amount
	}
	return nil
}

func (m *mockRepoWrapper) UpdateMilestoneStatus(milestoneID uuid.UUID, status models.MilestoneStatus, approvedAt *time.Time) error {
	if ms, ok := m.milestones[milestoneID]; ok {
		ms.Status = status
		ms.ApprovedAt = approvedAt
		if q, qOk := m.quests[ms.QuestID]; qOk {
			for i := range q.Milestones {
				if q.Milestones[i].ID == milestoneID {
					q.Milestones[i].Status = status
					q.Milestones[i].ApprovedAt = approvedAt
				}
			}
		}
		return nil
	}
	return errors.New("milestone not found")
}

func (m *mockRepoWrapper) LogMilestoneApproval(log *models.MilestoneApprovalLog) error {
	m.approvalLog = append(m.approvalLog, *log)
	return nil
}

func (m *mockRepoWrapper) UpdateQuestStatus(questID uuid.UUID, status models.QuestStatus) error {
	if q, ok := m.quests[questID]; ok {
		q.Status = status
		return nil
	}
	return errors.New("quest not found")
}

func (m *mockRepoWrapper) CreateQuestReview(review *models.QuestReview) error {
	m.reviews[review.QuestID] = review
	return nil
}

func (m *mockRepoWrapper) CreateActivity(activity *models.QuestActivity) error {
	m.activities = append(m.activities, *activity)
	return nil
}

func (m *mockRepoWrapper) GetActivitiesByQuestID(questID uuid.UUID, limit, offset int) ([]models.QuestActivity, int64, error) {
	return m.activities, int64(len(m.activities)), nil
}

func (m *mockRepoWrapper) GetRecentActivities(questID uuid.UUID, limit int) ([]models.QuestActivity, error) {
	return m.activities, nil
}

func (m *mockRepoWrapper) GetMetrics(clientID uuid.UUID, questID *uuid.UUID) (dto.TaskManagementMetricsResponse, error) {
	return dto.TaskManagementMetricsResponse{
		ActiveQuests:      1,
		PendingReviews:    0,
		TasksInProgress:   2,
		UpcomingDeadlines: 1,
		CompletedQuests:   0,
	}, nil
}

func (m *mockRepoWrapper) CreateTask(task *models.Task) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	m.tasks[task.ID] = task
	if ms, ok := m.milestones[task.MilestoneID]; ok {
		ms.Tasks = append(ms.Tasks, *task)
		if q, qOk := m.quests[ms.QuestID]; qOk {
			for i := range q.Milestones {
				if q.Milestones[i].ID == ms.ID {
					q.Milestones[i].Tasks = append(q.Milestones[i].Tasks, *task)
				}
			}
		}
	}
	return nil
}

func (m *mockRepoWrapper) GetTaskByID(taskID uuid.UUID) (*models.Task, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (m *mockRepoWrapper) UpdateTaskStatus(taskID uuid.UUID, status models.TaskStatus) error {
	t, ok := m.tasks[taskID]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	return nil
}

type mockRepoWrapper struct {
	*mockTaskManagementRepository
}

// ------------------- TEST CASES -------------------

func setupTestEnvironment() (services.TaskManagementService, *mockRepoWrapper, uuid.UUID, *models.Quest) {
	rawMock := newMockRepo()
	mockRepo := &mockRepoWrapper{rawMock}
	svc := services.NewTaskManagementServiceWithRepo(mockRepo)

	clientID := uuid.New()
	questID := uuid.New()
	ms1ID := uuid.New()
	ms2ID := uuid.New()

	ms1 := models.QuestMilestone{
		ID:         ms1ID,
		QuestID:    questID,
		Title:      "Wireframes & Research",
		Amount:     15000,
		DueDays:    5,
		OrderIndex: 1,
		Status:     models.MilestonePending,
	}
	ms2 := models.QuestMilestone{
		ID:         ms2ID,
		QuestID:    questID,
		Title:      "UI Design & Prototype",
		Amount:     25000,
		DueDays:    10,
		OrderIndex: 2,
		Status:     models.MilestonePending,
	}

	quest := &models.Quest{
		ID:         questID,
		ClientID:   clientID,
		Title:      "E-Commerce Website Redesign",
		Category:   "UI/UX Design",
		BudgetINR:  40000,
		Status:     models.QuestStatusInProgress,
		Milestones: []models.QuestMilestone{ms1, ms2},
	}

	mockRepo.quests[questID] = quest
	mockRepo.milestones[ms1ID] = &ms1
	mockRepo.milestones[ms2ID] = &ms2

	return svc, mockRepo, clientID, quest
}

func TestGetActiveQuests(t *testing.T) {
	svc, _, clientID, quest := setupTestEnvironment()

	res, err := svc.GetActiveQuests(clientID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalActive != 1 {
		t.Errorf("expected total_active 1, got %d", res.TotalActive)
	}
	if res.Quests[0].Title != quest.Title {
		t.Errorf("expected quest title %s, got %s", quest.Title, res.Quests[0].Title)
	}
	if res.Quests[0].Progress != 0 {
		t.Errorf("expected 0 progress for pending milestones, got %d", res.Quests[0].Progress)
	}
}

func TestGetQuestDashboard(t *testing.T) {
	svc, _, clientID, quest := setupTestEnvironment()

	dashboard, err := svc.GetQuestDashboard(clientID.String(), quest.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.Quest.Title != quest.Title {
		t.Errorf("expected title %s, got %s", quest.Title, dashboard.Quest.Title)
	}
	if dashboard.MilestonesOverview.TotalMilestones != 2 {
		t.Errorf("expected 2 milestones, got %d", dashboard.MilestonesOverview.TotalMilestones)
	}
	if dashboard.Quest.CanApproveQuest {
		t.Errorf("expected CanApproveQuest to be false initially")
	}
}

func TestAddAndMilestoneManagement(t *testing.T) {
	svc, _, clientID, quest := setupTestEnvironment()

	// 1. Add Milestone
	addReq := dto.AddMilestoneRequestBody{
		Title:       "Final Testing & QA",
		Description: "Testing cross-browser support and performance",
		Amount:      10000,
		DueDays:     5,
	}

	addRes, err := svc.AddMilestone(clientID.String(), quest.ID.String(), addReq)
	if err != nil {
		t.Fatalf("failed to add milestone: %v", err)
	}

	if addRes.NewTotalBudget != 50000 {
		t.Errorf("expected new budget 50000, got %f", addRes.NewTotalBudget)
	}

	// 2. Delete Milestone
	delRes, err := svc.DeleteMilestone(clientID.String(), addRes.Milestone.ID)
	if err != nil {
		t.Fatalf("failed to delete milestone: %v", err)
	}
	if delRes.MilestoneID != addRes.Milestone.ID {
		t.Errorf("expected deleted milestone ID %s, got %s", addRes.Milestone.ID, delRes.MilestoneID)
	}
}

func TestMilestoneApprovalAndQuestCompletionLifecycle(t *testing.T) {
	svc, _, clientID, quest := setupTestEnvironment()
	ms1 := quest.Milestones[0]
	ms2 := quest.Milestones[1]

	// 1. Cannot approve quest before all milestones are approved
	_, err := svc.ApproveQuest(clientID.String(), quest.ID.String(), dto.ApproveQuestTMDRequestBody{
		Rating:   5,
		Feedback: "Looks great",
	})
	if !errors.Is(err, services.ErrTMDAllMilestonesNotDone) {
		t.Fatalf("expected ErrTMDAllMilestonesNotDone, got %v", err)
	}

	// 2. Approve Milestone 1
	appr1, err := svc.ApproveMilestone(clientID.String(), ms1.ID.String(), "Milestone 1 looks great")
	if err != nil {
		t.Fatalf("failed to approve ms1: %v", err)
	}
	if appr1.OverallQuestProgress != 50 {
		t.Errorf("expected 50%% progress after 1 of 2 milestones approved, got %d", appr1.OverallQuestProgress)
	}
	if appr1.CanApproveQuest {
		t.Errorf("expected CanApproveQuest to be false when 1 milestone remains")
	}

	// 3. Cancel Milestone 1 approval
	cancelRes, err := svc.CancelMilestoneApproval(clientID.String(), ms1.ID.String(), "Need revisions on wireframe")
	if err != nil {
		t.Fatalf("failed to cancel ms1 approval: %v", err)
	}
	if cancelRes.Status != string(models.MilestoneInProgress) {
		t.Errorf("expected status in_progress, got %s", cancelRes.Status)
	}

	// Re-approve Milestone 1
	_, _ = svc.ApproveMilestone(clientID.String(), ms1.ID.String(), "Approved after revision")

	// 4. Approve Milestone 2
	appr2, err := svc.ApproveMilestone(clientID.String(), ms2.ID.String(), "Milestone 2 fully delivered")
	if err != nil {
		t.Fatalf("failed to approve ms2: %v", err)
	}
	if appr2.OverallQuestProgress != 100 {
		t.Errorf("expected 100%% progress, got %d", appr2.OverallQuestProgress)
	}
	if !appr2.CanApproveQuest {
		t.Errorf("expected CanApproveQuest to be true after 100%% approved")
	}

	// 5. Now Approve Quest
	questAppr, err := svc.ApproveQuest(clientID.String(), quest.ID.String(), dto.ApproveQuestTMDRequestBody{
		Rating:   5,
		Feedback: "Outstanding work delivered ahead of time!",
	})
	if err != nil {
		t.Fatalf("failed to approve quest: %v", err)
	}
	if questAppr.Status != string(models.QuestStatusCompleted) {
		t.Errorf("expected quest status completed, got %s", questAppr.Status)
	}

	// 6. Test Cancel Quest Approval (Reopen)
	cancelQuestRes, err := svc.CancelQuestApproval(clientID.String(), quest.ID.String(), "Integration patch needed")
	if err != nil {
		t.Fatalf("failed to cancel quest approval: %v", err)
	}
	if cancelQuestRes.Status != string(models.QuestStatusInProgress) {
		t.Errorf("expected quest reopened as in_progress, got %s", cancelQuestRes.Status)
	}
}

func TestTaskCreationAndStatusProgression(t *testing.T) {
	svc, _, clientID, quest := setupTestEnvironment()
	ms1 := quest.Milestones[0]

	// 1. Create Task
	createdTask, err := svc.CreateMilestoneTask(clientID.String(), ms1.ID.String(), dto.CreateMilestoneTaskRequestBody{
		Title:       "Design Navigation Bar",
		Description: "Create interactive Figma header component",
		Status:      "todo",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if createdTask.Name != "Design Navigation Bar" {
		t.Errorf("expected task title 'Design Navigation Bar', got %s", createdTask.Name)
	}

	// 2. Update Task Status
	updatedTask, err := svc.UpdateTaskStatus(clientID.String(), createdTask.ID, "in_progress")
	if err != nil {
		t.Fatalf("failed to update task status: %v", err)
	}
	if updatedTask.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %s", updatedTask.Status)
	}

	// 3. View Tasks under milestone
	tasksRes, err := svc.GetMilestoneTasks(clientID.String(), ms1.ID.String())
	if err != nil {
		t.Fatalf("failed to get milestone tasks: %v", err)
	}
	if len(tasksRes.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasksRes.Tasks))
	}
}
