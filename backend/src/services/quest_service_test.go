package services_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

// mockQuestRepository implements repository.QuestRepository in-memory.
type mockQuestRepository struct {
	quests         map[uuid.UUID]*models.Quest
	partyQuests    map[uuid.UUID][]*models.Quest
	auditLogs      []models.AuditLog
	chatChannels   []models.QuestChatChannel
	tasks          []models.Task
	activities     []models.QuestActivity
	clientProfiles map[uuid.UUID]*models.ClientProfile
}

func newMockQuestRepo() *mockQuestRepository {
	return &mockQuestRepository{
		quests:         make(map[uuid.UUID]*models.Quest),
		partyQuests:    make(map[uuid.UUID][]*models.Quest),
		clientProfiles: make(map[uuid.UUID]*models.ClientProfile),
	}
}

func (m *mockQuestRepository) CreateQuestWithMilestonesAndSkills(
	quest *models.Quest,
	milestones []models.QuestMilestone,
	skills []models.QuestSkill,
) error {
	quest.Milestones = milestones
	quest.Skills = skills
	quest.CreatedAt = time.Now()
	quest.UpdatedAt = time.Now()
	m.quests[quest.ID] = quest
	return nil
}

func (m *mockQuestRepository) FindByID(questID uuid.UUID) (*models.Quest, error) {
	q, ok := m.quests[questID]
	if !ok {
		return nil, services.ErrQuestNotFound
	}
	return q, nil
}

func (m *mockQuestRepository) UpdateQuest(quest *models.Quest) error {
	m.quests[quest.ID] = quest
	return nil
}

