package services

import (
	"errors"
	"testing"
)

func TestGuildhallPollOptionsFromMap(t *testing.T) {
	tests := []struct {
		name    string
		details map[string]any
		want    []string
		wantErr bool
	}{
		{
			name:    "trims valid options",
			details: map[string]any{"options": []any{" PostgreSQL ", "MySQL"}},
			want:    []string{"PostgreSQL", "MySQL"},
		},
		{
			name:    "requires at least two options",
			details: map[string]any{"options": []any{"PostgreSQL"}},
			wantErr: true,
		},
		{
			name:    "rejects duplicate options ignoring case",
			details: map[string]any{"options": []any{"PostgreSQL", " postgresql "}},
			wantErr: true,
		},
		{
			name:    "rejects blank option",
			details: map[string]any{"options": []any{"PostgreSQL", "  "}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := guildhallPollOptionsFromMap(tt.details)
			if tt.wantErr {
				if !errors.Is(err, ErrGuildhallInvalidPoll) {
					t.Fatalf("expected invalid poll error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got options %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got options %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestFindGuildhallPollOption(t *testing.T) {
	got, err := findGuildhallPollOption([]string{"PostgreSQL", "MySQL"}, "  POSTGRESQL ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "PostgreSQL" {
		t.Fatalf("got %q, want canonical option %q", got, "PostgreSQL")
	}

	if _, err := findGuildhallPollOption([]string{"PostgreSQL", "MySQL"}, "SQLite"); !errors.Is(err, ErrGuildhallInvalidPoll) {
		t.Fatalf("expected invalid poll error for unknown option, got %v", err)
	}
}

func TestSameGuildhallPollOptions(t *testing.T) {
	if !sameGuildhallPollOptions(
		[]string{"PostgreSQL", "MySQL"},
		[]string{"mysql", " postgresql "},
	) {
		t.Fatal("expected same option set to be accepted regardless of order/case")
	}
	if sameGuildhallPollOptions(
		[]string{"PostgreSQL", "MySQL"},
		[]string{"PostgreSQL", "SQLite"},
	) {
		t.Fatal("expected changed options to be rejected")
	}
}
