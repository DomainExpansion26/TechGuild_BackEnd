package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

const (
	ClientFeePercent     = 5.0
	FreelancerFeePercent = 5.0
)

var (
	ErrFreelancerOnly              = errors.New("only freelancer or party lead accounts can apply")
	ErrFreelancerNotVerified       = errors.New("freelancer account must be active/verified")
	ErrQuestNotOpenForApplications = errors.New("quest is not open for applications")
	ErrAlreadyApplied              = errors.New("you have already applied to this quest")
	ErrApplicationNotFound         = errors.New("application not found")
	ErrOnlyClientCanReview         = errors.New("only the quest client can review applications")
	ErrApplicationNotPending       = errors.New("application is no longer pending")
	ErrFreelancerRankInsufficient  = errors.New("your rank is insufficient for this quest")
	ErrNotApplicationOwner         = errors.New("you are not the owner of this application")
	ErrCannotApplyOwnQuest         = errors.New("you cannot apply to your own quest")
	ErrPartySkillMatchInsufficient = errors.New("party skill match is below 60%")
)

type QuestApplicationService interface {
	ApplyToQuest(userID, questID string, req dto.ApplyQuestBody) (*dto.ApplyQuestResponse, error)
	ListApplications(userID, questID, status string) (*dto.ListApplicationsResponse, error)
	AcceptApplication(userID, questID, applicationID string) (*dto.AcceptApplicationResponse, error)
	RejectApplication(userID, questID, applicationID, reason string) (*dto.RejectApplicationResponse, error)
	WithdrawApplication(userID, questID, applicationID string) (*dto.WithdrawApplicationResponse, error)
	MyWork(userID, status string) (*dto.MyWorkResponse, error)
}

type questApplicationService struct {
	appRepo   repository.QuestApplicationRepository
	questRepo repository.QuestRepository
	userRepo  repository.UserRepository
	partyRepo *repository.PartyRepository
}

func NewQuestApplicationService() QuestApplicationService {
	return &questApplicationService{
		appRepo:   repository.NewQuestApplicationRepository(),
		questRepo: repository.NewQuestRepository(),
		userRepo:  repository.NewUserRepository(),
		partyRepo: repository.NewPartyRepository(),
	}
}

func isFreelancerAccount(user *models.User) bool {
	if user == nil || user.AccountType == nil {
		return false
	}
	at := string(*user.AccountType)
	return at == string(models.AccountTypeIndividual) ||
		at == string(models.AccountTypeAgencyAdmin)
}

