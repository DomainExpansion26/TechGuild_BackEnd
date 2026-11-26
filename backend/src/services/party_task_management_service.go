package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

var (
	ErrPartyTMDForbidden         = errors.New("forbidden: caller is not a team lead or team member of this quest")
	ErrPartyTMDQuestNotFound     = errors.New("quest not found")
	ErrPartyTMDMilestoneNotFound = errors.New("milestone not found")
	ErrPartyTMDAssignmentState   = errors.New("milestone is not in a state that accepts assignments")
	ErrPartyTMDMemberInvalid     = errors.New("member is not an active member of this party")
	ErrPartyTMDInvalidAssignee   = errors.New("task assignee must be the lead or an active party member")
	ErrPartyTMDInvalidDateRange  = errors.New("start_date must be before end_date")
	ErrPartyTMDInvalidStatus     = errors.New("invalid task status")
	ErrPartyTMDNoMembers         = errors.New("at least one member id is required")
)

// PartyTaskManagementService powers the freelancer/party-side dashboard.
type PartyTaskManagementService interface {
	GetDashboard(userIDStr, questIDStr string) (*dto.PartyTMDDashboardResponse, error)
	GetActiveQuests(userIDStr string) (*dto.PartyTMDActiveQuestsResponse, error)
	GetMembersOverview(userIDStr, questIDStr string) (*dto.PartyTMDMembersResponse, error)
	AssignMilestone(userIDStr, milestoneIDStr string, req dto.AssignMilestoneBody) (*dto.AssignMilestoneResponse, error)
	ListAssignments(userIDStr, milestoneIDStr string) (*dto.ListMilestoneAssignmentsResponse, error)
}

type partyTaskManagementService struct {
	questRepo  repository.QuestRepository
	taskRepo   repository.TaskManagementRepository
	partyRepo  *repository.PartyRepository
	assignRepo repository.MilestoneAssignmentRepository
	attachRepo repository.MilestoneAttachmentRepository
}

func NewPartyTaskManagementService() PartyTaskManagementService {
	return &partyTaskManagementService{
		questRepo:  repository.NewQuestRepository(),
		taskRepo:   repository.NewTaskManagementRepository(),
		partyRepo:  repository.NewPartyRepository(),
		assignRepo: repository.NewMilestoneAssignmentRepository(),
		attachRepo: repository.NewMilestoneAttachmentRepository(),
	}
}

// memberLookup resolves a party member row; injected so role resolution is testable.
type memberLookup func(partyID, userID uuid.UUID) (*models.PartyMember, error)

