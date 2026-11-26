package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

var (
	ErrOnlyClientAllowed     = errors.New("only client accounts can create quests")
	ErrClientNotVerified     = errors.New("client account verification required before creating quests")
	ErrInvalidUserID         = errors.New("invalid user id")
	ErrInvalidQuestID        = errors.New("invalid quest id")
	ErrQuestNotFound         = errors.New("quest not found")
	ErrQuestNotVisible       = errors.New("quest is not visible for your rank")
	ErrQuestNotDraft         = errors.New("only draft quests can be published")
	ErrClientFundingRequired = errors.New("client must configure a funding method before publishing a quest")
	ErrOnlyPartyLeadAllowed  = errors.New("only Party Lead of an active, verified party can accept quests")
	ErrPartyNotVerified      = errors.New("party must be verified to accept quests")
	ErrOnlyClientCanCancel   = errors.New("only the client owner can cancel/reject this quest")
	ErrQuestAlreadyFinished  = errors.New("quest is already completed or cancelled")
	ErrQuestLegacyEscrow     = errors.New("quest uses legacy quest-level escrow; cancellation requires manual reconciliation")
	ErrQuestNotOwner         = errors.New("only the quest owner can perform this action")
	ErrQuestNotEditable      = errors.New("only draft quests can be edited")
)

type QuestService interface {
	CreateQuestDraft(userID string, req dto.CreateQuestBody) (*dto.CreateQuestResponse, error)
	PublishQuest(userID string, questID string, req dto.PublishQuestBody) (*dto.PublishQuestResponse, error)
	ListQuests(userID string, req dto.ListQuestsInput) (*dto.ListQuestsResponse, error)
	GetQuestDetails(userID string, questID string) (*dto.QuestDetailsResponse, error)
	RejectQuest(userID string, questID string, req dto.RejectQuestBody) (*dto.RejectQuestResponse, error)
	UpdateQuest(userID string, questID string, req dto.UpdateQuestBody) (*dto.UpdateQuestResponse, error)
	SaveQuest(userID string, questID string) (*dto.SaveQuestResponse, error)
	ListSavedQuests(userID string) (*dto.ListQuestsResponse, error)
	GetBoardSummary(userID string) (*dto.QuestBoardSummaryResponse, error)
}

type questService struct {
	questRepo     repository.QuestRepository
	userRepo      repository.UserRepository
	partyRepo     *repository.PartyRepository
	milestoneRepo repository.QuestMilestoneRepository
	appRepo       repository.QuestApplicationRepository
	refundSvc     *RefundService
}

func NewQuestService() QuestService {
	return &questService{
		questRepo:     repository.NewQuestRepository(),
		userRepo:      repository.NewUserRepository(),
		partyRepo:     repository.NewPartyRepository(),
		milestoneRepo: repository.NewQuestMilestoneRepository(),
		appRepo:       repository.NewQuestApplicationRepository(),
	}
}

// NewQuestServiceWithRefund wires refunds into RejectQuest (cancel-with-refund).
// Constructors without a RefundService still cancel, but refuse to guess when a
// refundable milestone exists.
func NewQuestServiceWithRefund(refundSvc *RefundService) QuestService {
	svc := NewQuestService().(*questService)
	svc.refundSvc = refundSvc
	return svc
}

func NewQuestServiceWithRepos(
	questRepo repository.QuestRepository,
	userRepo repository.UserRepository,
	partyRepo *repository.PartyRepository,
) QuestService {
	return &questService{
		questRepo:     questRepo,
		userRepo:      userRepo,
		partyRepo:     partyRepo,
		milestoneRepo: repository.NewQuestMilestoneRepository(),
		appRepo:       repository.NewQuestApplicationRepository(),
	}
}

