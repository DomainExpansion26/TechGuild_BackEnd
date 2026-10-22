package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// Service errors
// ============================================================

var (
	ErrQMSQuestNotFound         = errors.New("quest not found")
	ErrQMSMilestoneNotFound     = errors.New("milestone not found")
	ErrQMSUnauthorized          = errors.New("not authorized for this action")
	ErrQMSInvalidState          = errors.New("action not allowed in current milestone state")
	ErrQMSNotAssignedFreelancer = errors.New("only the assigned freelancer can perform this action")
	ErrQMSNotQuestClient        = errors.New("only the quest's client can perform this action")
	ErrQMSQuestNotInProgress    = errors.New("quest is not in progress")
	ErrQMSCannotDelete          = errors.New("milestone cannot be deleted in its current state")
	ErrQMSCannotEdit            = errors.New("milestone cannot be edited in its current state")
	ErrQMSAmountRequired        = errors.New("milestone amount is required")
	ErrQMSTitleRequired         = errors.New("milestone title is required")
	ErrQMSRevisionNoteRequired  = errors.New("revision note is required and must be at least 5 characters")
)

// ============================================================
// Service
// ============================================================

type QuestMilestoneService interface {
	// Freelancer actions
	CreateMilestone(userID, questIDStr string, req dto.CreateQuestMilestoneBody) (*dto.MilestoneDetailResponse, error)
	UpdateMilestone(userID, questIDStr, milestoneIDStr string, req dto.UpdateQuestMilestoneBody) (*dto.MilestoneDetailResponse, error)
	DeleteMilestone(userID, questIDStr, milestoneIDStr string) error
	SendForReview(userID, questIDStr, milestoneIDStr string, req dto.SendForReviewBody) (*dto.MilestoneDetailResponse, error)
	ResubmitAfterRevision(userID, questIDStr, milestoneIDStr string, req dto.SendForReviewBody) (*dto.MilestoneDetailResponse, error)

	// Client actions
	ApproveMilestone(userID, questIDStr, milestoneIDStr string, req dto.ApproveMilestoneBody) (*dto.MilestoneDetailResponse, error)
	RequestRevision(userID, questIDStr, milestoneIDStr string, req dto.RequestMilestoneRevisionBody) (*dto.MilestoneDetailResponse, error)

	// Read
	GetMilestone(userID, questIDStr, milestoneIDStr string) (*dto.MilestoneDetailResponse, error)
	ListMilestones(userID, questIDStr string) (*dto.ListQuestMilestonesResponse, error)
	GetRevisions(userID, questIDStr, milestoneIDStr string) (*dto.GetMilestoneRevisionsResponse, error)

	SubmitRoadmap(userID, questIDStr string, req dto.SubmitRoadmapBody) (int64, error)
	ApproveRoadmap(userID, questIDStr string, req dto.ApproveRoadmapBody) (int64, error)
}

type questMilestoneService struct {
	cfg           *config.Config
	milestoneRepo repository.QuestMilestoneRepository
	questRepo     repository.QuestRepository
	activityRepo  repository.TaskManagementRepository
}

func NewQuestMilestoneService(cfg *config.Config) QuestMilestoneService {
	return &questMilestoneService{
		cfg:           cfg,
		milestoneRepo: repository.NewQuestMilestoneRepository(),
		questRepo:     repository.NewQuestRepository(),
		activityRepo:  repository.NewTaskManagementRepository(),
	}
}

// ============================================================
// 1. CREATE MILESTONE (freelancer)
// ============================================================