// roleForQuest derives the caller's party role. Clients are rejected here on
// purpose: they use the client-side TMD routes.
func roleForQuest(q *models.Quest, userID uuid.UUID, findMember memberLookup) (string, error) {
	if q.ClientID == userID {
		return "", ErrPartyTMDForbidden
	}
	if q.AssignedFreelancerID != nil && *q.AssignedFreelancerID == userID {
		return dto.PartyTMDRoleTeamLead, nil
	}
	if q.AcceptedPartyID != nil && findMember != nil {
		m, err := findMember(*q.AcceptedPartyID, userID)
		if err == nil && m.Status == models.MemberActive {
			return dto.PartyTMDRoleTeamMember, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	return "", ErrPartyTMDForbidden
}

func (s *partyTaskManagementService) viewerRole(q *models.Quest, userID uuid.UUID) (string, error) {
	return roleForQuest(q, userID, func(partyID, uid uuid.UUID) (*models.PartyMember, error) {
		return s.partyRepo.FindMember(partyID, uid)
	})
}

// isParticipant allows the client plus team members (used for read endpoints).
func (s *partyTaskManagementService) isParticipant(q *models.Quest, userID uuid.UUID) bool {
	if q.ClientID == userID {
		return true
	}
	if q.AssignedFreelancerID != nil && *q.AssignedFreelancerID == userID {
		return true
	}
	if q.AcceptedPartyID != nil {
		m, err := s.partyRepo.FindMember(*q.AcceptedPartyID, userID)
		return err == nil && m.Status == models.MemberActive
	}
	return false
}

// ------------------------------------------------------------
// Read helpers
// ------------------------------------------------------------

// activeMembers returns the accepted party's active members. Solo quests have no
// party row, so the assigned freelancer (the "lead") is the only member.
func (s *partyTaskManagementService) activeMembers(q *models.Quest) []models.PartyMember {
	if q.AcceptedPartyID == nil {
		if q.AssignedFreelancerID == nil {
			return nil
		}
		return []models.PartyMember{{
			ID:     *q.AssignedFreelancerID,
			UserID: *q.AssignedFreelancerID,
			Role:   models.PartyRoleLeader,
			Status: models.MemberActive,
		}}
	}
	members, err := s.partyRepo.GetActiveMembersForParty(*q.AcceptedPartyID)
	if err != nil {
		return nil
	}
	return members
}

func displayName(u *models.User) string {
	if u == nil {
		return ""
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return u.Email
	}
	return name
}

// milestoneProgress mirrors the client TMD progress heuristic.
func milestoneProgress(status models.MilestoneStatus, tasks []models.Task) int {
	if status == models.MilestoneStatusReleased {
		return 100
	}
	if len(tasks) == 0 {
		switch status {
		case models.MilestoneStatusSubmittedForRelease:
			return 90
		case models.MilestoneStatusWorkInProgress:
			return 25
		default:
			return 0
		}
	}
	completed := 0
	for _, t := range tasks {
		if t.Status == models.TaskStatusCompleted {
			completed++
		}
	}
	return completed * 100 / len(tasks)
}

func milestoneEndDate(m *models.QuestMilestone) *time.Time {
	if m.DueDate != nil {
		return m.DueDate
	}
	return nil
}

func assignableMilestoneStatus(status models.MilestoneStatus) bool {
	switch status {
	case models.MilestoneStatusAgreed,
		models.MilestoneStatusFundingPending,
		models.MilestoneStatusFunded,
		models.MilestoneStatusWorkInProgress,
		models.MilestoneStatusSubmittedForRelease:
		return true
	}
	return false
}

// bannerStage derives the party-side banner from milestone + assignment state.
func bannerStage(milestones []models.QuestMilestone, assignments []models.MilestoneAssignment) string {
	if len(milestones) == 0 {
		return dto.PartyTMDStageInitial
	}
	assigned := make(map[uuid.UUID]bool, len(assignments))
	for _, a := range assignments {
		assigned[a.MilestoneID] = true
	}

	stage := dto.PartyTMDStageInProgress
	for _, m := range milestones {
		switch m.Status {
		case models.MilestoneStatusDraft,
			models.MilestoneStatusRevisionInProgress,
			models.MilestoneStatusRevisionRequested:
			return dto.PartyTMDStageMilestonesCreated
		}
	}
	for _, m := range milestones {
		if m.Status == models.MilestoneStatusAwaitingClientReview {
			stage = dto.PartyTMDStageAwaitingClient
			break
		}
	}
	if assignedCount := len(assigned); assignedCount == 0 {
		for _, m := range milestones {
			if m.Status == models.MilestoneStatusAgreed || m.Status == models.MilestoneStatusFundingPending {
				return dto.PartyTMDStageApprovedAssignTeam
			}
		}
	}
	return stage
}

// ------------------------------------------------------------
// Dashboard
// ------------------------------------------------------------

func (s *partyTaskManagementService) GetDashboard(userIDStr, questIDStr string) (*dto.PartyTMDDashboardResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrPartyTMDQuestNotFound
	}

	quest, err := s.questRepo.FindByID(questID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPartyTMDQuestNotFound
		}
		return nil, err
	}

	role, err := s.viewerRole(quest, userID)
	if err != nil {
		return nil, err
	}

	assignments, err := s.assignRepo.ListByQuest(questID)
	if err != nil {
		return nil, err
	}
	tasks, err := s.taskRepo.GetTasksByQuest(questID)
	if err != nil {
		return nil, err
	}
	attachmentCounts, err := s.attachRepo.CountByQuestGrouped(questID)
	if err != nil {
		return nil, err
	}
	activities, err := s.taskRepo.GetRecentActivities(questID, 10)
	if err != nil {
		return nil, err
	}

	members := s.activeMembers(quest)
	teamSize := len(members)
	if quest.AcceptedPartyID == nil {
		teamSize = 1
	}

	assignmentsByMilestone := make(map[uuid.UUID][]models.MilestoneAssignment, len(milestones(quest)))
	for _, a := range assignments {
		assignmentsByMilestone[a.MilestoneID] = append(assignmentsByMilestone[a.MilestoneID], a)
	}
	tasksByMilestone := make(map[uuid.UUID][]models.Task, len(milestones(quest)))
	for _, t := range tasks {
		tasksByMilestone[t.MilestoneID] = append(tasksByMilestone[t.MilestoneID], t)
	}

	assignedMilestones := 0
	if role == dto.PartyTMDRoleTeamLead {
		assignedMilestones = len(assignmentsByMilestone)
	} else {
		for _, a := range assignments {
			if a.MemberID == userID {
				assignedMilestones++
			}
		}
	}

	items := make([]dto.PartyTMDMilestoneItem, 0, len(quest.Milestones))
	overallProgress := 0
	breakdown := dto.PartyTMDMilestoneBreakdown{}

	for i := range quest.Milestones {
		m := quest.Milestones[i]
		mTasks := tasksByMilestone[m.ID]
		progress := milestoneProgress(m.Status, mTasks)
		overallProgress += progress

		switch {
		case m.Status == models.MilestoneStatusReleased:
			breakdown.Completed++
		case m.Status == models.MilestoneStatusWorkInProgress || m.Status == models.MilestoneStatusFunded:
			breakdown.InProgress++
		default:
			breakdown.Pending++
		}

		assignedMembers := make([]dto.PartyTMDAssignmentItem, 0, len(assignmentsByMilestone[m.ID]))
		isAssignedToCaller := false
		for _, a := range assignmentsByMilestone[m.ID] {
			if a.MemberID == userID {
				isAssignedToCaller = true
			}
			assignedMembers = append(assignedMembers, dto.PartyTMDAssignmentItem{
				ID:        a.ID.String(),
				MemberID:  a.MemberID.String(),
				Name:      displayName(&a.Member),
				StartDate: a.StartDate,
				EndDate:   a.EndDate,
				Note:      a.Note,
			})
		}

		taskItems := make([]dto.PartyTMDTaskItem, 0, len(mTasks))
		for _, t := range mTasks {
			item := dto.PartyTMDTaskItem{
				ID:      t.ID.String(),
				Title:   t.Title,
				Status:  string(t.Status),
				EndDate: t.EndDate,
			}
			if t.AssigneeID != nil {
				item.AssigneeID = t.AssigneeID.String()
			}
			taskItems = append(taskItems, item)
		}

		items = append(items, dto.PartyTMDMilestoneItem{
			ID:                 m.ID.String(),
			OrderIndex:         m.OrderIndex,
			Title:              m.Title,
			Description:        m.Description,
			AmountMinor:        m.AmountMinor,
			Status:             string(m.Status),
			Progress:           progress,
			StartDate:          m.StartDate,
			EndDate:            milestoneEndDate(&m),
			AssignedMembers:    assignedMembers,
			Tasks:              taskItems,
			AttachmentsCount:   int(attachmentCounts[m.ID]),
			IsAssignedToCaller: isAssignedToCaller,
		})
	}

	if len(items) > 0 {
		overallProgress /= len(items)
	}

	activityItems := make([]dto.PartyTMDActivityItem, 0, len(activities))
	for _, a := range activities {
		activityItems = append(activityItems, dto.PartyTMDActivityItem{
			ID:          a.ID.String(),
			ActionType:  string(a.ActionType),
			Title:       a.Title,
			Description: a.Description,
			ActorID:     a.ActorID.String(),
			CreatedAt:   a.CreatedAt,
		})
	}

	return &dto.PartyTMDDashboardResponse{
		QuestID:     quest.ID.String(),
		CallerRole:  role,
		BannerStage: bannerStage(quest.Milestones, assignments),
		QuestTitle:  quest.Title,
		QuestStatus: string(quest.Status),
		ClientName:  displayName(&quest.Client),
		KPIs: dto.PartyTMDKPIs{
			ActiveQuests:       1,
			TotalMilestones:    len(quest.Milestones),
			AssignedMilestones: assignedMilestones,
			TeamMembers:        teamSize,
			OverallProgress:    overallProgress,
		},
		MilestoneBreakdown: breakdown,
		Milestones:         items,
		RecentActivity:     activityItems,
	}, nil
}