// ============================================================
// 1. CreateQuestDraft — creates a draft quest (NO milestones)
// ============================================================
// Freelancer will create milestones AFTER being accepted via the
// application flow. Quest has NO milestones at creation time.
func (s *questService) CreateQuestDraft(userID string, req dto.CreateQuestBody) (*dto.CreateQuestResponse, error) {
	clientUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	if user.AccountType == nil {
		return nil, ErrOnlyClientAllowed
	}

	accType := string(*user.AccountType)
	if accType != string(models.AccountTypeClientAdmin) &&
		accType != string(models.AccountTypeClient) &&
		accType != string(models.AccountTypeClientOwner) &&
		accType != string(models.AccountTypeClientMember) {
		return nil, ErrOnlyClientAllowed
	}

	isVerified := false
	if user.Status == models.StatusActive {
		isVerified = true
	} else {
		vRecord, err := s.userRepo.GetVerificationRecordByUserID(userID)
		if err == nil && vRecord != nil && vRecord.Status == models.VerificationApproved {
			isVerified = true
		}
	}
	if !isVerified {
		return nil, ErrClientNotVerified
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 3 || len(title) > 100 {
		return nil, errors.New("quest title must be between 3 and 100 characters")
	}

	description := strings.TrimSpace(req.Description)
	if len(description) < 10 {
		return nil, errors.New("quest description must be at least 10 characters")
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, errors.New("quest category is required")
	}

	if req.BudgetMinor < 100000 {
		return nil, errors.New("quest budget must be at least ₹1,000 (100000 paise)")
	}

	if req.TimelineDays < 1 {
		return nil, errors.New("quest timeline must be at least 1 day")
	}

	if len(req.Skills) == 0 {
		return nil, errors.New("at least one technical skill is required")
	}

	// Budget is reference-only at draft time
	budgetRupees := float64(req.BudgetMinor) / 100.0

	clientRank := user.Rank
	if clientRank == "" {
		clientRank = models.RankF
	}
	if err := utils.ValidateBudgetForRank(clientRank, budgetRupees); err != nil {
		return nil, err
	}

	requiredMinRank, err := utils.ComputeRequiredMinRank(budgetRupees)
	if err != nil {
		return nil, err
	}

	complexity := models.QuestComplexity(req.Complexity)
	if complexity == "" {
		complexity = models.QuestComplexityIntermediate
	}

	questType := models.QuestType(req.QuestType)
	if questType == "" {
		questType = models.QuestTypeStandard
	}

	quest := &models.Quest{
		ClientID:        clientUUID,
		Title:           title,
		Description:     description,
		Category:        category,
		Industry:        strings.TrimSpace(req.Industry),
		Complexity:      complexity,
		BudgetMinor:     req.BudgetMinor,
		TimelineDays:    req.TimelineDays,
		QuestType:       questType,
		RequiredMinRank: requiredMinRank,
		ExperienceLevel: strings.TrimSpace(req.ExperienceLevel),
		SoftSkills:      pq.StringArray(req.SoftSkills),
		Status:          models.QuestStatusDraft,
	}

	if len(req.Languages) > 0 {
		langBytes, err := json.Marshal(req.Languages)
		if err == nil {
			quest.Languages = datatypes.JSON(langBytes)
		}
	}

	// Build Skills
	skills := make([]models.QuestSkill, len(req.Skills))
	for i, s := range req.Skills {
		prof := strings.TrimSpace(s.ProficiencyLevel)
		if prof == "" {
			prof = "intermediate"
		}
		skills[i] = models.QuestSkill{
			Name:             strings.TrimSpace(s.Name),
			ProficiencyLevel: prof,
			IsCore:           s.IsCore,
		}
	}

	// Milestones are NOT created here.
	// Freelancer creates them AFTER being accepted via the application flow.
	if err := s.questRepo.CreateQuestWithSkills(quest, skills); err != nil {
		return nil, fmt.Errorf("failed to save quest draft: %w", err)
	}

	return &dto.CreateQuestResponse{
		QuestID:         quest.ID.String(),
		Title:           quest.Title,
		Status:          string(quest.Status),
		BudgetMinor:     quest.BudgetMinor,
		RequiredMinRank: quest.RequiredMinRank,
		CreatedAt:       quest.CreatedAt,
	}, nil
}

// ============================================================
// 2. PublishQuest — draft → open (escrow deposit)
// ============================================================
func (s *questService) PublishQuest(userID string, questID string, req dto.PublishQuestBody) (*dto.PublishQuestResponse, error) {
	clientUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, errors.New("only the client owner can publish this quest")
	}

	if quest.Status != models.QuestStatusDraft {
		return nil, ErrQuestNotDraft
	}

	// Publishing has no payment prerequisite: escrow is funded per-milestone by
	// MilestoneFundingService.FundMilestone once a milestone reaches `agreed`.
	now := time.Now()
	quest.Status = models.QuestStatusOpen
	quest.PublishedAt = &now

	if err := s.questRepo.UpdateQuest(quest); err != nil {
		return nil, fmt.Errorf("failed to publish quest: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"budget_minor":      quest.BudgetMinor,
		"required_min_rank": quest.RequiredMinRank,
		"published_at":      now,
	})
	_ = s.questRepo.CreateAuditLog(&models.AuditLog{
		UserID:   &clientUUID,
		Action:   "quest_published",
		Entity:   "quest",
		EntityID: quest.ID.String(),
		Metadata: datatypes.JSON(meta),
	})

	return &dto.PublishQuestResponse{
		QuestID:        quest.ID.String(),
		Title:          quest.Title,
		Status:         string(quest.Status),
		PublishedAt:    now,
		IsEscrowFunded: quest.IsEscrowFunded,
		Message:        "Quest published successfully and open for applications",
	}, nil
}