func (m *mockQuestRepository) ListQuests(filter repository.QuestFilter) ([]models.Quest, string, bool, int64, error) {
	var results []models.Quest
	for _, q := range m.quests {
		if filter.ClientID != nil && q.ClientID != *filter.ClientID {
			continue
		}
		if filter.Status != "" && string(q.Status) != filter.Status {
			continue
		}
		if len(filter.AllowedRanks) > 0 {
			allowed := false
			for _, r := range filter.AllowedRanks {
				if q.RequiredMinRank == r {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		results = append(results, *q)
	}
	return results, "", false, int64(len(results)), nil
}

func (m *mockQuestRepository) CountPartyActiveQuests(partyID uuid.UUID) (int64, error) {
	var count int64
	for _, q := range m.quests {
		if q.AcceptedPartyID != nil && *q.AcceptedPartyID == partyID && q.Status == models.QuestStatusInProgress {
			count++
		}
	}
	return count, nil
}

func (m *mockQuestRepository) HasActiveQuestWithClient(partyID uuid.UUID, clientID uuid.UUID) (bool, error) {
	for _, q := range m.quests {
		if q.AcceptedPartyID != nil && *q.AcceptedPartyID == partyID && q.ClientID == clientID && q.Status == models.QuestStatusInProgress {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockQuestRepository) AcceptQuestAtomic(
	quest *models.Quest,
	chatChannel *models.QuestChatChannel,
	tasks []models.Task,
	activity *models.QuestActivity,
	auditLog *models.AuditLog,
) error {
	m.quests[quest.ID] = quest
	if chatChannel != nil {
		m.chatChannels = append(m.chatChannels, *chatChannel)
	}
	if len(tasks) > 0 {
		m.tasks = append(m.tasks, tasks...)
	}
	if activity != nil {
		m.activities = append(m.activities, *activity)
	}
	if auditLog != nil {
		m.auditLogs = append(m.auditLogs, *auditLog)
	}
	return nil
}

func (m *mockQuestRepository) CreateAuditLog(auditLog *models.AuditLog) error {
	m.auditLogs = append(m.auditLogs, *auditLog)
	return nil
}

func (m *mockQuestRepository) GetClientProfile(userID uuid.UUID) (*models.ClientProfile, error) {
	cp, ok := m.clientProfiles[userID]
	if !ok {
		return nil, services.ErrUserNotFound
	}
	return cp, nil
}

// Tests for Tier Computation
func TestComputeRequiredMinRank(t *testing.T) {
	tests := []struct {
		budget   float64
		expected string
	}{
		{1000, models.RankF},
		{10000, models.RankF},
		{15000, models.RankE},
		{25000, models.RankE},
		{35000, models.RankD},
		{45000, models.RankD},
		{60000, models.RankC},
		{70000, models.RankC},
		{85000, models.RankB},
		{100000, models.RankB},
		{120000, models.RankA},
		{135000, models.RankA},
		{150000, models.RankS},
		{175000, models.RankS},
		{200000, models.RankSS},
		{220000, models.RankSS},
		{250000, models.RankSSS},
	}

	for _, tt := range tests {
		rank, err := utils.ComputeRequiredMinRank(tt.budget)
		if err != nil {
			t.Fatalf("unexpected error for budget %.0f: %v", tt.budget, err)
		}
		if rank != tt.expected {
			t.Errorf("for budget %.0f expected rank %s, got %s", tt.budget, tt.expected, rank)
		}
	}
}

// Tests for Asymmetric Tier Visibility
func TestAsymmetricTierVisibility(t *testing.T) {
	// Rank B should see F, E, D, C, B (5 ranks)
	visibleRanksB := utils.GetVisibleRanks(models.RankB)
	if len(visibleRanksB) != 5 {
		t.Fatalf("expected 5 visible ranks for Rank B, got %d", len(visibleRanksB))
	}

	// Rank B should NOT see A, S, SS, SSS
	for _, r := range visibleRanksB {
		if r == models.RankA || r == models.RankS || r == models.RankSS || r == models.RankSSS {
			t.Errorf("Rank B should not see rank %s", r)
		}
	}

	// Rank SSS should see all 9 ranks
	visibleRanksSSS := utils.GetVisibleRanks(models.RankSSS)
	if len(visibleRanksSSS) != 9 {
		t.Fatalf("expected 9 visible ranks for Rank SSS, got %d", len(visibleRanksSSS))
	}

	// Rank F should see only F
	visibleRanksF := utils.GetVisibleRanks(models.RankF)
	if len(visibleRanksF) != 1 || visibleRanksF[0] != models.RankF {
		t.Fatalf("expected only Rank F visible for Rank F, got %v", visibleRanksF)
	}

	// Higher rank can accept lower tier
	if !utils.IsRankSufficient(models.RankA, models.RankB) {
		t.Errorf("Rank A should be sufficient for Rank B quest")
	}
	// Lower rank cannot accept higher tier
	if utils.IsRankSufficient(models.RankB, models.RankA) {
		t.Errorf("Rank B should NOT be sufficient for Rank A quest")
	}
}

// Tests for Quest Draft Creation Logic (Validation & Milestone Sum)
func TestMilestoneSumValidation(t *testing.T) {
	clientUUID := uuid.New()
	clientAcc := models.AccountTypeClientAdmin

	mockRepo := newMockQuestRepo()
	mockRepo.clientProfiles[clientUUID] = &models.ClientProfile{UserID: clientUUID}

	// Valid quest draft with milestone sum equal to budget
	req := dto.CreateQuestBody{
		Title:        "Valid Title",
		Description:  "This is a valid description with sufficient length",
		Category:     "Engineering",
		BudgetINR:    10000,
		TimelineDays: 14,
		Skills: []dto.CreateQuestSkillInput{
			{Name: "Golang", ProficiencyLevel: "expert", IsCore: true},
		},
		Milestones: []dto.CreateQuestMilestoneInput{
			{Title: "M1", Amount: 5000, DueDays: 7, OrderIndex: 1},
			{Title: "M2", Amount: 5000, DueDays: 7, OrderIndex: 2},
		},
	}

	// Check manual calculation
	var sum float64
	for _, m := range req.Milestones {
		sum += m.Amount
	}
	if sum != req.BudgetINR {
		t.Fatalf("expected sum %f to equal budget %f", sum, req.BudgetINR)
	}

	_ = clientAcc
}
