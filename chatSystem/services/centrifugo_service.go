package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"techguild-chat/config"
	"techguild-chat/dto"
	"techguild-chat/models"
	"techguild-chat/utils"

	"gorm.io/gorm"
)

type CentrifugoService interface {
	GenerateConnectionToken(userID string) (string, int64, error)
	Publish(ctx context.Context, channel string, data interface{}) error
	CheckSubscribePermission(ctx context.Context, userIDStr string, channel string) (*dto.CentrifugoSubscribeProxyResponse, error)
}

type centrifugoService struct {
	cfg        *config.Config
	db         *gorm.DB
	httpClient *http.Client
}

func NewCentrifugoService(cfg *config.Config, db *gorm.DB) CentrifugoService {
	return &centrifugoService{
		cfg: cfg,
		db:  db,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GenerateConnectionToken issues short-lived connection JWT for Centrifugo WebSocket
func (s *centrifugoService) GenerateConnectionToken(userID string) (string, int64, error) {
	return utils.GenerateCentrifugoToken(
		userID,
		s.cfg.CentrifugoTokenHMACSecretKey,
		s.cfg.CentrifugoTokenTTL,
	)
}

// Publish broadcasts event data to Centrifugo clients on the specified channel
func (s *centrifugoService) Publish(ctx context.Context, channel string, data interface{}) error {
	if s.cfg.CentrifugoAPIURL == "" || s.cfg.CentrifugoAPIKey == "" {
		log.Printf("[Centrifugo] API URL or Key not set, skipping publish for channel %s", channel)
		return nil
	}

	publishPayload := map[string]interface{}{
		"channel": channel,
		"data":    data,
	}

	bodyBytes, err := json.Marshal(publishPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal centrifugo payload: %w", err)
	}

	apiURL := s.cfg.CentrifugoAPIURL
	if !strings.HasSuffix(apiURL, "/publish") {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/publish"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create centrifugo request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "apikey "+s.cfg.CentrifugoAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Log warning rather than failing user action if real-time push fails
		log.Printf("[Centrifugo] Error publishing to channel %s: %v", channel, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[Centrifugo] Publish status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CheckSubscribePermission verifies if the connecting user is an active member of the channel
func (s *centrifugoService) CheckSubscribePermission(ctx context.Context, userIDStr string, channel string) (*dto.CentrifugoSubscribeProxyResponse, error) {
	// Normalize channel name (e.g., "quest:uuid" -> "uuid", "dm:uuid" -> "uuid", "channel_id")
	channelID := channel
	if parts := strings.SplitN(channel, ":", 2); len(parts) == 2 {
		channelID = parts[1]
	}

	var member models.ChannelMember
	err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userIDStr).
		First(&member).Error

	if err != nil {
		// Also check with raw channel if different
		if channelID != channel {
			err = s.db.WithContext(ctx).
				Where("channel_id = ? AND user_id = ?", channel, userIDStr).
				First(&member).Error
		}
	}

	if err != nil {
		log.Printf("[Centrifugo Proxy] Denying access to channel %s for user %s: not a member", channel, userIDStr)
		return &dto.CentrifugoSubscribeProxyResponse{
			Error: &dto.CentrifugoProxyError{
				Code:    1000,
				Message: "Permission denied: user is not a member of this channel",
			},
		}, nil
	}

	// Permission granted
	return &dto.CentrifugoSubscribeProxyResponse{
		Result: &dto.CentrifugoProxyResult{},
	}, nil
}