func (s *questMilestoneService) CreateMilestone(
	userID, questIDStr string,
	req dto.CreateQuestMilestoneBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}

	// Load quest
	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}

	// Only assigned freelancer can create milestones
	if quest.AssignedFreelancerID == nil || *quest.AssignedFreelancerID != userUUID {
		return nil, ErrQMSNotAssignedFreelancer
	}

	// Quest must be in progress
	if quest.Status != models.QuestStatusInProgress {
		return nil, ErrQMSQuestNotInProgress
	}

	// Validate
	title := strings.TrimSpace(req.Title)
	if len(title) < 2 {
		return nil, ErrQMSTitleRequired
	}
	if req.Amount <= 0 {
		return nil, ErrQMSAmountRequired
	}

	// Get next order index
	maxOrder, err := s.milestoneRepo.GetMaxOrderIndex(questUUID)
	if err != nil {
		return nil, err
	}

	// Compute due days
	dueDays := req.DueDays
	if dueDays <= 0 {
		if req.StartDate != nil && req.DueDate != nil && req.DueDate.After(*req.StartDate) {
			dueDays = int(req.DueDate.Sub(*req.StartDate).Hours() / 24)
		} else {
			dueDays = 7
		}
	}

	// Build milestone
	milestone := &models.QuestMilestone{
		QuestID:     questUUID,
		CreatedByID: userUUID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		AmountMinor: utils.RupeesToPaise(req.Amount),
		DueDays:     dueDays,
		OrderIndex:  maxOrder + 1,
		Status:      models.MilestoneStatusDraft,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
	}

	// Build tasks (optional)
	var tasks []models.Task
	for _, t := range req.Tasks {
		tasks = append(tasks, models.Task{
			MilestoneID: milestone.ID,
			QuestID:     questUUID,
			Title:       strings.TrimSpace(t.Title),
			Description: strings.TrimSpace(t.Description),
			StartDate:   t.StartDate,
			EndDate:     t.EndDate,
			Status:      models.TaskStatusToDo,
		})
	}

	// Persist atomically
	if err := s.milestoneRepo.CreateWithTasks(milestone, tasks); err != nil {
		return nil, fmt.Errorf("failed to create milestone: %w", err)
	}

	// Activity log
	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneCreated,
		Title:       fmt.Sprintf("Milestone '%s' Created", milestone.Title),
		Description: fmt.Sprintf("Freelancer created milestone for ₹%.2f", utils.PaiseToRupees(milestone.AmountMinor)),
	})

	// Reload for response
	reloaded, err := s.milestoneRepo.FindByID(milestone.ID)
	if err != nil {
		return nil, err
	}
	resp := s.toDetailResponse(reloaded, nil)
	return &resp, nil
}

// ============================================================
// 2. UPDATE MILESTONE (freelancer)
// ============================================================

func (s *questMilestoneService) UpdateMilestone(
	userID, questIDStr, milestoneIDStr string,
	req dto.UpdateQuestMilestoneBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	// Load milestone
	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	// Ownership check
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.CreatedByID != userUUID {
		return nil, ErrQMSNotAssignedFreelancer
	}

	// Only draft or revision_in_progress can be edited
	if milestone.Status != models.MilestoneStatusDraft &&
		milestone.Status != models.MilestoneStatusRevisionInProgress &&
		milestone.Status != models.MilestoneStatusRevisionRequested {
		return nil, ErrQMSCannotEdit
	}

	// Apply updates (partial)
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if len(t) < 2 {
			return nil, ErrQMSTitleRequired
		}
		milestone.Title = t
	}
	if req.Description != nil {
		milestone.Description = strings.TrimSpace(*req.Description)
	}
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, ErrQMSAmountRequired
		}
		milestone.AmountMinor = utils.RupeesToPaise(*req.Amount)
	}
	if req.StartDate != nil {
		milestone.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		milestone.DueDate = req.DueDate
	}
	if req.DueDays != nil {
		milestone.DueDays = *req.DueDays
	}

	// If we were in revision_requested, move to revision_in_progress
	if milestone.Status == models.MilestoneStatusRevisionRequested {
		milestone.Status = models.MilestoneStatusRevisionInProgress
	}

	if err := s.milestoneRepo.Update(milestone); err != nil {
		return nil, fmt.Errorf("failed to update milestone: %w", err)
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneRevisionInProgress,
		Title:       fmt.Sprintf("Milestone '%s' Updated", milestone.Title),
		Description: "Freelancer updated milestone details",
	})

	resp := s.toDetailResponse(milestone, nil)
	return &resp, nil
}

// ============================================================
// 3. DELETE MILESTONE (freelancer)
// ============================================================

func (s *questMilestoneService) DeleteMilestone(userID, questIDStr, milestoneIDStr string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return ErrQMSMilestoneNotFound
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return ErrQMSMilestoneNotFound
	}
	if milestone.CreatedByID != userUUID {
		return ErrQMSNotAssignedFreelancer
	}

	// Only draft can be deleted
	if milestone.Status != models.MilestoneStatusDraft {
		return ErrQMSCannotDelete
	}

	if err := s.milestoneRepo.Delete(milestoneUUID); err != nil {
		return err
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneDeleted,
		Title:       fmt.Sprintf("Milestone '%s' Deleted", milestone.Title),
		Description: "Freelancer deleted draft milestone",
	})
	return nil
}

