package middleware

import (
	"context"
	"errors"

	"techguild-chat/models"
	"techguild-chat/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAccessDeniedNotMember = errors.New("access denied: user is not a member of this channel")
)

// ValidateChannelMembership verifies if the user in context is a member of the given channel ID
func ValidateChannelMembership(ctx context.Context, db *gorm.DB, channelID string) (uuid.UUID, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	var member models.ChannelMember
	if err := db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userUUID).
		First(&member).Error; err != nil {
		return uuid.Nil, ErrAccessDeniedNotMember
	}

	return userUUID, nil
}

