package services

import (
	"encoding/json"
	"errors"
	"fmt"
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
	ErrOnlyClientAllowed      = errors.New("only client accounts can create quests")
	ErrClientNotVerified      = errors.New("client account verification required before creating quests")
	ErrInvalidUserID          = errors.New("invalid user id")
	ErrInvalidQuestID         = errors.New("invalid quest id")
	ErrQuestNotFound          = errors.New("quest not found")
	ErrQuestNotVisible        = errors.New("quest is not visible for your rank")
	ErrQuestNotDraft          = errors.New("only draft quests can be published")
	ErrClientFundingRequired  = errors.New("client must configure a funding method before publishing a quest")
	ErrOnlyPartyLeadAllowed   = errors.New("only Party Lead of an active, verified party can accept quests")
	ErrPartyNotVerified       = errors.New("party must be verified to accept quests")
	ErrPartyCapacityExceeded  = errors.New("party has reached maximum concurrent quests capacity (5)")
	ErrSkillMatchInsufficient = errors.New("party does not meet the minimum 60% skill match requirement")
	ErrClientConflict         = errors.New("party already has an active quest with this client")
	ErrEscrowNotFunded        = errors.New("client has not deposited funds into escrow for this quest")
	ErrQuestNotOpen           = errors.New("quest is not open for acceptance")
	ErrOnlyClientCanCancel    = errors.New("only the client owner can cancel/reject this quest")
	ErrQuestAlreadyFinished   = errors.New("quest is already completed or cancelled")
)

type QuestService interface {
	CreateQuestDraft(userID string, req dto.CreateQuestBody) (*dto.CreateQuestResponse, error)
	PublishQuest(userID string, questID string, req dto.PublishQuestBody) (*dto.PublishQuestResponse, error)
	ListQuests(userID string, req dto.ListQuestsInput) (*dto.ListQuestsResponse, error)
	GetQuestDetails(userID string, questID string) (*dto.QuestDetailsResponse, error)
	RejectQuest(userID string, questID string, req dto.RejectQuestBody) (*dto.RejectQuestResponse, error)
	AcceptQuest(userID string, questID string, req dto.AcceptQuestBody) (*dto.AcceptQuestResponse, error)
}

type questService struct {
	questRepo repository.QuestRepository
	userRepo  repository.UserRepository
	partyRepo *repository.PartyRepository
}

func NewQuestService() QuestService {
	return &questService{
		questRepo: repository.NewQuestRepository(),
		userRepo:  repository.NewUserRepository(),
		partyRepo: repository.NewPartyRepository(),
	}
}

func NewQuestServiceWithRepos(
	questRepo repository.QuestRepository,
	userRepo repository.UserRepository,
	partyRepo *repository.PartyRepository,
) QuestService {
	return &questService{
		questRepo: questRepo,
		userRepo:  userRepo,
		partyRepo: partyRepo,
	}
}

// 1. CreateQuestDraft creates a draft quest (NO milestones — freelancer creates them later)
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

	// Budget is reference-only
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

	// Milestones are NO LONGER created here.
	// Freelancer will create them after quest is accepted.
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

// 2. PublishQuest publishes a draft quest
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

	hasFundingMethod := false
	if strings.TrimSpace(req.FundingMethod) != "" {
		hasFundingMethod = true
	} else if cp, err := s.questRepo.GetClientProfile(clientUUID); err == nil && cp != nil {
		hasFundingMethod = true
	} else if vr, err := s.userRepo.GetVerificationRecordByUserID(userID); err == nil && vr != nil {
		if vr.BankName != "" || vr.AccountNumber != "" || vr.Status == models.VerificationApproved {
			hasFundingMethod = true
		}
	}

	if !hasFundingMethod {
		return nil, ErrClientFundingRequired
	}

	now := time.Now()
	quest.Status = models.QuestStatusOpen
	quest.PublishedAt = &now

	if req.FundEscrowNow == nil || *req.FundEscrowNow {
		quest.IsEscrowFunded = true
		quest.EscrowFundedAt = &now
	}

	if err := s.questRepo.UpdateQuest(quest); err != nil {
		return nil, fmt.Errorf("failed to publish quest: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"budget_minor":      quest.BudgetMinor,
		"required_min_rank": quest.RequiredMinRank,
		"is_escrow_funded":  quest.IsEscrowFunded,
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
		Message:        "Quest published successfully and open for acceptance",
	}, nil
}