// ============================================================
// 3. ListQuests — tier-locked asymmetric visibility
// ============================================================
func (s *questService) ListQuests(userID string, req dto.ListQuestsInput) (*dto.ListQuestsResponse, error) {
	requesterRank := models.RankF

	var user *models.User
	if userID != "" {
		if u, err := s.userRepo.GetUserByID(userID); err == nil && u != nil {
			user = u
			if u.Rank != "" {
				requesterRank = u.Rank
			}
		}
	}

	var minBudgetPtr, maxBudgetPtr *float64
	if req.MinBudget > 0 {
		minBudgetPtr = &req.MinBudget
	}
	if req.MaxBudget > 0 {
		maxBudgetPtr = &req.MaxBudget
	}

	tlMin, tlMax := timelineBounds(req.TimelineBucket)

	filter := repository.QuestFilter{
		Category:    req.Category,
		Complexity:  req.Complexity,
		MinBudget:   minBudgetPtr,
		MaxBudget:   maxBudgetPtr,
		Search:      strings.TrimSpace(req.Search),
		QuestType:   strings.TrimSpace(req.QuestType),
		Skills:      splitCSV(req.Skills),
		ClientRank:  strings.TrimSpace(req.ClientRank),
		TimelineMin: tlMin,
		TimelineMax: tlMax,
		Sort:        req.Sort,
		Cursor:      req.Cursor,
		Limit:       req.Limit,
	}

	// Three audiences:
	//   admin      → all quests, no client scoping (explicit filters still apply)
	//   client     → own quests only, all statuses
	//   freelancer → rank-filtered marketplace, open/published by default
	accountType := ""
	if user != nil && user.AccountType != nil {
		accountType = string(*user.AccountType)
	}

	switch {
	case accountType == string(models.AccountTypeAdmin):
		if req.Status != "" {
			filter.Status = req.Status
		}
	case strings.HasPrefix(accountType, "client"):
		clientUUID := user.ID
		filter.ClientID = &clientUUID
		if req.Status != "" {
			filter.Status = req.Status
		}
	default:
		// Freelancer marketplace view; also covers anonymous / account-type-less callers.
		filter.AllowedRanks = utils.GetVisibleRanks(requesterRank)
		if req.Status != "" {
			filter.Status = req.Status
		} else {
			filter.StatusIn = []string{
				string(models.QuestStatusOpen),
				string(models.QuestStatusPublished),
			}
		}
	}

	quests, nextCursor, hasMore, total, err := s.questRepo.ListQuests(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query quests: %w", err)
	}

	ids := questIDs(quests)
	applicantCounts, err := s.questRepo.CountApplicantsByQuestIDs(ids)
	if err != nil {
		applicantCounts = map[uuid.UUID]int64{}
	}
	savedIDs, userSkills := s.viewerState(user, ids)

	items := make([]dto.QuestListItem, len(quests))
	for i, q := range quests {
		items[i] = s.questListItem(q, userSkills, savedIDs, applicantCounts)
	}

	return &dto.ListQuestsResponse{
		Quests:     items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      int(total),
	}, nil
}

