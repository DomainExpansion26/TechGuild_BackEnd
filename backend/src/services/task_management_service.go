package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

var (
	ErrTMDUnauthorized               = errors.New("unauthorized access to quest")
	ErrTMDQuestNotFound              = errors.New("quest not found")
	ErrTMDMilestoneNotFound          = errors.New("milestone not found")
	ErrTMDTaskNotFound               = errors.New("task not found")
	ErrTMDAllMilestonesNotDone       = errors.New("all milestones must be approved before approving the quest")
	ErrTMDInvalidMilestoneStatus     = errors.New("milestone is not in an approved state to cancel")
	ErrTMDQuestNotCompleted          = errors.New("quest is not completed; cannot cancel approval")
	ErrTMDInvalidRating              = errors.New("rating must be between 1 and 5")
	ErrTMDReasonRequired             = errors.New("reason is mandatory and must be at least 5 characters")
	ErrTMDInvalidStatus              = errors.New("invalid task status; allowed values: todo, in_progress, completed")
	ErrTMDInvalidAssignee            = errors.New("assignee_id must be a valid user id")
	ErrTMDHoldInvariant              = errors.New("wallet hold invariant violated")
	ErrTMDQuestAlreadyCompleted      = errors.New("quest is already completed")
	ErrTMDQuestFundsAlreadyWithdrawn = errors.New("cannot reopen quest: freelancer has already withdrawn released funds")
	ErrTMDDeprecated                 = errors.New("this Task Management endpoint is deprecated; use POST /v1/quests/{id}/milestones for milestone creation and POST /v1/quests/{id}/milestones/{milestone_id}/release for release")
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
	repo       repository.TaskManagementRepository
	partyRepo  *repository.PartyRepository
	assignRepo repository.MilestoneAssignmentRepository
}