// ============ 1. APPLY ============
func (s *questApplicationService) ApplyToQuest(userID, questID string, req dto.ApplyQuestBody) (*dto.ApplyQuestResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}
	if !isFreelancerAccount(user) {
		return nil, ErrFreelancerOnly
	}
	switch user.Status {
	case models.StatusActive:
		// OK
	case models.StatusSuspended, models.StatusRejected, models.StatusDeleted:
		return nil, ErrFreelancerNotVerified
	default:
		// pending_verification etc. — require approved verification
		vRec, err := s.userRepo.GetVerificationRecordByUserID(userID)
		if err != nil || vRec == nil || vRec.Status != models.VerificationApproved {
			return nil, ErrFreelancerNotVerified
		}
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}
	if quest.Status != models.QuestStatusOpen && quest.Status != models.QuestStatusPublished {
		return nil, ErrQuestNotOpenForApplications
	}
	if quest.ClientID == userUUID {
		return nil, ErrCannotApplyOwnQuest
	}

	// Determine if applying as party
	var partyID *uuid.UUID
	if strings.TrimSpace(req.PartyID) != "" {
		pUUID, err := uuid.Parse(req.PartyID)
		if err != nil {
			return nil, errors.New("invalid party_id")
		}
		party, err := s.partyRepo.FindByID(pUUID)
		if err != nil || party == nil {
			return nil, errors.New("party not found")
		}
		if party.LeaderID != userUUID {
			return nil, ErrOnlyPartyLeadAllowed
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
			return nil, ErrFreelancerRankInsufficient
		}

		// Party skill match
		if len(quest.Skills) > 0 {
			partySkills, err := s.partyRepo.GetSkills(party.ID)
			if err == nil {
				skillMap := make(map[string]bool)
				for _, ps := range partySkills {
					skillMap[strings.ToLower(strings.TrimSpace(ps.SkillName))] = true
				}
				matched := 0
				for _, qs := range quest.Skills {
					if skillMap[strings.ToLower(strings.TrimSpace(qs.Name))] {
						matched++
					}
				}
				if (float64(matched)/float64(len(quest.Skills)))*100.0 < 60.0 {
					return nil, ErrPartySkillMatchInsufficient
				}
			}
		}

		partyID = &pUUID
	} else {
		// Individual rank check
		userRank := user.Rank
		if userRank == "" {
			userRank = models.RankF
		}
		if !utils.IsRankSufficient(userRank, quest.RequiredMinRank) {
			return nil, ErrFreelancerRankInsufficient
		}
	}
	// Validate inputs FIRST (so reapply block can reuse them)
	cover := strings.TrimSpace(req.CoverLetter)
	if len(cover) < 20 {
		return nil, errors.New("cover letter must be at least 20 characters")
	}
	if req.ProposedBudget <= 0 {
		return nil, errors.New("proposed budget must be > 0")
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "INR"
	}

	// Duplicate / re-apply check
	if existing, _ := s.appRepo.FindByQuestAndFreelancer(qUUID, userUUID); existing != nil {
		if existing.Status != models.ApplicationWithdrawn {
			return nil, ErrAlreadyApplied
		}
		// Reactivate the withdrawn application with new details
		existing.Status = models.ApplicationPending
		existing.CoverLetter = cover
		existing.ProposedBudget = utils.RupeesToPaise(req.ProposedBudget)
		existing.Currency = currency
		existing.EstimatedDuration = strings.TrimSpace(req.EstimatedDuration)
		existing.WithdrawnAt = nil
		if err := s.appRepo.Update(existing); err != nil {
			return nil, fmt.Errorf("failed to reapply: %w", err)
		}
		proposedINR := utils.PaiseToRupees(existing.ProposedBudget)
		netPayout := proposedINR * (1.0 - FreelancerFeePercent/100.0)
		return &dto.ApplyQuestResponse{
			ApplicationID:         existing.ID.String(),
			QuestID:               existing.QuestID.String(),
			Status:                string(existing.Status),
			ProposedBudgetINR:     proposedINR,
			EstimatedNetPayoutINR: netPayout,
			CreatedAt:             existing.CreatedAt,
			Message:               "Application resubmitted successfully",
		}, nil
	}

	app := &models.QuestApplication{
		QuestID:           qUUID,
		FreelancerID:      userUUID,
		PartyID:           partyID,
		CoverLetter:       cover,
		ProposedBudget:    utils.RupeesToPaise(req.ProposedBudget),
		Currency:          currency,
		EstimatedDuration: strings.TrimSpace(req.EstimatedDuration),
		Status:            models.ApplicationPending,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, fmt.Errorf("failed to submit application: %w", err)
	}

	proposedINR := utils.PaiseToRupees(app.ProposedBudget)
	netPayout := proposedINR * (1.0 - FreelancerFeePercent/100.0)

	return &dto.ApplyQuestResponse{
		ApplicationID:         app.ID.String(),
		QuestID:               app.QuestID.String(),
		Status:                string(app.Status),
		ProposedBudgetINR:     proposedINR,
		EstimatedNetPayoutINR: netPayout,
		CreatedAt:             app.CreatedAt,
		Message:               "Application submitted successfully",
	}, nil
}

// ============ 2. LIST ============
func (s *questApplicationService) ListApplications(userID, questID, status string) (*dto.ListApplicationsResponse, error) {
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

	// Client sees all
	if quest.ClientID == userUUID {
		apps, err := s.appRepo.ListByQuestID(qUUID, status)
		if err != nil {
			return nil, fmt.Errorf("failed to list applications: %w", err)
		}
		items := make([]dto.ApplicationItem, len(apps))
		for i := range apps {
			items[i] = toApplicationItem(&apps[i])
		}
		return &dto.ListApplicationsResponse{Applications: items, Total: len(items)}, nil
	}

	// Freelancer/party lead sees only their own application
	own, _ := s.appRepo.FindByQuestAndFreelancer(qUUID, userUUID)
	if own == nil {
		return &dto.ListApplicationsResponse{Applications: []dto.ApplicationItem{}, Total: 0}, nil
	}
	// Preload Freelancer + Party for response
	full, err := s.appRepo.FindByID(own.ID)
	if err != nil || full == nil {
		return nil, ErrApplicationNotFound
	}
	return &dto.ListApplicationsResponse{
		Applications: []dto.ApplicationItem{toApplicationItem(full)},
		Total:        1,
	}, nil
}