// 3. ListQuests
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

	filter := repository.QuestFilter{
		Category:   req.Category,
		Complexity: req.Complexity,
		MinBudget:  minBudgetPtr,
		MaxBudget:  maxBudgetPtr,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
	}

	if user != nil && user.AccountType != nil &&
		(string(*user.AccountType) == string(models.AccountTypeClientAdmin) || string(*user.AccountType) == string(models.AccountTypeClient)) &&
		req.Status == string(models.QuestStatusDraft) {
		clientUUID := user.ID
		filter.ClientID = &clientUUID
		filter.Status = req.Status
	} else {
		filter.AllowedRanks = utils.GetVisibleRanks(requesterRank)
		if req.Status != "" {
			filter.Status = req.Status
		} else {
			filter.StatusIn = []string{string(models.QuestStatusOpen), string(models.QuestStatusPublished)}
		}
	}

	quests, nextCursor, hasMore, total, err := s.questRepo.ListQuests(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query quests: %w", err)
	}

	items := make([]dto.QuestListItem, len(quests))
	for i, q := range quests {
		skillNames := make([]string, len(q.Skills))
		for j, sk := range q.Skills {
			skillNames[j] = sk.Name
		}

		clientName := "Anonymous Client"
		if q.Client.FirstName != "" {
			clientName = strings.TrimSpace(q.Client.FirstName + " " + q.Client.LastName)
		}

		items[i] = dto.QuestListItem{
			ID:              q.ID.String(),
			Title:           q.Title,
			Description:     q.Description,
			Category:        q.Category,
			Complexity:      string(q.Complexity),
			BudgetMinor:     q.BudgetMinor,
			TimelineDays:    q.TimelineDays,
			RequiredMinRank: q.RequiredMinRank,
			Status:          string(q.Status),
			ClientID:        q.ClientID.String(),
			ClientName:      clientName,
			Skills:          skillNames,
			MilestoneCount:  len(q.Milestones),
			PublishedAt:     q.PublishedAt,
			CreatedAt:       q.CreatedAt,
		}
	}

	return &dto.ListQuestsResponse{
		Quests:     items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      int(total),
	}, nil
}

// 4. GetQuestDetails
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
	isAdmin := user != nil && user.AccountType != nil && string(*user.AccountType) == string(models.AccountTypeAdmin)

	if !isOwner && !isAdmin {
		if quest.Status != models.QuestStatusOpen && quest.Status != models.QuestStatusPublished {
			if quest.Status == models.QuestStatusInProgress && quest.AcceptedPartyID != nil {
				if user == nil {
					return nil, ErrQuestNotVisible
				}
				members, err := s.partyRepo.GetMembers(*quest.AcceptedPartyID)
				isMember := false
				if err == nil {
					for _, m := range members {
						if m.UserID == requesterUUID && m.Status == models.MemberActive {
							isMember = true
							break
						}
					}
				}
				if !isMember && (quest.AssignedFreelancerID == nil || *quest.AssignedFreelancerID != requesterUUID) {
					return nil, ErrQuestNotVisible
				}
			} else {
				return nil, ErrQuestNotVisible
			}
		} else {
			if !utils.IsRankSufficient(requesterRank, quest.RequiredMinRank) {
				return nil, fmt.Errorf("quest requires rank %s which is higher than your rank %s", quest.RequiredMinRank, requesterRank)
			}
		}
	}

	milestones := make([]dto.QuestDetailMilestone, len(quest.Milestones))
	for i, m := range quest.Milestones {
		milestones[i] = dto.QuestDetailMilestone{
			ID:          m.ID.String(),
			Title:       m.Title,
			Description: m.Description,
			AmountMinor: m.AmountMinor,
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
		BudgetMinor:     quest.BudgetMinor,
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
		CreatedAt:       quest.CreatedAt,
		UpdatedAt:       quest.UpdatedAt,
	}, nil
}

// 5. RejectQuest
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

	if quest.Status == models.QuestStatusCancelled || quest.Status == models.QuestStatusCompleted {
		return nil, ErrQuestAlreadyFinished
	}

	escrowRefunded := false
	refundAmount := int64(0)

	if quest.IsEscrowFunded {
		escrowRefunded = true
		refundAmount = quest.BudgetMinor
		quest.IsEscrowFunded = false
		now := time.Now()
		quest.EscrowRefundedAt = &now
	}

	quest.Status = models.QuestStatusCancelled
	if err := s.questRepo.UpdateQuest(quest); err != nil {
		return nil, fmt.Errorf("failed to cancel quest: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"refunded":      escrowRefunded,
		"refund_amount": refundAmount,
		"reason":        req.Reason,
	})
	_ = s.questRepo.CreateAuditLog(&models.AuditLog{
		UserID:   &clientUUID,
		Action:   "quest_cancelled",
		Entity:   "quest",
		EntityID: quest.ID.String(),
		Metadata: datatypes.JSON(meta),
	})

	return &dto.RejectQuestResponse{
		QuestID:            quest.ID.String(),
		Status:             string(quest.Status),
		EscrowRefunded:     escrowRefunded,
		EscrowRefundAmount: refundAmount,
		Message:            "Quest has been cancelled successfully",
	}, nil
}