// ============================================================
// 4. GetQuestDetails — FULL FIX for visibility bug
// ============================================================
// Previous bug: AcceptedPartyID != nil condition meant direct-flow
// (individual) accepted quests were NOT visible to the assigned freelancer.
//
// Fix:
//   - Owner/Admin → always visible
//   - Open/Published → rank sufficient → visible
//   - InProgress/Completed/Cancelled → visible to:
//   - assigned freelancer (AssignedFreelancerID == requester)
//   - active party member (if AcceptedPartyID != nil)
//   - Anything else → 403
func (s *questService) GetQuestDetails(userID string, questID string) (*dto.QuestDetailsResponse, error) {
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}

	var requesterUUID uuid.UUID
	var user *models.User
	requesterRank := models.RankF

	if userID != "" {
		if uUUID, err := uuid.Parse(userID); err == nil {
			requesterUUID = uUUID
		}
		if u, err := s.userRepo.GetUserByID(userID); err == nil && u != nil {
			user = u
			if u.Rank != "" {
				requesterRank = u.Rank
			}
		}
	}

	isOwner := quest.ClientID == requesterUUID
	isAdmin := user != nil && user.AccountType != nil &&
		string(*user.AccountType) == string(models.AccountTypeAdmin)

	if !isOwner && !isAdmin {
		isOpenish := quest.Status == models.QuestStatusOpen ||
			quest.Status == models.QuestStatusPublished

		if isOpenish {
			// Marketplace visibility — rank gate
			if !utils.IsRankSufficient(requesterRank, quest.RequiredMinRank) {
				return nil, fmt.Errorf(
					"quest requires rank %s which is higher than your rank %s",
					quest.RequiredMinRank, requesterRank,
				)
			}
		} else if quest.Status == models.QuestStatusInProgress ||
			quest.Status == models.QuestStatusCompleted ||
			quest.Status == models.QuestStatusCancelled {

			// Post-acceptance visibility — only assigned entities can see
			isAssigned := quest.AssignedFreelancerID != nil &&
				*quest.AssignedFreelancerID == requesterUUID

			isMember := false
			if quest.AcceptedPartyID != nil {
				if members, err := s.partyRepo.GetMembers(*quest.AcceptedPartyID); err == nil {
					for _, m := range members {
						if m.UserID == requesterUUID && m.Status == models.MemberActive {
							isMember = true
							break
						}
					}
				}
			}

			if !isAssigned && !isMember {
				return nil, ErrQuestNotVisible
			}
		} else {
			// Draft or unknown status — only owner/admin (already handled above)
			return nil, ErrQuestNotVisible
		}
	}

	// --- Build response ---

	if !isOwner {
		if err := s.questRepo.IncrementQuestViews(quest.ID); err == nil {
			quest.ViewsCount++
		}
	}

	applicantCounts, err := s.questRepo.CountApplicantsByQuestIDs([]uuid.UUID{quest.ID})
	if err != nil {
		applicantCounts = map[uuid.UUID]int64{}
	}
	savedIDs, userSkills := s.viewerState(user, []uuid.UUID{quest.ID})

	milestones := make([]dto.QuestDetailMilestone, len(quest.Milestones))
	for i, m := range quest.Milestones {
		milestones[i] = dto.QuestDetailMilestone{
			ID:          m.ID.String(),
			Title:       m.Title,
			Description: m.Description,
			Amount:      utils.PaiseToRupees(m.AmountMinor),
			DueDays:     m.DueDays,
			OrderIndex:  m.OrderIndex,
			Status:      string(m.Status),
		}
	}

	skills := make([]dto.QuestDetailSkill, len(quest.Skills))
	for i, sk := range quest.Skills {
		skills[i] = dto.QuestDetailSkill{
			ID:               sk.ID.String(),
			Name:             sk.Name,
			ProficiencyLevel: sk.ProficiencyLevel,
			IsCore:           sk.IsCore,
		}
	}

	clientName := "Anonymous Client"
	if quest.Client.FirstName != "" {
		clientName = strings.TrimSpace(quest.Client.FirstName + " " + quest.Client.LastName)
	}

	var acceptedPartyIDStr *string
	if quest.AcceptedPartyID != nil {
		s := quest.AcceptedPartyID.String()
		acceptedPartyIDStr = &s
	}

	return &dto.QuestDetailsResponse{
		ID:              quest.ID.String(),
		ClientID:        quest.ClientID.String(),
		ClientName:      clientName,
		ClientRank:      quest.Client.Rank,
		Title:           quest.Title,
		Description:     quest.Description,
		Category:        quest.Category,
		Industry:        quest.Industry,
		Complexity:      string(quest.Complexity),
		BudgetINR:       utils.PaiseToRupees(quest.BudgetMinor),
		TimelineDays:    quest.TimelineDays,
		QuestType:       string(quest.QuestType),
		RequiredMinRank: quest.RequiredMinRank,
		ExperienceLevel: quest.ExperienceLevel,
		SoftSkills:      []string(quest.SoftSkills),
		Status:          string(quest.Status),
		PublishedAt:     quest.PublishedAt,
		IsEscrowFunded:  quest.IsEscrowFunded,
		AcceptedPartyID: acceptedPartyIDStr,
		Milestones:      milestones,
		Skills:          skills,
		ApplicantsCount: int(applicantCounts[quest.ID]),
		ViewsCount:      quest.ViewsCount,
		IsSaved:         savedIDs[quest.ID],
		MatchPercentage: matchPercentage(skillNamesOf(quest.Skills), userSkills),
		CreatedAt:       quest.CreatedAt,
		UpdatedAt:       quest.UpdatedAt,
	}, nil
}

