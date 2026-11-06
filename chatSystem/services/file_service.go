package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"techguild-chat/config"
	"techguild-chat/dto"
	"techguild-chat/models"
	"techguild-chat/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FileService interface {
	UploadFileContent(ctx context.Context, userID uuid.UUID, channelID string, filename string, content []byte, size int64, contentType string) (*dto.UploadFileResponse, error)
	UploadFileDirect(ctx context.Context, userID uuid.UUID, channelID string, req dto.UploadFileDirectRequest) (*dto.UploadFileResponse, error)
	GeneratePresignedUploadURL(ctx context.Context, userID uuid.UUID, channelID string, req dto.PresignedURLRequest) (*dto.PresignedURLResponse, error)
}

type fileService struct {
	cfg          *config.Config
	db           *gorm.DB
	centrifugo   CentrifugoService
	virusScanner utils.VirusScanHook
}

func NewFileService(cfg *config.Config, db *gorm.DB, centrifugo CentrifugoService) FileService {
	return &fileService{
		cfg:          cfg,
		db:           db,
		centrifugo:   centrifugo,
		virusScanner: &utils.DefaultVirusScanner{},
	}
}

// UploadFileContent handles file content upload, validates extension/size, scans, stores, and broadcasts
func (s *fileService) UploadFileContent(ctx context.Context, userID uuid.UUID, channelID string, filename string, content []byte, size int64, contentType string) (*dto.UploadFileResponse, error) {
	if len(content) == 0 {
		return nil, utils.ErrEmptyFile
	}

	if size <= 0 {
		size = int64(len(content))
	}

	// 1. Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	// 2. Validate file type and size
	if err := utils.ValidateFile(filename, size, s.cfg.MaxFileSizeBytes); err != nil {
		return nil, err
	}

	// 3. Virus scanning hook (flagged open item in PDF)
	isClean, threatDetails, err := s.virusScanner.ScanFile(filename, content)
	if err != nil || !isClean {
		log.Printf("[FileService] Virus scan rejected file %s: %s", filename, threatDetails)
		return nil, fmt.Errorf("file rejected by virus scan: %s", threatDetails)
	}

	// 4. Store file (R2 or persistent uploads folder)
	fileExt := strings.ToLower(filepath.Ext(filename))
	uniqueFileName := fmt.Sprintf("%s-%s%s", uuid.New().String(), filepath.Base(filename), fileExt)
	fileURL := s.persistFile(uniqueFileName, content)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 5. Record message in communication_records
	return s.createFileShareRecord(ctx, userID, channelID, filename, fileURL, size, contentType)
}

// UploadFileDirect handles file share when file URL was already generated via R2 presigned upload
func (s *fileService) UploadFileDirect(ctx context.Context, userID uuid.UUID, channelID string, req dto.UploadFileDirectRequest) (*dto.UploadFileResponse, error) {
	// 1. Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	// 2. Validate file type and size
	if err := utils.ValidateFile(req.FileName, req.FileSize, s.cfg.MaxFileSizeBytes); err != nil {
		return nil, err
	}

	fileURL := req.FileURL
	if fileURL == "" {
		fileURL = fmt.Sprintf("%s/%s/%s", s.cfg.R2PublicURL, channelID, req.FileName)
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return s.createFileShareRecord(ctx, userID, channelID, req.FileName, fileURL, req.FileSize, contentType)
}

// GeneratePresignedUploadURL generates an upload target URL for R2
func (s *fileService) GeneratePresignedUploadURL(ctx context.Context, userID uuid.UUID, channelID string, req dto.PresignedURLRequest) (*dto.PresignedURLResponse, error) {
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	if err := utils.ValidateFile(req.FileName, req.FileSize, s.cfg.MaxFileSizeBytes); err != nil {
		return nil, err
	}

	fileExt := strings.ToLower(filepath.Ext(req.FileName))
	objectKey := fmt.Sprintf("chats/%s/%s%s", channelID, uuid.New().String(), fileExt)

	uploadURL := fmt.Sprintf("%s/%s/%s", s.cfg.R2PublicURL, s.cfg.R2BucketName, objectKey)
	publicURL := fmt.Sprintf("%s/%s", s.cfg.R2PublicURL, objectKey)

	return &dto.PresignedURLResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		ExpiresIn: 900,
	}, nil
}

func (s *fileService) createFileShareRecord(ctx context.Context, userID uuid.UUID, channelID string, fileName, fileURL string, fileSize int64, contentType string) (*dto.UploadFileResponse, error) {
	var channel models.ChatChannel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&channel).Error; err != nil {
		return nil, ErrChannelNotFound
	}

	now := time.Now().UTC()
	metaJSON := datatypes.JSON([]byte(fmt.Sprintf(`{"file_name":%q,"file_size":%d,"content_type":%q}`, fileName, fileSize, contentType)))

	record := models.CommunicationRecord{
		ID:          uuid.New(),
		QuestID:     channel.QuestID,
		PartyID:     channel.PartyID,
		FromUserID:  userID,
		ChannelID:   channelID,
		ChannelType: channel.Type,
		ContentRef:  &fileURL,
		ContentText: fileName,
		MessageType: models.MessageTypeFileShare,
		Metadata:    metaJSON,
		IsPinned:    false,
		Deleted:     false,
		CreatedAt:   now,
	}

	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to persist file record: %w", err)
	}

	senderName := s.getUserFullName(ctx, userID)

	msgResp := dto.MessageResponse{
		ID:          record.ID,
		ChannelID:   record.ChannelID,
		ChannelType: string(record.ChannelType),
		FromUserID:  record.FromUserID,
		SenderName:  senderName,
		ContentRef:  record.ContentRef,
		ContentText: record.ContentText,
		MessageType: string(record.MessageType),
		CreatedAt:   record.CreatedAt,
		Metadata: map[string]interface{}{
			"file_name":    fileName,
			"file_size":    fileSize,
			"content_type": contentType,
		},
	}

	// Real-time Centrifugo delivery
	event := map[string]interface{}{
		"event":   "message.created",
		"message": msgResp,
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return &dto.UploadFileResponse{
		MessageID:   record.ID.String(),
		FileName:    fileName,
		FileURL:     fileURL,
		FileSize:    fileSize,
		ContentType: contentType,
		Message:     msgResp,
	}, nil
}

func (s *fileService) persistFile(uniqueFileName string, content []byte) string {
	uploadsDir := "./uploads/chat"
	_ = os.MkdirAll(uploadsDir, 0755)

	filePath := filepath.Join(uploadsDir, uniqueFileName)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		log.Printf("[FileService] Warning: could not write local copy of %s: %v", uniqueFileName, err)
	}

	if s.cfg.R2PublicURL != "" {
		return fmt.Sprintf("%s/uploads/chat/%s", strings.TrimSuffix(s.cfg.R2PublicURL, "/"), uniqueFileName)
	}
	return fmt.Sprintf("/uploads/chat/%s", uniqueFileName)
}

func (s *fileService) getUserFullName(ctx context.Context, userID uuid.UUID) string {
	type userSimple struct {
		FirstName string
		LastName  string
	}
	var u userSimple
	_ = s.db.WithContext(ctx).Table("users").Select("first_name, last_name").Where("id = ?", userID).Scan(&u)
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}