func NewTaskManagementService() TaskManagementService {
	return &taskManagementService{
		repo:       repository.NewTaskManagementRepository(),
		partyRepo:  repository.NewPartyRepository(),
		assignRepo: repository.NewMilestoneAssignmentRepository(),
	}
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

// 2. GetActiveQuests
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
			Budget:     utils.PaiseToRupees(q.BudgetMinor),
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
		case models.MilestoneStatusReleased:
			completedMilestones++
		case models.MilestoneStatusWorkInProgress, models.MilestoneStatusSubmittedForRelease:
			inProgressMilestones++
		case models.MilestoneStatusDraft,
			models.MilestoneStatusAwaitingClientReview,
			models.MilestoneStatusRevisionRequested,
			models.MilestoneStatusRevisionInProgress,
			models.MilestoneStatusAgreed:
			// Negotiation stages: not proposed yet, awaiting review, or being revised.
			pendingMilestones++
		default:
			// funding_pending / funded (money in, work not started) plus
			// disputed / cancelled / refund_* all read as "pending" to the client.
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
			Budget:          utils.PaiseToRupees(quest.BudgetMinor),
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
		totalAmount += utils.PaiseToRupees(m.AmountMinor)
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
			Amount:      utils.PaiseToRupees(m.AmountMinor),
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

// 5. AddMilestone — DEPRECATED.
// Clients must not author milestones: creation belongs to the assigned freelancer via
// POST /v1/quests/{id}/milestones so the review/agreement/funding flow is always followed.
func (s *taskManagementService) AddMilestone(clientIDStr, questIDStr string, req dto.AddMilestoneRequestBody) (*dto.AddMilestoneResponse, error) {
	return nil, ErrTMDDeprecated
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

	// Only pre-funding states may be deleted: from `agreed` onwards the milestone is
	// part of the negotiation/funding/release lifecycle.
	if milestone.Status != models.MilestoneStatusDraft &&
		milestone.Status != models.MilestoneStatusRevisionRequested {
		return nil, ErrTMDInvalidMilestoneStatus
	}

	if err := s.repo.DeleteMilestone(milestone.ID, quest.ID); err != nil {
		return nil, fmt.Errorf("failed to delete milestone: %w", err)
	}

	_ = s.repo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		ActorID:     clientUUID,
		ActionType:  models.ActivityMilestoneDeleted,
		Title:       fmt.Sprintf("Milestone '%s' Deleted", milestone.Title),
		Description: "Client removed milestone and its tasks",
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

// 8. ApproveMilestone — DEPRECATED.
// Setting a milestone to `released` without moving money was the C1 bypass; the only
// path to `released` is now MilestoneReleaseService.ReleaseMilestone.
func (s *taskManagementService) ApproveMilestone(clientIDStr, milestoneIDStr string, feedback string) (*dto.ApproveMilestoneTMDResponse, error) {
	return nil, ErrTMDDeprecated
}

// 9. CancelMilestoneApproval — DEPRECATED.
// A released milestone has already credited the freelancer wallet, so it cannot be
// un-released from here; disputes use the refund flow instead.
func (s *taskManagementService) CancelMilestoneApproval(clientIDStr, milestoneIDStr string, reason string) (*dto.CancelMilestoneApprovalTMDResponse, error) {
	return nil, ErrTMDDeprecated
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
		if m.Status != models.MilestoneStatusReleased {
			return nil, ErrTMDAllMilestonesNotDone
		}
	}

	// Status alone is not proof of payment: require the money to have actually moved
	// (MilestonePayment released + EscrowTransaction released) for every milestone.
	if err := s.repo.VerifyQuestFullySettled(quest.ID); err != nil {
		return nil, ErrTMDAllMilestonesNotDone
	}

	now := time.Now()
	var (
		unlockedToAvailable int64
		availableAfter      int64
		pendingAfter        int64
	)

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		repo := repository.NewTaskManagementRepositoryWithDB(tx)

		// Conditional transition: the first caller wins, a retry sees 0 rows.
		res := tx.Model(&models.Quest{}).
			Where("id = ? AND status = ?", quest.ID, models.QuestStatusInProgress).
			Update("status", models.QuestStatusCompleted)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrTMDQuestAlreadyCompleted
		}

		if err := repo.CreateQuestReview(&models.QuestReview{
			QuestID:  quest.ID,
			ClientID: clientUUID,
			Rating:   req.Rating,
			Feedback: feedback,
		}); err != nil {
			return fmt.Errorf("create quest review: %w", err)
		}

		if err := repo.CreateActivity(&models.QuestActivity{
			QuestID:     quest.ID,
			ActorID:     clientUUID,
			ActionType:  models.ActivityQuestApproved,
			Title:       fmt.Sprintf("Quest '%s' Completed & Approved", quest.Title),
			Description: fmt.Sprintf("Client gave %d stars: %s", req.Rating, feedback),
		}); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}

		total, err := repository.NewMilestonePaymentRepositoryTx(tx).SumReleasedNetByQuest(quest.ID)
		if err != nil {
			return fmt.Errorf("sum released milestone payments: %w", err)
		}
		if total == 0 || quest.AssignedFreelancerID == nil {
			return nil
		}
		unlockedToAvailable = total

		walletRepo := repository.NewWalletRepositoryTx(tx)
		w, err := walletRepo.GetByUserIDForUpdate(*quest.AssignedFreelancerID)
		if err != nil || w.PendingBalanceMinor < total {
			return ErrTMDHoldInvariant
		}

		// Cycle-aware key: a reopened and re-approved quest must not collide.
		var priorReleases int64
		if err := tx.Model(&models.WalletTransaction{}).
			Where("reference = ? AND type = ?", quest.ID.String(), models.WalletTxTypeHoldRelease).
			Count(&priorReleases).Error; err != nil {
			return err
		}

		w.PendingBalanceMinor -= total
		w.AvailableBalanceMinor += total
		if err := walletRepo.Save(w); err != nil {
			return fmt.Errorf("unlock wallet hold: %w", err)
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"pending_balance_after_minor": w.PendingBalanceMinor,
			"quest_id":                    quest.ID.String(),
		})
		if err := tx.Create(&models.WalletTransaction{
			WalletID:          w.ID,
			UserID:            *quest.AssignedFreelancerID,
			Type:              models.WalletTxTypeHoldRelease,
			AmountMinor:       total,
			BalanceAfterMinor: w.AvailableBalanceMinor,
			Currency:          w.Currency,
			Reference:         quest.ID.String(),
			Description:       "Quest completed: pending earnings released",
			IdempotencyKey:    fmt.Sprintf("quest:%s:hold_release:%d", quest.ID, priorReleases+1),
			Metadata:          datatypes.JSON(meta),
		}).Error; err != nil {
			return fmt.Errorf("create hold release tx: %w", err)
		}

		availableAfter = w.AvailableBalanceMinor
		pendingAfter = w.PendingBalanceMinor
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTMDQuestAlreadyCompleted) {
			// Idempotent replay: nothing moved, report the current state.
			return &dto.ApproveQuestTMDResponse{
				Message:     "Quest already approved and marked as completed",
				QuestID:     quest.ID.String(),
				Status:      string(models.QuestStatusCompleted),
				CompletedAt: now,
			}, nil
		}
		return nil, err
	}

	return &dto.ApproveQuestTMDResponse{
		Message:                     "Quest approved and marked as completed",
		QuestID:                     quest.ID.String(),
		Status:                      string(models.QuestStatusCompleted),
		CompletedAt:                 now,
		UnlockedToAvailableMinor:    unlockedToAvailable,
		UnlockedToAvailable:         utils.PaiseToRupees(unlockedToAvailable),
		WalletAvailableBalanceMinor: availableAfter,
		WalletPendingBalanceMinor:   pendingAfter,
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

	var (
		relockedToPending int64
		availableAfter    int64
		pendingAfter      int64
	)

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		repo := repository.NewTaskManagementRepositoryWithDB(tx)

		res := tx.Model(&models.Quest{}).
			Where("id = ? AND status = ?", quest.ID, models.QuestStatusCompleted).
			Update("status", models.QuestStatusInProgress)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrTMDQuestNotCompleted
		}

		total, err := repository.NewMilestonePaymentRepositoryTx(tx).SumReleasedNetByQuest(quest.ID)
		if err != nil {
			return fmt.Errorf("sum released milestone payments: %w", err)
		}

		if total > 0 && quest.AssignedFreelancerID != nil {
			walletRepo := repository.NewWalletRepositoryTx(tx)
			w, err := walletRepo.GetByUserIDForUpdate(*quest.AssignedFreelancerID)
			if err != nil {
				return ErrTMDHoldInvariant
			}
			// Any withdrawal at all makes available < total, so the reopen is refused.
			if w.AvailableBalanceMinor < total {
				return ErrTMDQuestFundsAlreadyWithdrawn
			}

			var priorHolds int64
			if err := tx.Model(&models.WalletTransaction{}).
				Where("reference = ? AND type = ?", quest.ID.String(), models.WalletTxTypeHold).
				Count(&priorHolds).Error; err != nil {
				return err
			}

			w.AvailableBalanceMinor -= total
			w.PendingBalanceMinor += total
			if err := walletRepo.Save(w); err != nil {
				return fmt.Errorf("relock wallet hold: %w", err)
			}

			meta, _ := json.Marshal(map[string]interface{}{
				"pending_balance_after_minor": w.PendingBalanceMinor,
				"quest_id":                    quest.ID.String(),
			})
			if err := tx.Create(&models.WalletTransaction{
				WalletID:          w.ID,
				UserID:            *quest.AssignedFreelancerID,
				Type:              models.WalletTxTypeHold,
				AmountMinor:       -total,
				BalanceAfterMinor: w.AvailableBalanceMinor,
				Currency:          w.Currency,
				Reference:         quest.ID.String(),
				Description:       "Quest reopened: released earnings returned to pending",
				IdempotencyKey:    fmt.Sprintf("quest:%s:hold_relock:%d", quest.ID, priorHolds+1),
				Metadata:          datatypes.JSON(meta),
			}).Error; err != nil {
				return fmt.Errorf("create hold tx: %w", err)
			}

			relockedToPending = total
			availableAfter = w.AvailableBalanceMinor
			pendingAfter = w.PendingBalanceMinor
		}

		if err := repo.CreateActivity(&models.QuestActivity{
			QuestID:     quest.ID,
			ActorID:     clientUUID,
			ActionType:  models.ActivityQuestApprovalCancelled,
			Title:       fmt.Sprintf("Quest Approval Cancelled for '%s'", quest.Title),
			Description: fmt.Sprintf("Reason: %s", reason),
		}); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.CancelQuestApprovalTMDResponse{
		Message:                     "Quest approval cancelled and reopened",
		QuestID:                     quest.ID.String(),
		Status:                      string(models.QuestStatusInProgress),
		RelockedToPendingMinor:      relockedToPending,
		RelockedToPending:           utils.PaiseToRupees(relockedToPending),
		WalletAvailableBalanceMinor: availableAfter,
		WalletPendingBalanceMinor:   pendingAfter,
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

	if err := s.authorizeTaskWrite(quest, milestone, clientUUID); err != nil {
		return nil, err
	}

	// Task planning is only allowed before the milestone is submitted for release.
	switch milestone.Status {
	case models.MilestoneStatusDraft,
		models.MilestoneStatusRevisionInProgress,
		models.MilestoneStatusWorkInProgress:
	default:
		return nil, ErrTMDInvalidMilestoneStatus
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
	if req.AssigneeID != nil && strings.TrimSpace(*req.AssigneeID) != "" {
		parsed, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			return nil, ErrTMDInvalidAssignee
		}
		assigneeUUID = &parsed
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

	assigneeName := ""
	if assigneeUUID != nil {
		// GetTaskByID preloads Assignee, so one re-read yields the real name.
		if created, err := s.repo.GetTaskByID(task.ID); err == nil && created != nil && created.Assignee != nil {
			assigneeName = strings.TrimSpace(created.Assignee.FirstName + " " + created.Assignee.LastName)
		}
	}

	return &dto.MilestoneTaskItem{
		SlNo:        1,
		ID:          task.ID.String(),
		Name:        task.Title,
		Description: task.Description,
		Assignee:    assigneeName,
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

	// Only the quest client or the assigned freelancer may move its tasks.
	quest, err := s.repo.GetQuestByID(task.QuestID)
	if err != nil || quest == nil {
		return nil, ErrTMDQuestNotFound
	}
	// Tasks are only editable before the milestone is submitted for release.
	ms, err := s.repo.GetMilestoneByID(task.MilestoneID)
	if err != nil || ms == nil {
		return nil, ErrTMDMilestoneNotFound
	}
	if err := s.authorizeTaskWrite(quest, ms, clientUUID); err != nil {
		return nil, err
	}
	switch ms.Status {
	case models.MilestoneStatusDraft,
		models.MilestoneStatusRevisionInProgress,
		models.MilestoneStatusWorkInProgress:
	default:
		return nil, ErrTMDInvalidMilestoneStatus
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

	assigneeName := ""
	if task.Assignee != nil {
		assigneeName = strings.TrimSpace(task.Assignee.FirstName + " " + task.Assignee.LastName)
	}

	return &dto.MilestoneTaskItem{
		SlNo:        1,
		ID:          task.ID.String(),
		Name:        task.Title,
		Description: task.Description,
		Assignee:    assigneeName,
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
		if m.Status == models.MilestoneStatusReleased {
			completed++
		}
	}
	return int(math.Round((float64(completed) / float64(len(milestones))) * 100))
}

// Helper: Calculate individual milestone progress
func calculateMilestoneProgress(m models.QuestMilestone) int {
	if m.Status == models.MilestoneStatusReleased {
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
	case models.MilestoneStatusSubmittedForRelease:
		return 90
	case models.MilestoneStatusWorkInProgress:
		return 25
	default:
		return 0
	}
}

// authorizeTaskWrite lets the client write tasks unconditionally, the assigned
// freelancer (team lead) unconditionally, and an active party member only on
// milestones assigned to them.
func (s *taskManagementService) authorizeTaskWrite(quest *models.Quest, milestone *models.QuestMilestone, userID uuid.UUID) error {
	if quest.ClientID == userID {
		return nil
	}
	if quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userID {
		return nil
	}
	if s.partyRepo == nil || s.assignRepo == nil || quest.AcceptedPartyID == nil {
		return ErrTMDUnauthorized
	}
	member, err := s.partyRepo.FindMember(*quest.AcceptedPartyID, userID)
	if err != nil || member.Status != models.MemberActive {
		return ErrTMDUnauthorized
	}
	assigned, err := s.assignRepo.ExistsForMilestoneAndMember(milestone.ID, userID)
	if err != nil {
		return err
	}
	if !assigned {
		return ErrTMDUnauthorized
	}
	return nil
}
