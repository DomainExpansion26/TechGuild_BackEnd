package services

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

var (
	ErrAttachmentNotFound     = errors.New("attachment not found")
	ErrAttachmentTooLarge     = errors.New("attachment exceeds the 2MB limit")
	ErrAttachmentTypeNotAllow = errors.New("only PDF and image attachments are allowed")
	ErrPartyTMDMilestoneGone  = errors.New("milestone not found")
)

// MilestoneFileUploader isolates Cloudinary so the service stays testable.
type MilestoneFileUploader interface {
	UploadPDF(file multipart.File, filename string) (string, error)
	UploadImage(file multipart.File, filename string) (string, error)
}

type cloudinaryMilestoneUploader struct{}

func (cloudinaryMilestoneUploader) UploadPDF(file multipart.File, filename string) (string, error) {
	return utils.UploadPDFToCloudinary(file, filename)
}

func (cloudinaryMilestoneUploader) UploadImage(file multipart.File, filename string) (string, error) {
	return utils.UploadImageToCloudinary(file, filename)
}

type MilestoneAttachmentService interface {
	Upload(userIDStr, questIDStr, milestoneIDStr string, file multipart.File, fileName string, size int64) (*dto.MilestoneAttachmentResponse, error)
	List(userIDStr, questIDStr, milestoneIDStr string) (*dto.ListMilestoneAttachmentsResponse, error)
	Delete(userIDStr, questIDStr, milestoneIDStr, attachmentIDStr string) (*dto.DeleteMilestoneAttachmentResponse, error)
}

type milestoneAttachmentService struct {
	taskRepo   repository.TaskManagementRepository
	partyRepo  *repository.PartyRepository
	attachRepo repository.MilestoneAttachmentRepository
	uploader   MilestoneFileUploader
}

func NewMilestoneAttachmentService() MilestoneAttachmentService {
	return &milestoneAttachmentService{
		taskRepo:   repository.NewTaskManagementRepository(),
		partyRepo:  repository.NewPartyRepository(),
		attachRepo: repository.NewMilestoneAttachmentRepository(),
		uploader:   cloudinaryMilestoneUploader{},
	}
}

func NewMilestoneAttachmentServiceWith(uploader MilestoneFileUploader) MilestoneAttachmentService {
	svc := NewMilestoneAttachmentService().(*milestoneAttachmentService)
	svc.uploader = uploader
	return svc
}

// classifyAttachment validates size and extension before any upload happens, so
// rejected files never reach Cloudinary.
func classifyAttachment(filename string, size int64) (string, error) {
	if size <= 0 || size > utils.MaxUploadBytes {
		return "", ErrAttachmentTooLarge
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return "pdf", nil
	case ".png", ".jpg", ".jpeg", ".webp":
		return "image", nil
	}
	return "", ErrAttachmentTypeNotAllow
}

// participantOf loads a milestone + quest and authorizes the caller as client,
// team lead, or active party member.
func (s *milestoneAttachmentService) participantOf(userIDStr, questIDStr, milestoneIDStr string) (uuid.UUID, *models.Quest, *models.QuestMilestone, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, nil, nil, ErrInvalidUserID
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return uuid.Nil, nil, nil, ErrPartyTMDMilestoneGone
	}

	milestone, quest, err := s.taskRepo.GetMilestoneWithQuest(milestoneID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, nil, nil, ErrPartyTMDMilestoneGone
		}
		return uuid.Nil, nil, nil, err
	}
	// The path's quest id must match the milestone's quest.
	if questIDStr != "" {
		questID, err := uuid.Parse(questIDStr)
		if err != nil || questID != quest.ID {
			return uuid.Nil, nil, nil, ErrPartyTMDMilestoneGone
		}
	}

	ok := quest.ClientID == userID
	if !ok && quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userID {
		ok = true
	}
	if !ok && quest.AcceptedPartyID != nil {
		member, err := s.partyRepo.FindMember(*quest.AcceptedPartyID, userID)
		ok = err == nil && member.Status == models.MemberActive
	}
	if !ok {
		return uuid.Nil, nil, nil, ErrPartyTMDForbidden
	}

	return userID, quest, milestone, nil
}