func milestones(q *models.Quest) []models.QuestMilestone { return q.Milestones }

// ------------------------------------------------------------
// Active quests
// ------------------------------------------------------------

func (s *partyTaskManagementService) GetActiveQuests(userIDStr string) (*dto.PartyTMDActiveQuestsResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	quests, err := s.questRepo.ListQuestsForPartyViewer(userID, string(models.QuestStatusInProgress))
	if err != nil {
		return nil, err
	}

	items := make([]dto.PartyTMDQuestItem, 0, len(quests))
	for i := range quests {
		q := &quests[i]
		role, roleErr := s.viewerRole(q, userID)
		if roleErr != nil {
			continue // not a team member of this quest
		}
		progress := 0
		if len(q.Milestones) > 0 {
			for _, m := range q.Milestones {
				progress += milestoneProgress(m.Status, nil)
			}
			progress /= len(q.Milestones)
		}
		items = append(items, dto.PartyTMDQuestItem{
			QuestID:        q.ID.String(),
			Title:          q.Title,
			Status:         string(q.Status),
			ClientName:     displayName(&q.Client),
			MilestoneCount: len(q.Milestones),
			Progress:       progress,
			CallerRole:     role,
		})
	}

	return &dto.PartyTMDActiveQuestsResponse{Quests: items, Total: len(items)}, nil
}

