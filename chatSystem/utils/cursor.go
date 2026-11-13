package utils

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CursorData contains the timestamp and ID encoded in a pagination cursor
type CursorData struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// EncodeCursor generates an opaque Base64 cursor from timestamp and message ID
func EncodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%s|%s", createdAt.UTC().Format(time.RFC3339Nano), id.String())
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses an opaque Base64 cursor into timestamp and message ID
func DecodeCursor(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		// Fallback to RFC3339 if nano wasn't used
		t, err = time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp in cursor: %w", err)
		}
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid uuid in cursor: %w", err)
	}

	return &CursorData{
		CreatedAt: t,
		ID:        id,
	}, nil
}