// 6. AcceptQuest
func (s *questService) AcceptQuest(userID string, questID string, req dto.AcceptQuestBody) (*dto.AcceptQuestResponse, error) {
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

	if quest.Status != models.QuestStatusOpen && quest.Status != models.QuestStatusPublished {
		return nil, ErrQuestNotOpen
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	var party *models.Party
	if strings.TrimSpace(req.PartyID) != "" {
		pUUID, err := uuid.Parse(req.PartyID)
		if err != nil {
			return nil, errors.New("invalid party_id")
		}
		p, err := s.partyRepo.FindByID(pUUID)
		if err != nil || p == nil {
			return nil, errors.New("party not found")
		}
		if p.LeaderID != userUUID {
			return nil, ErrOnlyPartyLeadAllowed
		}
		party = p
	} else {
		parties, err := s.partyRepo.FindByLeader(userUUID)
		if err != nil || len(parties) == 0 {
			return nil, ErrOnlyPartyLeadAllowed
		}
		p, err := s.partyRepo.FindByID(parties[0].ID)
		if err != nil || p == nil {
			return nil, errors.New("party not found")
		}
		party = p
	}

	if !party.IsVerified {
		return nil, ErrPartyNotVerified
	}

	partyRank := user.Rank
	if partyRank == "" {
		partyRank = models.RankF
	}
	for _, m := range party.Members {
		if m.Status == models.MemberActive && m.User.Rank != "" {
			if utils.CompareRanks(m.User.Rank, partyRank) > 0 {
				partyRank = m.User.Rank
			}
		}
	}

	if !utils.IsRankSufficient(partyRank, quest.RequiredMinRank) {
		return nil, fmt.Errorf("party rank %s is insufficient for quest required rank %s", partyRank, quest.RequiredMinRank)
	}

	activeCount, err := s.questRepo.CountPartyActiveQuests(party.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify party capacity: %w", err)
	}
	if activeCount >= 5 {
		return nil, ErrPartyCapacityExceeded
	}

	if len(quest.Skills) > 0 {
		partySkills, err := s.partyRepo.GetSkills(party.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve party skills: %w", err)
		}

		partySkillMap := make(map[string]bool)
		for _, ps := range partySkills {
			partySkillMap[strings.ToLower(strings.TrimSpace(ps.SkillName))] = true
		}

		matchedCount := 0
		for _, qs := range quest.Skills {
			if partySkillMap[strings.ToLower(strings.TrimSpace(qs.Name))] {
				matchedCount++
			}
		}

		matchPercentage := (float64(matchedCount) / float64(len(quest.Skills))) * 100.0
		if matchPercentage < 60.0 {
			return nil, fmt.Errorf("party skills match %.1f%% of required skills, minimum 60%% required", matchPercentage)
		}
	}

	hasConflict, err := s.questRepo.HasActiveQuestWithClient(party.ID, quest.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify client conflict: %w", err)
	}
	if hasConflict {
		return nil, ErrClientConflict
	}

	isPremium := quest.RequiredMinRank == models.RankS ||
		quest.RequiredMinRank == models.RankSS ||
		quest.RequiredMinRank == models.RankSSS ||
		quest.Complexity == models.QuestComplexityEnterprise

	if isPremium && user.Status != models.StatusActive {
		return nil, errors.New("active verified status required to accept premium tier quests")
	}

	if !quest.IsEscrowFunded {
		return nil, ErrEscrowNotFunded
	}

	now := time.Now()
	quest.Status = models.QuestStatusInProgress
	quest.AcceptedPartyID = &party.ID
	quest.AssignedFreelancerID = &party.LeaderID

	chatChannel := &models.QuestChatChannel{
		QuestID:   quest.ID,
		PartyID:   &party.ID,
		Name:      fmt.Sprintf("Quest: %s", quest.Title),
		Type:      "party_client",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tasks := make([]models.Task, 0, len(quest.Milestones))
	for _, m := range quest.Milestones {
		endDate := now.Add(time.Duration(m.DueDays) * 24 * time.Hour)
		tasks = append(tasks, models.Task{
			MilestoneID: m.ID,
			QuestID:     quest.ID,
			Title:       fmt.Sprintf("Deliverable: %s", m.Title),
			Description: m.Description,
			AssigneeID:  &party.LeaderID,
			StartDate:   &now,
			EndDate:     &endDate,
			Status:      models.TaskStatusToDo,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	activity := &models.QuestActivity{
		QuestID:     quest.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityQuestAssigned,
		Title:       fmt.Sprintf("Quest accepted by %s", party.Name),
		Description: fmt.Sprintf("Party Lead %s accepted the quest on behalf of party %s.", user.FirstName, party.Name),
		CreatedAt:   now,
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"party_id":      party.ID.String(),
		"party_name":    party.Name,
		"leader_id":     userUUID.String(),
		"tasks_created": len(tasks),
		"accepted_at":   now,
	})
	auditLog := &models.AuditLog{
		UserID:   &userUUID,
		Action:   "quest_accepted",
		Entity:   "quest",
		EntityID: quest.ID.String(),
		Metadata: datatypes.JSON(meta),
	}

	if err := s.questRepo.AcceptQuestAtomic(quest, chatChannel, tasks, activity, auditLog); err != nil {
		return nil, fmt.Errorf("failed to complete quest acceptance: %w", err)
	}

	return &dto.AcceptQuestResponse{
		QuestID:       quest.ID.String(),
		PartyID:       party.ID.String(),
		PartyName:     party.Name,
		Status:        string(quest.Status),
		ChatChannelID: chatChannel.ID.String(),
		TasksCreated:  len(tasks),
		AcceptedAt:    now,
		Message:       "Quest accepted successfully by party",
	}, nil
}