func toApplicationItem(a *models.QuestApplication) dto.ApplicationItem {
	name := strings.TrimSpace(a.Freelancer.FirstName + " " + a.Freelancer.LastName)
	if name == "" {
		name = "Freelancer"
	}
	item := dto.ApplicationItem{
		ID:                a.ID.String(),
		FreelancerID:      a.FreelancerID.String(),
		FreelancerName:    name,
		FreelancerRank:    a.Freelancer.Rank,
		CoverLetter:       a.CoverLetter,
		ProposedBudgetINR: utils.PaiseToRupees(a.ProposedBudget),
		Currency:          a.Currency,
		EstimatedDuration: a.EstimatedDuration,
		Status:            string(a.Status),
		CreatedAt:         a.CreatedAt,
		IsAgency:          a.PartyID != nil,
	}
	if a.PartyID != nil {
		pid := a.PartyID.String()
		item.PartyID = &pid
		if a.Party != nil {
			pn := a.Party.Name
			item.PartyName = &pn
		}
	}
	return item
}

// ============ 3. ACCEPT (Client picks winner) ============
func (s *questApplicationService) AcceptApplication(userID, questID, applicationID string) (*dto.AcceptApplicationResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}
	aUUID, err := uuid.Parse(applicationID)
	if err != nil {
		return nil, ErrApplicationNotFound
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrOnlyClientCanReview
	}
	if quest.Status != models.QuestStatusOpen && quest.Status != models.QuestStatusPublished {
		return nil, ErrQuestNotOpenForApplications
	}
	if !quest.IsEscrowFunded {
		return nil, ErrEscrowNotFunded
	}

	app, err := s.appRepo.FindByID(aUUID)
	if err != nil || app == nil || app.QuestID != qUUID {
		return nil, ErrApplicationNotFound
	}
	if app.Status != models.ApplicationPending {
		return nil, ErrApplicationNotPending
	}

	now := time.Now()

	// Accept
	app.Status = models.ApplicationAccepted
	app.AcceptedAt = &now

	// Update quest — set assigned freelancer AND accepted party (if agency)
	quest.Status = models.QuestStatusInProgress
	quest.AssignedFreelancerID = &app.FreelancerID
	quest.AcceptedPartyID = app.PartyID // nil for individual

	// Auto-reject other pending apps
	pendingApps, _ := s.appRepo.ListByQuestID(qUUID, string(models.ApplicationPending))
	otherApps := make([]models.QuestApplication, 0, len(pendingApps))
	for i := range pendingApps {
		if pendingApps[i].ID == app.ID {
			continue
		}
		pendingApps[i].Status = models.ApplicationRejected
		pendingApps[i].RejectedAt = &now
		pendingApps[i].RejectReason = "Another applicant was selected"
		otherApps = append(otherApps, pendingApps[i])
	}

	// Chat channel
	chatType := "direct_client"
	var chatPartyID *uuid.UUID
	if app.PartyID != nil {
		chatType = "party_client"
		chatPartyID = app.PartyID
	}
	chat := &models.QuestChatChannel{
		QuestID:   quest.ID,
		PartyID:   chatPartyID,
		Name:      fmt.Sprintf("Quest: %s", quest.Title),
		Type:      chatType,
		CreatedAt: now,
		UpdatedAt: now,
	}

	activity := &models.QuestActivity{
		QuestID:     quest.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityQuestAssigned,
		Title:       fmt.Sprintf("Quest assigned to %s", strings.TrimSpace(app.Freelancer.FirstName+" "+app.Freelancer.LastName)),
		Description: "Client accepted an application and assigned the quest.",
		CreatedAt:   now,
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"application_id": app.ID.String(),
		"freelancer_id":  app.FreelancerID.String(),
		"party_id":       app.PartyID,
	})
	audit := &models.AuditLog{
		UserID:   &userUUID,
		Action:   "application_accepted",
		Entity:   "quest_application",
		EntityID: app.ID.String(),
		Metadata: datatypes.JSON(meta),
	}

	if err := s.appRepo.AcceptApplicationAtomic(app, quest, otherApps, chat, activity, audit); err != nil {
		return nil, fmt.Errorf("failed to accept application: %w", err)
	}

	resp := &dto.AcceptApplicationResponse{
		ApplicationID:     app.ID.String(),
		QuestID:           quest.ID.String(),
		FreelancerID:      app.FreelancerID.String(),
		FreelancerName:    strings.TrimSpace(app.Freelancer.FirstName + " " + app.Freelancer.LastName),
		ApplicationStatus: string(app.Status),
		QuestStatus:       string(quest.Status),
		ChatChannelID:     chat.ID.String(),
		AutoRejectedCount: len(otherApps),
		AcceptedAt:        now,
		IsAgency:          app.PartyID != nil,
		Message:           "Application accepted and quest assigned",
	}
	if app.PartyID != nil {
		pid := app.PartyID.String()
		resp.PartyID = &pid
		if app.Party != nil {
			pn := app.Party.Name
			resp.PartyName = &pn
		}
	}
	return resp, nil
}