// ============================================================
// 5. RejectQuest — client cancels/rejects (refunds held escrow)
// ============================================================
// Allowed on: open, published, in_progress
// Not allowed on: cancelled, completed
// Every milestone in a refundable state (work_in_progress,
// submitted_for_release) is refunded through RefundService; the quest only
// becomes cancelled once each of those succeeded.
func (s *questService) RejectQuest(userID string, questID string, req dto.RejectQuestBody) (*dto.RejectQuestResponse, error) {
	clientUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}

	if quest.ClientID != clientUUID {
		return nil, ErrOnlyClientCanCancel
	}

	if quest.Status == models.QuestStatusCancelled ||
		quest.Status == models.QuestStatusCompleted {
		return nil, ErrQuestAlreadyFinished
	}

	// Legacy quest-level escrow has no per-milestone rows to walk; cancelling
	// would strand the deposit, so hand it to manual reconciliation instead.
	if quest.IsEscrowFunded {
		return nil, ErrQuestLegacyEscrow
	}

	reason := strings.TrimSpace(req.Reason)
	if len(reason) < 5 {
		reason = "Quest cancelled by client"
	}

	// Refund sequentially, oldest milestone first. Keys are deterministic per
	// milestone, so a retried cancel replays the existing refund instead of
	// refunding twice.
	milestones, err := s.milestoneRepo.FindByQuestID(qUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load milestones: %w", err)
	}

	var refundedMinor int64
	var refundedCount int
	for i := range milestones {
		m := &milestones[i]
		if m.Status != models.MilestoneStatusWorkInProgress &&
			m.Status != models.MilestoneStatusSubmittedForRelease {
			continue
		}
		if s.refundSvc == nil {
			return nil, errors.New("refund service is not configured")
		}
		res, err := s.refundSvc.RequestRefund(userID, questID, m.ID.String(), dto.RequestRefundBody{
			Reason:         reason,
			IdempotencyKey: fmt.Sprintf("quest:%s:cancel:%s", questID, m.ID),
		})
		if err != nil {
			if skippableCancelRefundError(err) {
				continue
			}
			return nil, fmt.Errorf("refund failed for milestone %s: %w", m.ID, err)
		}
		refundedCount++
		refundedMinor += res.RefundAmountMinor
	}

	quest.Status = models.QuestStatusCancelled
	if err := s.questRepo.UpdateQuest(quest); err != nil {
		return nil, fmt.Errorf("failed to cancel quest: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"reason": req.Reason,
	})
	_ = s.questRepo.CreateAuditLog(&models.AuditLog{
		UserID:   &clientUUID,
		Action:   "quest_cancelled",
		Entity:   "quest",
		EntityID: quest.ID.String(),
		Metadata: datatypes.JSON(meta),
	})

	message := "Quest has been cancelled successfully"
	if refundedCount > 0 {
		message = fmt.Sprintf("Quest cancelled and %d milestone refund(s) initiated", refundedCount)
	}

	return &dto.RejectQuestResponse{
		QuestID:            quest.ID.String(),
		Status:             string(quest.Status),
		EscrowRefunded:     refundedCount > 0,
		EscrowRefundAmount: utils.PaiseToRupees(refundedMinor),
		Message:            message,
	}, nil
}