// ============================================================
// 4. SEND FOR REVIEW (freelancer) — draft → awaiting_client_review
// ============================================================

func (s *questMilestoneService) SendForReview(
	userID, questIDStr, milestoneIDStr string,
	req dto.SendForReviewBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.CreatedByID != userUUID {
		return nil, ErrQMSNotAssignedFreelancer
	}

	// Only draft can be sent
	if milestone.Status != models.MilestoneStatusDraft {
		return nil, ErrQMSInvalidState
	}

	// Amount must be > 0
	if milestone.AmountMinor <= 0 {
		return nil, ErrQMSAmountRequired
	}

	if err := s.milestoneRepo.UpdateStatus(
		milestone.ID,
		models.MilestoneStatusAwaitingClientReview,
	); err != nil {
		return nil, err
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneSentForReview,
		Title:       fmt.Sprintf("Milestone '%s' Sent for Review", milestone.Title),
		Description: req.Note,
	})

	milestone.Status = models.MilestoneStatusAwaitingClientReview
	resp := s.toDetailResponse(milestone, nil)
	return &resp, nil
}

// ============================================================
// 5. RESUBMIT AFTER REVISION (freelancer) — revision_in_progress → awaiting_client_review
// ============================================================

func (s *questMilestoneService) ResubmitAfterRevision(
	userID, questIDStr, milestoneIDStr string,
	req dto.SendForReviewBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.CreatedByID != userUUID {
		return nil, ErrQMSNotAssignedFreelancer
	}

	// Can only resubmit from revision states
	if milestone.Status != models.MilestoneStatusRevisionInProgress &&
		milestone.Status != models.MilestoneStatusRevisionRequested {
		return nil, ErrQMSInvalidState
	}

	if err := s.milestoneRepo.UpdateStatus(
		milestone.ID,
		models.MilestoneStatusAwaitingClientReview,
	); err != nil {
		return nil, err
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneResubmitted,
		Title:       fmt.Sprintf("Milestone '%s' Resubmitted", milestone.Title),
		Description: req.Note,
	})

	milestone.Status = models.MilestoneStatusAwaitingClientReview
	resp := s.toDetailResponse(milestone, nil)
	return &resp, nil
}

// ============================================================
// 6. APPROVE MILESTONE (client) — awaiting_client_review → agreed
// ============================================================

func (s *questMilestoneService) ApproveMilestone(
	userID, questIDStr, milestoneIDStr string,
	req dto.ApproveMilestoneBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrQMSNotQuestClient
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}

	if milestone.Status != models.MilestoneStatusAwaitingClientReview {
		return nil, ErrQMSInvalidState
	}

	now := time.Now()
	if err := s.milestoneRepo.UpdateStatusWithTimestamp(
		milestone.ID,
		models.MilestoneStatusAgreed,
		"agreed_at",
		now,
	); err != nil {
		return nil, err
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneAgreed,
		Title:       fmt.Sprintf("Milestone '%s' Approved", milestone.Title),
		Description: req.Feedback,
	})

	milestone.Status = models.MilestoneStatusAgreed
	milestone.AgreedAt = &now
	resp := s.toDetailResponse(milestone, nil)
	return &resp, nil
}

// ============================================================
// 7. REQUEST REVISION (client) — awaiting_client_review → revision_requested
// ============================================================

func (s *questMilestoneService) RequestRevision(
	userID, questIDStr, milestoneIDStr string,
	req dto.RequestMilestoneRevisionBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	note := strings.TrimSpace(req.Note)
	if len(note) < 5 {
		return nil, ErrQMSRevisionNoteRequired
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrQMSNotQuestClient
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}

	if milestone.Status != models.MilestoneStatusAwaitingClientReview {
		return nil, ErrQMSInvalidState
	}

	// Atomic: create revision entry + update milestone status + increment count
	err = s.milestoneRepo.WithTransaction(func(txRepo repository.QuestMilestoneRepository) error {
		// Get next round number
		maxRound, err := txRepo.GetMaxRevisionRound(milestone.ID)
		if err != nil {
			return err
		}

		rev := &models.MilestoneRevision{
			MilestoneID:         milestone.ID,
			RequestedByID:       userUUID,
			TitleSnapshot:       milestone.Title,
			DescriptionSnapshot: milestone.Description,
			AmountMinorSnapshot: milestone.AmountMinor,
			Note:                note,
			AttachmentURL:       req.AttachmentURL,
			RoundNumber:         maxRound + 1,
		}
		if err := txRepo.CreateRevision(rev); err != nil {
			return err
		}

		// Update milestone fields
		now := time.Now()
		milestone.Status = models.MilestoneStatusRevisionRequested
		milestone.RevisionCount = maxRound + 1
		milestone.LastRevisionAt = &now
		milestone.ClientFeedback = note

		return txRepo.Update(milestone)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request revision: %w", err)
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneRevisionRequested,
		Title:       fmt.Sprintf("Revision Requested for '%s'", milestone.Title),
		Description: note,
	})

	resp := s.toDetailResponse(milestone, nil)
	return &resp, nil
}

