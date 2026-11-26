package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"
)

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }

func TestRoleForQuest(t *testing.T) {
	clientID := uuid.New()
	leadID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()
	pendingID := uuid.New()
	partyID := uuid.New()

	quest := &models.Quest{
		ClientID:             clientID,
		AssignedFreelancerID: ptrUUID(leadID),
		AcceptedPartyID:      ptrUUID(partyID),
	}

	lookup := func(_ uuid.UUID, userID uuid.UUID) (*models.PartyMember, error) {
		switch userID {
		case memberID:
			return &models.PartyMember{UserID: memberID, Status: models.MemberActive}, nil
		case leadID:
			return &models.PartyMember{UserID: leadID, Status: models.MemberActive, Role: models.PartyRoleLeader}, nil
		case pendingID:
			return &models.PartyMember{UserID: pendingID, Status: models.MemberPending}, nil
		default:
			return nil, gorm.ErrRecordNotFound
		}
	}

	tests := []struct {
		name    string
		quest   *models.Quest
		userID  uuid.UUID
		want    string
		wantErr error
	}{
		{"client is rejected", quest, clientID, "", ErrPartyTMDForbidden},
		{"assigned freelancer is team lead", quest, leadID, dto.PartyTMDRoleTeamLead, nil},
		{"active party member", quest, memberID, dto.PartyTMDRoleTeamMember, nil},
		{"outsider", quest, outsiderID, "", ErrPartyTMDForbidden},
		{
			"non-active member is rejected",
			&models.Quest{ClientID: clientID, AcceptedPartyID: ptrUUID(partyID)},
			pendingID, "", ErrPartyTMDForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := roleForQuest(tc.quest, tc.userID, lookup)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("role = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClassifyAttachment(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		size     int64
		want     string
		wantErr  error
	}{
		{"pdf", "brief.pdf", 1024, "pdf", nil},
		{"uppercase pdf", "BRIEF.PDF", 1024, "pdf", nil},
		{"png", "shot.png", 1024, "image", nil},
		{"jpeg", "shot.JPEG", 1024, "image", nil},
		{"empty file", "brief.pdf", 0, "", ErrAttachmentTooLarge},
		{"too large", "brief.pdf", utils.MaxUploadBytes + 1, "", ErrAttachmentTooLarge},
		{"rejected type", "notes.txt", 1024, "", ErrAttachmentTypeNotAllow},
		{"no extension", "notes", 1024, "", ErrAttachmentTypeNotAllow},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyAttachment(tc.filename, tc.size)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("type = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBannerStage(t *testing.T) {
	milestoneID := uuid.New()

	tests := []struct {
		name        string
		milestones  []models.QuestMilestone
		assignments []models.MilestoneAssignment
		want        string
	}{
		{"no milestones", nil, nil, dto.PartyTMDStageInitial},
		{
			"draft milestone",
			[]models.QuestMilestone{{ID: milestoneID, Status: models.MilestoneStatusDraft}},
			nil,
			dto.PartyTMDStageMilestonesCreated,
		},
		{
			"awaiting client approval",
			[]models.QuestMilestone{{ID: milestoneID, Status: models.MilestoneStatusAwaitingClientReview}},
			nil,
			dto.PartyTMDStageAwaitingClient,
		},
		{
			"agreed and unassigned",
			[]models.QuestMilestone{{ID: milestoneID, Status: models.MilestoneStatusAgreed}},
			nil,
			dto.PartyTMDStageApprovedAssignTeam,
		},
		{
			"agreed and assigned",
			[]models.QuestMilestone{{ID: milestoneID, Status: models.MilestoneStatusAgreed}},
			[]models.MilestoneAssignment{{MilestoneID: milestoneID}},
			dto.PartyTMDStageInProgress,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := bannerStage(tc.milestones, tc.assignments); got != tc.want {
				t.Fatalf("stage = %q, want %q", got, tc.want)
			}
		})
	}
}
