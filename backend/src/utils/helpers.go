package utils

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetUserIDFromContext(c *gin.Context) (string, error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return "", errors.New("user is not authenticated")
	}
	userID, ok := userIDVal.(string)
	if !ok {
		return "", errors.New("invalid user id format")
	}
	return userID, nil
}

func GetUserIDFromHumaContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", errors.New("user is not authenticated")
	}
	return userID, nil
}

func GetSessionIDFromHumaContext(ctx context.Context) (uuid.UUID, error) {
	sessionIDstr, ok := ctx.Value(SessionIDKey).(string)
	if !ok || sessionIDstr == "" {
		return uuid.Nil, errors.New("session id not found in context")
	}
	return uuid.Parse(sessionIDstr)
}

// GetClientIP X-Forwarded-For
func GetClientIP(forwardedFor string) string {
	if forwardedFor == "" {
		return ""
	}
	parts := strings.Split(forwardedFor, ",")
	return strings.TrimSpace(parts[0])
}