// ============================================================
// 8. GET MILESTONE
// ============================================================

func (s *questMilestoneService) GetMilestone(
	userID, questIDStr, milestoneIDStr string,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}

	// Access: client OR assigned freelancer
	isClient := quest.ClientID == userUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userUUID
	if !isClient && !isFreelancer {
		return nil, ErrQMSUnauthorized
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrQMSMilestoneNotFound
	}

	revisions, _ := s.milestoneRepo.GetRevisionsByMilestone(milestoneUUID)

	resp := s.toDetailResponse(milestone, revisions)
	return &resp, nil
}

// ============================================================
// 9. LIST MILESTONES
// ============================================================

func (s *questMilestoneService) ListMilestones(
	userID, questIDStr string,
) (*dto.ListQuestMilestonesResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}

	isClient := quest.ClientID == userUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userUUID
	if !isClient && !isFreelancer {
		return nil, ErrQMSUnauthorized
	}

	milestones, err := s.milestoneRepo.FindByQuestID(questUUID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.MilestoneSummaryItem, 0, len(milestones))
	for _, m := range milestones {
		items = append(items, dto.MilestoneSummaryItem{
			ID:            m.ID.String(),
			Title:         m.Title,
			Description:   m.Description,
			AmountMinor:   m.AmountMinor,
			Amount:        utils.PaiseToRupees(m.AmountMinor),
			Currency:      "INR",
			DueDays:       m.DueDays,
			OrderIndex:    m.OrderIndex,
			Status:        string(m.Status),
			TaskCount:     len(m.Tasks),
			RevisionCount: m.RevisionCount,
			StartDate:     m.StartDate,
			DueDate:       m.DueDate,
			CreatedAt:     m.CreatedAt,
			UpdatedAt:     m.UpdatedAt,
		})
	}

	return &dto.ListQuestMilestonesResponse{
		Milestones: items,
		Total:      len(items),
		Currency:   "INR",
	}, nil
}

// ============================================================
// 10. GET REVISIONS
// ============================================================

func (s *questMilestoneService) GetRevisions(
	userID, questIDStr, milestoneIDStr string,
) (*dto.GetMilestoneRevisionsResponse, error) {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrQMSMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrQMSQuestNotFound
	}
	isClient := quest.ClientID == userUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userUUID
	if !isClient && !isFreelancer {
		return nil, ErrQMSUnauthorized
	}

	revisions, err := s.milestoneRepo.GetRevisionsByMilestone(milestoneUUID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.RevisionItem, 0, len(revisions))
	for _, r := range revisions {
		requesterName := ""
		requesterRole := ""
		if r.RequestedBy.ID != uuid.Nil {
			requesterName = r.RequestedBy.FirstName + " " + r.RequestedBy.LastName
		}
		if r.RequestedByID == quest.ClientID {
			requesterRole = "client"
		} else if quest.AssignedFreelancerID != nil && r.RequestedByID == *quest.AssignedFreelancerID {
			requesterRole = "freelancer"
		}

		items = append(items, dto.RevisionItem{
			ID:                  r.ID.String(),
			RequestedByID:       r.RequestedByID.String(),
			RequestedByName:     requesterName,
			RequestedByRole:     requesterRole,
			TitleSnapshot:       r.TitleSnapshot,
			DescriptionSnapshot: r.DescriptionSnapshot,
			AmountMinorSnapshot: r.AmountMinorSnapshot,
			Note:                r.Note,
			AttachmentURL:       r.AttachmentURL,
			RoundNumber:         r.RoundNumber,
			CreatedAt:           r.CreatedAt,
		})
	}

	return &dto.GetMilestoneRevisionsResponse{
		MilestoneID: milestoneUUID.String(),
		Total:       len(items),
		Revisions:   items,
	}, nil
}