// skippableCancelRefundError reports refund failures that mean "this milestone
// has nothing to refund during a bulk cancel": a refund already in flight or
// done, escrow never held (e.g. only failed funding attempts), or the milestone
// left a refundable state concurrently.
func skippableCancelRefundError(err error) bool {
	return errors.Is(err, ErrRefStateRace) ||
		errors.Is(err, ErrRefAlreadyRefunded) ||
		errors.Is(err, ErrRefInvalidState) ||
		errors.Is(err, ErrRefEscrowNotFound) ||
		errors.Is(err, ErrRefPaymentNotFound) ||
		errors.Is(err, ErrRefEscrowNotHeld)
}

// ============================================================
// 6. UpdateQuest — partial edit of a draft quest (owner only)
// ============================================================
func (s *questService) UpdateQuest(userID string, questID string, req dto.UpdateQuestBody) (*dto.UpdateQuestResponse, error) {
	clientUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}
	if quest.ClientID != clientUUID {
		return nil, ErrQuestNotOwner
	}
	if quest.Status != models.QuestStatusDraft {
		return nil, ErrQuestNotEditable
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if len(title) < 3 || len(title) > 100 {
			return nil, errors.New("quest title must be between 3 and 100 characters")
		}
		quest.Title = title
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if len(description) < 10 {
			return nil, errors.New("quest description must be at least 10 characters")
		}
		quest.Description = description
	}
	if req.Category != nil {
		category := strings.TrimSpace(*req.Category)
		if category == "" {
			return nil, errors.New("quest category is required")
		}
		quest.Category = category
	}
	if req.Industry != nil {
		quest.Industry = strings.TrimSpace(*req.Industry)
	}
	if req.Complexity != nil {
		if v := strings.TrimSpace(*req.Complexity); v != "" {
			quest.Complexity = models.QuestComplexity(v)
		}
	}
	if req.BudgetMinor != nil {
		if *req.BudgetMinor < 100000 {
			return nil, errors.New("quest budget must be at least ₹1,000 (100000 paise)")
		}
		quest.BudgetMinor = *req.BudgetMinor
	}
	if req.TimelineDays != nil {
		if *req.TimelineDays < 1 {
			return nil, errors.New("quest timeline must be at least 1 day")
		}
		quest.TimelineDays = *req.TimelineDays
	}
	if req.QuestType != nil {
		if v := strings.TrimSpace(*req.QuestType); v != "" {
			quest.QuestType = models.QuestType(v)
		}
	}
	if req.ExperienceLevel != nil {
		quest.ExperienceLevel = strings.TrimSpace(*req.ExperienceLevel)
	}
	if req.SoftSkills != nil {
		quest.SoftSkills = pq.StringArray(*req.SoftSkills)
	}
	if req.Languages != nil {
		if langBytes, err := json.Marshal(*req.Languages); err == nil {
			quest.Languages = datatypes.JSON(langBytes)
		}
	}

	// A budget change moves the marketplace rank gate with it.
	budgetRupees := float64(quest.BudgetMinor) / 100.0
	clientRank := quest.Client.Rank
	if clientRank == "" {
		clientRank = models.RankF
	}
	if err := utils.ValidateBudgetForRank(clientRank, budgetRupees); err != nil {
		return nil, err
	}
	requiredMinRank, err := utils.ComputeRequiredMinRank(budgetRupees)
	if err != nil {
		return nil, err
	}
	quest.RequiredMinRank = requiredMinRank

	replaceSkills := req.Skills != nil
	skills := make([]models.QuestSkill, 0)
	if replaceSkills {
		if len(*req.Skills) == 0 {
			return nil, errors.New("at least one technical skill is required")
		}
		for _, sk := range *req.Skills {
			prof := strings.TrimSpace(sk.ProficiencyLevel)
			if prof == "" {
				prof = "intermediate"
			}
			skills = append(skills, models.QuestSkill{
				Name:             strings.TrimSpace(sk.Name),
				ProficiencyLevel: prof,
				IsCore:           sk.IsCore,
			})
		}
	}

	if err := s.questRepo.UpdateDraftQuest(quest, skills, replaceSkills); err != nil {
		return nil, fmt.Errorf("failed to update quest draft: %w", err)
	}

	return &dto.UpdateQuestResponse{
		QuestID:         quest.ID.String(),
		Title:           quest.Title,
		Status:          string(quest.Status),
		BudgetMinor:     quest.BudgetMinor,
		RequiredMinRank: quest.RequiredMinRank,
		UpdatedAt:       quest.UpdatedAt,
	}, nil
}