// ============ 4. REJECT ============
func (s *questApplicationService) RejectApplication(userID, questID, applicationID, reason string) (*dto.RejectApplicationResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}
	aUUID, err := uuid.Parse(applicationID)
	if err != nil {
		return nil, ErrApplicationNotFound
	}

	quest, err := s.questRepo.FindByID(qUUID)
	if err != nil || quest == nil {
		return nil, ErrQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrOnlyClientCanReview
	}

	app, err := s.appRepo.FindByID(aUUID)
	if err != nil || app == nil || app.QuestID != qUUID {
		return nil, ErrApplicationNotFound
	}
	if app.Status != models.ApplicationPending {
		return nil, ErrApplicationNotPending
	}

	now := time.Now()
	app.Status = models.ApplicationRejected
	app.RejectedAt = &now
	app.RejectReason = strings.TrimSpace(reason)

	if err := s.appRepo.Update(app); err != nil {
		return nil, fmt.Errorf("failed to reject: %w", err)
	}
	return &dto.RejectApplicationResponse{
		ApplicationID: app.ID.String(),
		Status:        string(app.Status),
		Message:       "Application rejected",
	}, nil
}

// ============ 5. WITHDRAW ============
func (s *questApplicationService) WithdrawApplication(userID, questID, applicationID string) (*dto.WithdrawApplicationResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	qUUID, err := uuid.Parse(questID)
	if err != nil {
		return nil, ErrInvalidQuestID
	}
	aUUID, err := uuid.Parse(applicationID)
	if err != nil {
		return nil, ErrApplicationNotFound
	}

	app, err := s.appRepo.FindByID(aUUID)
	if err != nil || app == nil || app.QuestID != qUUID {
		return nil, ErrApplicationNotFound
	}
	if app.FreelancerID != userUUID {
		return nil, ErrNotApplicationOwner
	}
	if app.Status != models.ApplicationPending {
		return nil, ErrApplicationNotPending
	}

	now := time.Now()
	app.Status = models.ApplicationWithdrawn
	app.WithdrawnAt = &now

	if err := s.appRepo.Update(app); err != nil {
		return nil, fmt.Errorf("failed to withdraw: %w", err)
	}
	return &dto.WithdrawApplicationResponse{
		ApplicationID: app.ID.String(),
		Status:        string(app.Status),
		Message:       "Application withdrawn",
	}, nil
}

// ============ 6. MY WORK (freelancer's accepted quests) ============
func (s *questApplicationService) MyWork(userID, status string) (*dto.MyWorkResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if status == "" {
		status = string(models.QuestStatusInProgress)
	}

	quests, err := s.questRepo.ListAssignedQuests(userUUID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list my work: %w", err)
	}

	items := make([]dto.MyWorkItem, len(quests))
	for i, q := range quests {
		clientName := "Client"
		if q.Client.FirstName != "" {
			clientName = strings.TrimSpace(q.Client.FirstName + " " + q.Client.LastName)
		}
		items[i] = dto.MyWorkItem{
			QuestID:        q.ID.String(),
			Title:          q.Title,
			Description:    q.Description,
			Category:       q.Category,
			BudgetINR:      utils.PaiseToRupees(q.BudgetMinor),
			Status:         string(q.Status),
			ClientID:       q.ClientID.String(),
			ClientName:     clientName,
			IsAgency:       q.AcceptedPartyID != nil,
			MilestoneCount: len(q.Milestones),
			AcceptedAt:     nil, // can add later from audit
		}
	}

	return &dto.MyWorkResponse{Quests: items, Total: len(items)}, nil
}