func (s *milestoneAttachmentService) Upload(userIDStr, questIDStr, milestoneIDStr string, file multipart.File, fileName string, size int64) (*dto.MilestoneAttachmentResponse, error) {
	userID, quest, milestone, err := s.participantOf(userIDStr, questIDStr, milestoneIDStr)
	if err != nil {
		return nil, err
	}

	fileType, err := classifyAttachment(fileName, size)
	if err != nil {
		return nil, err
	}

	var fileURL string
	if fileType == "pdf" {
		fileURL, err = s.uploader.UploadPDF(file, fileName)
	} else {
		fileURL, err = s.uploader.UploadImage(file, fileName)
	}
	if err != nil {
		return nil, err
	}

	attachment := &models.MilestoneAttachment{
		MilestoneID:   milestone.ID,
		QuestID:       quest.ID,
		UploadedByID:  userID,
		FileURL:       fileURL,
		FileName:      fileName,
		FileType:      fileType,
		FileSizeBytes: size,
	}
	if err := s.attachRepo.Create(attachment); err != nil {
		return nil, err
	}

	_ = s.taskRepo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     userID,
		ActionType:  models.ActivityMilestoneAttachmentAdded,
		Title:       "Attachment uploaded",
		Description: fileName,
	})

	return toAttachmentResponse(attachment), nil
}

func (s *milestoneAttachmentService) List(userIDStr, questIDStr, milestoneIDStr string) (*dto.ListMilestoneAttachmentsResponse, error) {
	if _, _, milestone, err := s.participantOf(userIDStr, questIDStr, milestoneIDStr); err != nil {
		return nil, err
	} else {
		rows, err := s.attachRepo.ListByMilestone(milestone.ID)
		if err != nil {
			return nil, err
		}
		items := make([]dto.MilestoneAttachmentResponse, 0, len(rows))
		for i := range rows {
			items = append(items, *toAttachmentResponse(&rows[i]))
		}
		return &dto.ListMilestoneAttachmentsResponse{Attachments: items, Total: len(items)}, nil
	}
}

func (s *milestoneAttachmentService) Delete(userIDStr, questIDStr, milestoneIDStr, attachmentIDStr string) (*dto.DeleteMilestoneAttachmentResponse, error) {
	userID, quest, milestone, err := s.participantOf(userIDStr, questIDStr, milestoneIDStr)
	if err != nil {
		return nil, err
	}

	attachmentID, err := uuid.Parse(attachmentIDStr)
	if err != nil {
		return nil, ErrAttachmentNotFound
	}
	attachment, err := s.attachRepo.FindByIDAndMilestone(attachmentID, milestone.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAttachmentNotFound
		}
		return nil, err
	}

	// Uploader, the client, and the team lead may delete.
	allowed := attachment.UploadedByID == userID ||
		quest.ClientID == userID ||
		(quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userID)
	if !allowed {
		return nil, ErrPartyTMDForbidden
	}

	if err := s.attachRepo.Delete(attachment.ID); err != nil {
		return nil, err
	}

	_ = s.taskRepo.CreateActivity(&models.QuestActivity{
		QuestID:     quest.ID,
		MilestoneID: &milestone.ID,
		ActorID:     userID,
		ActionType:  models.ActivityMilestoneAttachmentDeleted,
		Title:       "Attachment deleted",
		Description: attachment.FileName,
	})

	return &dto.DeleteMilestoneAttachmentResponse{
		Message:      "attachment deleted",
		AttachmentID: attachment.ID.String(),
	}, nil
}

func toAttachmentResponse(a *models.MilestoneAttachment) *dto.MilestoneAttachmentResponse {
	return &dto.MilestoneAttachmentResponse{
		ID:            a.ID.String(),
		MilestoneID:   a.MilestoneID.String(),
		QuestID:       a.QuestID.String(),
		FileName:      a.FileName,
		FileURL:       a.FileURL,
		FileType:      a.FileType,
		FileSizeBytes: a.FileSizeBytes,
		UploadedByID:  a.UploadedByID.String(),
		CreatedAt:     a.CreatedAt,
	}
}
