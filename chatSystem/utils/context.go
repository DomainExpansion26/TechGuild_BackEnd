package utils

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type CtxKey string

const (
	UserIDKey    CtxKey = "user_id"
	SessionIDKey CtxKey = "session_id"
)

// GetUserIDFromContext retrieves the user ID string from context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("nil context")
	}

	// Try custom CtxKey
	if val, ok := ctx.Value(UserIDKey).(string); ok && val != "" {
		return val, nil
	}

	// Try string key "user_id"
	if val, ok := ctx.Value("user_id").(string); ok && val != "" {
		return val, nil
	}

	return "", errors.New("unauthorized: user id not found in context")
}

// GetUserUUIDFromContext retrieves the parsed UUID of the user from context
func GetUserUUIDFromContext(ctx context.Context) (uuid.UUID, error) {
	str, err := GetUserIDFromContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(str)
}