// ------------------------------------------------------------
// Members overview
// ------------------------------------------------------------

func (s *partyTaskManagementService) GetMembersOverview(userIDStr, questIDStr string) (*dto.PartyTMDMembersResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	questID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrPartyTMDQuestNotFound
	}

	quest, err := s.questRepo.FindByID(questID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPartyTMDQuestNotFound
		}
		return nil, err
	}
	if _, err := s.viewerRole(quest, userID); err != nil {
		return nil, err
	}

	assignments, err := s.assignRepo.ListByQuest(questID)
	if err != nil {
		return nil, err
	}

	byMember := make(map[uuid.UUID][]models.MilestoneAssignment)
	nameByMember := make(map[uuid.UUID]string, len(assignments))
	for _, a := range assignments {
		byMember[a.MemberID] = append(byMember[a.MemberID], a)
		if nameByMember[a.MemberID] == "" {
			nameByMember[a.MemberID] = displayName(&a.Member)
		}
	}

	partySkills := []string{}
	if quest.AcceptedPartyID != nil {
		if skills, err := s.partyRepo.GetSkills(*quest.AcceptedPartyID); err == nil {
			for _, sk := range skills {
				partySkills = append(partySkills, sk.SkillName)
			}
		}
	}

	members := s.activeMembers(quest)
	items := make([]dto.PartyTMDMemberItem, 0, len(members))
	unassigned := 0

	for i := range members {
		m := &members[i]
		name := displayName(&m.User)
		if name == "" {
			// Solo quests synthesize the member row without a preloaded User.
			name = nameByMember[m.UserID]
		}
		item := dto.PartyTMDMemberItem{
			MemberID:           m.ID.String(),
			UserID:             m.UserID.String(),
			Name:               name,
			Rank:               m.User.Rank,
			Skills:             partySkills,
			AssignmentStatus:   "unassigned",
			AssignedMilestones: []dto.PartyTMDMilestoneRef{},
		}
		if m.Role == models.PartyRoleLeader {
			item.Role = dto.PartyTMDRoleTeamLead
		} else {
			item.Role = dto.PartyTMDRoleTeamMember
		}
		for _, a := range byMember[m.UserID] {
			for _, ms := range quest.Milestones {
				if ms.ID == a.MilestoneID {
					item.AssignedMilestones = append(item.AssignedMilestones, dto.PartyTMDMilestoneRef{
						ID:     ms.ID.String(),
						Title:  ms.Title,
						Status: string(ms.Status),
					})
					break
				}
			}
		}
		if len(item.AssignedMilestones) > 0 {
			item.AssignmentStatus = "assigned"
		} else {
			unassigned++
		}
		items = append(items, item)
	}

	assignedMilestones := len(distinctMilestoneIDs(assignments))

	return &dto.PartyTMDMembersResponse{
		TotalMembers:       len(members),
		UnassignedMembers:  unassigned,
		TotalMilestones:    len(quest.Milestones),
		AssignedMilestones: assignedMilestones,
		Members:            items,
	}, nil
}

func distinctMilestoneIDs(assignments []models.MilestoneAssignment) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(assignments))
	ids := make([]uuid.UUID, 0, len(assignments))
	for _, a := range assignments {
		if _, ok := seen[a.MilestoneID]; ok {
			continue
		}
		seen[a.MilestoneID] = struct{}{}
		ids = append(ids, a.MilestoneID)
	}
	return ids
}

// ------------------------------------------------------------
// Assignment
// ------------------------------------------------------------