// ============================================================
// Helper — convert model → DTO
// ============================================================

func (s *questMilestoneService) toDetailResponse(
	m *models.QuestMilestone,
	revisions []models.MilestoneRevision,
) dto.MilestoneDetailResponse {

	resp := dto.MilestoneDetailResponse{
		ID:             m.ID.String(),
		QuestID:        m.QuestID.String(),
		CreatedByID:    m.CreatedByID.String(),
		Title:          m.Title,
		Description:    m.Description,
		AmountMinor:    m.AmountMinor,
		Amount:         utils.PaiseToRupees(m.AmountMinor),
		Currency:       "INR",
		DueDays:        m.DueDays,
		OrderIndex:     m.OrderIndex,
		Status:         string(m.Status),
		RevisionCount:  m.RevisionCount,
		ClientFeedback: m.ClientFeedback,
		StartDate:      m.StartDate,
		DueDate:        m.DueDate,
		AgreedAt:       m.AgreedAt,
		FundedAt:       m.FundedAt,
		ReleasedAt:     m.ReleasedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}

	// Tasks
	tasks := make([]dto.MilestoneTaskSummary, 0, len(m.Tasks))
	for _, t := range m.Tasks {
		tasks = append(tasks, dto.MilestoneTaskSummary{
			ID:          t.ID.String(),
			Title:       t.Title,
			Description: t.Description,
			Status:      string(t.Status),
			StartDate:   t.StartDate,
			EndDate:     t.EndDate,
		})
	}
	resp.Tasks = tasks

	// Revisions
	if len(revisions) > 0 {
		items := make([]dto.RevisionItem, 0, len(revisions))
		for _, r := range revisions {
			requesterName := ""
			if r.RequestedBy.ID != uuid.Nil {
				requesterName = r.RequestedBy.FirstName + " " + r.RequestedBy.LastName
			}
			items = append(items, dto.RevisionItem{
				ID:                  r.ID.String(),
				RequestedByID:       r.RequestedByID.String(),
				RequestedByName:     requesterName,
				TitleSnapshot:       r.TitleSnapshot,
				DescriptionSnapshot: r.DescriptionSnapshot,
				AmountMinorSnapshot: r.AmountMinorSnapshot,
				Note:                r.Note,
				AttachmentURL:       r.AttachmentURL,
				RoundNumber:         r.RoundNumber,
				CreatedAt:           r.CreatedAt,
			})
		}
		resp.Revisions = items
	}

	return resp
}

// ============================================================
// 11. SUBMIT ROADMAP (Freelancer)
// ============================================================
func (s *questMilestoneService) SubmitRoadmap(userID, questIDStr string, req dto.SubmitRoadmapBody) (int64, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return 0, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return 0, ErrQMSQuestNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return 0, ErrQMSQuestNotFound
	}

	if quest.AssignedFreelancerID == nil || *quest.AssignedFreelancerID != userUUID {
		return 0, ErrQMSNotAssignedFreelancer
	}
	if quest.Status != models.QuestStatusInProgress {
		return 0, ErrQMSQuestNotInProgress
	}

	// Bulk update all 'draft' milestones to 'awaiting_client_review'
	count, err := s.milestoneRepo.BulkUpdateMilestoneStatus(
		questUUID,
		models.MilestoneStatusDraft,
		models.MilestoneStatusAwaitingClientReview,
	)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errors.New("no draft milestones found to submit")
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneSentForReview,
		Title:       "Roadmap Submitted",
		Description: req.Note,
	})

	return count, nil
}

// ============================================================
// 12. APPROVE ROADMAP (Client)
// ============================================================
func (s *questMilestoneService) ApproveRoadmap(userID, questIDStr string, req dto.ApproveRoadmapBody) (int64, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return 0, ErrInvalidUserID
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return 0, ErrQMSQuestNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return 0, ErrQMSQuestNotFound
	}

	if quest.ClientID != userUUID {
		return 0, ErrQMSNotQuestClient
	}

	// Bulk update all 'awaiting_client_review' to 'agreed'
	count, err := s.milestoneRepo.BulkUpdateMilestoneStatus(
		questUUID,
		models.MilestoneStatusAwaitingClientReview,
		models.MilestoneStatusAgreed,
	)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errors.New("no milestones pending review")
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneAgreed,
		Title:       "Roadmap Approved",
		Description: req.Feedback,
	})

	return count, nil
}

// Silence unused import warning if needed
var _ = gorm.ErrRecordNotFound
