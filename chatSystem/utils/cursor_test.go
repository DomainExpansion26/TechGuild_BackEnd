package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorEncodeDecode(t *testing.T) {
	testID := uuid.New()
	testTime := time.Date(2026, 11, 1, 12, 0, 0, 123456000, time.UTC)

	cursor := EncodeCursor(testTime, testID)
	if cursor == "" {
		t.Fatal("expected non-empty cursor string")
	}

	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("unexpected error decoding cursor: %v", err)
	}

	if decoded == nil {
		t.Fatal("expected non-nil decoded cursor")
	}

	if decoded.ID != testID {
		t.Errorf("expected ID %s, got %s", testID, decoded.ID)
	}

	if !decoded.CreatedAt.Equal(testTime) {
		t.Errorf("expected time %s, got %s", testTime, decoded.CreatedAt)
	}
}

func TestCursorEmpty(t *testing.T) {
	decoded, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("expected nil error for empty cursor, got: %v", err)
	}
	if decoded != nil {
		t.Fatal("expected nil decoded cursor for empty string")
	}
}

func TestCursorInvalid(t *testing.T) {
	_, err := DecodeCursor("invalid-base64-!@#$")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