func (s *partyTaskManagementService) AssignMilestone(userIDStr, milestoneIDStr string, req dto.AssignMilestoneBody) (*dto.AssignMilestoneResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrPartyTMDMilestoneNotFound
	}
	if len(req.MemberIDs) == 0 {
		return nil, ErrPartyTMDNoMembers
	}
	if err := validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	milestone, quest, err := s.taskRepo.GetMilestoneWithQuest(milestoneID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPartyTMDMilestoneNotFound
		}
		return nil, err
	}

	// Only the assigned freelancer (team lead) assigns work.
	if quest.AssignedFreelancerID == nil || *quest.AssignedFreelancerID != userID {
		return nil, ErrPartyTMDForbidden
	}
	if !assignableMilestoneStatus(milestone.Status) {
		return nil, ErrPartyTMDAssignmentState
	}

	memberIDs := make([]uuid.UUID, 0, len(req.MemberIDs))
	seenMembers := make(map[uuid.UUID]struct{}, len(req.MemberIDs))
	for _, raw := range req.MemberIDs {
		memberID, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, ErrPartyTMDMemberInvalid
		}
		if _, dup := seenMembers[memberID]; dup {
			continue // unique index is (milestone_id, member_id)
		}
		allowed, err := s.isAssignableMember(quest, memberID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrPartyTMDMemberInvalid
		}
		seenMembers[memberID] = struct{}{}
		memberIDs = append(memberIDs, memberID)
	}

	rows := make([]models.MilestoneAssignment, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		rows = append(rows, models.MilestoneAssignment{
			MilestoneID:  milestone.ID,
			MemberID:     memberID,
			AssignedByID: userID,
			StartDate:    req.StartDate,
			EndDate:      req.EndDate,
			Note:         req.Note,
		})
	}

	if err := s.assignRepo.ReplaceForMilestone(milestone.ID, rows); err != nil {
		return nil, err
	}

	// Activity feed entry (best effort, matching the rest of the TMD flow).
	note := "Team assigned to milestone"
	_ = s.taskRepo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     userID,
		ActionType:  models.ActivityMilestoneAssigned,
		Title:       note,
		Description: milestone.Title,
	})

	saved, err := s.assignRepo.ListByMilestone(milestone.ID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.PartyTMDAssignmentItem, 0, len(saved))
	for i := range saved {
		a := &saved[i]
		items = append(items, dto.PartyTMDAssignmentItem{
			ID:        a.ID.String(),
			MemberID:  a.MemberID.String(),
			UserID:    a.MemberID.String(),
			Name:      displayName(&a.Member),
			StartDate: a.StartDate,
			EndDate:   a.EndDate,
			Note:      a.Note,
		})
	}

	return &dto.AssignMilestoneResponse{
		MilestoneID: milestone.ID.String(),
		Assignments: items,
	}, nil
}

func (s *partyTaskManagementService) isAssignableMember(quest *models.Quest, userID uuid.UUID) (bool, error) {
	if quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userID {
		return true, nil
	}
	if quest.AcceptedPartyID == nil {
		return false, nil
	}
	member, err := s.partyRepo.FindMember(*quest.AcceptedPartyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return member.Status == models.MemberActive, nil
}

func (s *partyTaskManagementService) ListAssignments(userIDStr, milestoneIDStr string) (*dto.ListMilestoneAssignmentsResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidUserID
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrPartyTMDMilestoneNotFound
	}

	milestone, quest, err := s.taskRepo.GetMilestoneWithQuest(milestoneID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPartyTMDMilestoneNotFound
		}
		return nil, err
	}
	if !s.isParticipant(quest, userID) {
		return nil, ErrPartyTMDForbidden
	}

	saved, err := s.assignRepo.ListByMilestone(milestone.ID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.PartyTMDAssignmentItem, 0, len(saved))
	for i := range saved {
		a := &saved[i]
		items = append(items, dto.PartyTMDAssignmentItem{
			ID:        a.ID.String(),
			MemberID:  a.MemberID.String(),
			UserID:    a.MemberID.String(),
			Name:      displayName(&a.Member),
			StartDate: a.StartDate,
			EndDate:   a.EndDate,
			Note:      a.Note,
		})
	}

	return &dto.ListMilestoneAssignmentsResponse{
		MilestoneID: milestone.ID.String(),
		Assignments: items,
		Total:       len(items),
	}, nil
}

func validateDateRange(start, end *time.Time) error {
	if start != nil && end != nil && start.After(*end) {
		return ErrPartyTMDInvalidDateRange
	}
	return nil
}
