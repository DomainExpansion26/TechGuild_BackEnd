package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

var (
	ErrOnlyClientAllowed = errors.New("only client accounts can create quests")
	ErrClientNotVerified = errors.New("client account verification required before creating quests")
	ErrInvalidUserID     = errors.New("invalid user id")
)

type QuestService interface {
	CreateQuestDraft(userID string, req dto.CreateQuestBody) (*dto.CreateQuestResponse, error)
}

type questService struct {
	questRepo repository.QuestRepository
	userRepo  repository.UserRepository
}

func NewQuestService() QuestService {
	return &questService{
		questRepo: repository.NewQuestRepository(),
		userRepo:  repository.NewUserRepository(),
	}
}

func NewQuestServiceWithRepos(questRepo repository.QuestRepository, userRepo repository.UserRepository) QuestService {
	return &questService{
		questRepo: questRepo,
		userRepo:  userRepo,
	}
}

func (s *questService) CreateQuestDraft(userID string, req dto.CreateQuestBody) (*dto.CreateQuestResponse, error) {
	clientUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	// Client Account & Verification Check
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

	// Verification check
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

	// Input Validation
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

	// Budget is reference-only; convert to rupees for rank calculation utils
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