// ============================================================
// 7. SaveQuest — toggle a bookmark
// ============================================================
func (s *questService) SaveQuest(userID string, questID string) (*dto.SaveQuestResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}

	saved, err := s.questRepo.ToggleSavedQuest(userUUID, qUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle saved quest: %w", err)
	}

	if saved {
		// Best-effort feed entry; a bookmark must not fail because of it.
		_ = s.questRepo.CreateQuestActivity(&models.QuestActivity{
			QuestID:     qUUID,
			ActorID:     userUUID,
			ActionType:  models.ActivityQuestSaved,
			Title:       "Quest saved",
			Description: quest.Title,
		})
	}

	return &dto.SaveQuestResponse{QuestID: qUUID.String(), Saved: saved}, nil
}

// ============================================================
// 8. ListSavedQuests — the caller's bookmarks
// ============================================================
func (s *questService) ListSavedQuests(userID string) (*dto.ListQuestsResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	quests, err := s.questRepo.ListSavedQuests(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved quests: %w", err)
	}

	ids := questIDs(quests)
	applicantCounts, err := s.questRepo.CountApplicantsByQuestIDs(ids)
	if err != nil {
		applicantCounts = map[uuid.UUID]int64{}
	}
	savedIDs := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		savedIDs[id] = true
	}

	userSkills := map[string]bool{}
	if names, err := s.questRepo.GetUserSkillNames(userUUID); err == nil {
		userSkills = lowerSkillSet(names)
	}

	items := make([]dto.QuestListItem, len(quests))
	for i, q := range quests {
		items[i] = s.questListItem(q, userSkills, savedIDs, applicantCounts)
	}

	return &dto.ListQuestsResponse{Quests: items, Total: len(items)}, nil
}

