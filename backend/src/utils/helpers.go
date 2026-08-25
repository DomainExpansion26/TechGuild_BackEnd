package utils

import (
	"context"
	"errors"

	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
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
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", errors.New("user is not authenticated")
	}
	return userID, nil
}