// ============================================================
// 9. GetBoardSummary — freelancer quest-board KPIs
// ============================================================
func (s *questService) GetBoardSummary(userID string) (*dto.QuestBoardSummaryResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	statusCounts, err := s.appRepo.CountByStatusForFreelancer(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}
	var applied int64
	for _, count := range statusCounts {
		applied += count
	}

	activeQuests, err := s.questRepo.ListQuestsForPartyViewer(userUUID, string(models.QuestStatusInProgress))
	if err != nil {
		return nil, fmt.Errorf("failed to list active quests: %w", err)
	}

	recent, err := s.appRepo.ListRecentByFreelancer(userUUID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent applications: %w", err)
	}
	recentItems := make([]dto.QuestBoardApplicationItem, 0, len(recent))
	for _, app := range recent {
		recentItems = append(recentItems, dto.QuestBoardApplicationItem{
			QuestID:    app.QuestID.String(),
			QuestTitle: app.Quest.Title,
			BudgetINR:  utils.PaiseToRupees(app.ProposedBudget),
			Status:     string(app.Status),
			AppliedAt:  app.CreatedAt,
		})
	}

	rank := user.Rank
	if rank == "" {
		rank = models.RankF
	}
	candidates, _, _, _, err := s.questRepo.ListQuests(repository.QuestFilter{
		AllowedRanks: utils.GetVisibleRanks(rank),
		Status:       string(models.QuestStatusOpen),
		Limit:        50,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load recommended quests: %w", err)
	}

	ids := questIDs(candidates)
	applicantCounts, err := s.questRepo.CountApplicantsByQuestIDs(ids)
	if err != nil {
		applicantCounts = map[uuid.UUID]int64{}
	}
	savedIDs, userSkills := s.viewerState(user, ids)

	recommended := make([]dto.QuestListItem, len(candidates))
	for i, q := range candidates {
		recommended[i] = s.questListItem(q, userSkills, savedIDs, applicantCounts)
	}
	sort.SliceStable(recommended, func(i, j int) bool {
		return recommended[i].MatchPercentage > recommended[j].MatchPercentage
	})
	if len(recommended) > 5 {
		recommended = recommended[:5]
	}

	return &dto.QuestBoardSummaryResponse{
		KPIs: dto.QuestBoardKPIs{
			AppliedCount:      int(applied),
			UnderReviewCount:  int(statusCounts[string(models.ApplicationPending)]),
			ShortlistedCount:  int(statusCounts[string(models.ApplicationShortlisted)]),
			ActiveQuestsCount: len(activeQuests),
		},
		RecentApplications: recentItems,
		RecommendedQuests:  recommended,
	}, nil
}

// ============================================================
// Card enrichment helpers
// ============================================================

func questIDs(quests []models.Quest) []uuid.UUID {
	ids := make([]uuid.UUID, len(quests))
	for i := range quests {
		ids[i] = quests[i].ID
	}
	return ids
}

// viewerState returns the caller's bookmarked quest ids and their lowercased
// skill set. Anonymous callers get empty maps.
func (s *questService) viewerState(viewer *models.User, ids []uuid.UUID) (map[uuid.UUID]bool, map[string]bool) {
	saved := map[uuid.UUID]bool{}
	skills := map[string]bool{}
	if viewer == nil {
		return saved, skills
	}
	if rows, err := s.questRepo.SavedQuestIDs(viewer.ID, ids); err == nil {
		saved = rows
	}
	if names, err := s.questRepo.GetUserSkillNames(viewer.ID); err == nil {
		skills = lowerSkillSet(names)
	}
	return saved, skills
}

func (s *questService) questListItem(q models.Quest, userSkills map[string]bool, saved map[uuid.UUID]bool, applicantCounts map[uuid.UUID]int64) dto.QuestListItem {
	skillNames := skillNamesOf(q.Skills)

	clientName := "Anonymous Client"
	if q.Client.FirstName != "" {
		clientName = strings.TrimSpace(q.Client.FirstName + " " + q.Client.LastName)
	}

	return dto.QuestListItem{
		ID:              q.ID.String(),
		Title:           q.Title,
		Description:     q.Description,
		Category:        q.Category,
		Complexity:      string(q.Complexity),
		BudgetINR:       utils.PaiseToRupees(q.BudgetMinor),
		TimelineDays:    q.TimelineDays,
		RequiredMinRank: q.RequiredMinRank,
		Status:          string(q.Status),
		ClientID:        q.ClientID.String(),
		ClientName:      clientName,
		Skills:          skillNames,
		MilestoneCount:  len(q.Milestones),
		PublishedAt:     q.PublishedAt,
		CreatedAt:       q.CreatedAt,
		QuestType:       string(q.QuestType),
		ClientRank:      q.Client.Rank,
		ApplicantsCount: int(applicantCounts[q.ID]),
		ViewsCount:      q.ViewsCount,
		IsSaved:         saved[q.ID],
		MatchPercentage: matchPercentage(skillNames, userSkills),
	}
}

func skillNamesOf(skills []models.QuestSkill) []string {
	names := make([]string, len(skills))
	for i, sk := range skills {
		names[i] = sk.Name
	}
	return names
}

func lowerSkillSet(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, name := range names {
		if v := strings.ToLower(strings.TrimSpace(name)); v != "" {
			out[v] = true
		}
	}
	return out
}

// matchPercentage is the share of the quest's required skills the caller has.
func matchPercentage(questSkillNames []string, userSkills map[string]bool) int {
	if len(questSkillNames) == 0 || len(userSkills) == 0 {
		return 0
	}
	hits := 0
	for _, name := range questSkillNames {
		if userSkills[strings.ToLower(strings.TrimSpace(name))] {
			hits++
		}
	}
	return hits * 100 / len(questSkillNames)
}

// splitCSV is the lenient parser for comma-separated query params.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// timelineBounds maps a timeline bucket to inclusive day bounds.
func timelineBounds(bucket string) (*int, *int) {
	switch strings.TrimSpace(bucket) {
	case "under_7":
		max := 6
		return nil, &max
	case "7_15":
		min, max := 7, 15
		return &min, &max
	case "15_30":
		min, max := 15, 30
		return &min, &max
	case "30_plus":
		min := 31
		return &min, nil
	default:
		return nil, nil
	}
}
