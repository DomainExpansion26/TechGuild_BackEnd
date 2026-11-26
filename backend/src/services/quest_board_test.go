package services

import (
	"strconv"
	"testing"
)

func formatBound(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

func TestTimelineBounds(t *testing.T) {
	cases := []struct {
		bucket   string
		min, max string
	}{
		{"under_7", "-", "6"},
		{"7_15", "7", "15"},
		{"15_30", "15", "30"},
		{"30_plus", "31", "-"},
		{" 7_15 ", "7", "15"},
		{"", "-", "-"},
		{"weekly", "-", "-"},
	}

	for _, tc := range cases {
		min, max := timelineBounds(tc.bucket)
		if got := formatBound(min); got != tc.min {
			t.Errorf("timelineBounds(%q) min = %s, want %s", tc.bucket, got, tc.min)
		}
		if got := formatBound(max); got != tc.max {
			t.Errorf("timelineBounds(%q) max = %s, want %s", tc.bucket, got, tc.max)
		}
	}
}

func TestMatchPercentage(t *testing.T) {
	cases := []struct {
		name  string
		quest []string
		user  map[string]bool
		want  int
	}{
		{"no quest skills", nil, map[string]bool{"go": true}, 0},
		{"no user skills", []string{"go"}, map[string]bool{}, 0},
		{"half match", []string{"go", "react", "sql", "figma"}, map[string]bool{"go": true, "sql": true}, 50},
		{"case and space insensitive", []string{" Go ", "REACT"}, map[string]bool{"go": true, "react": true}, 100},
		{"partial credit", []string{"go", "react", "sql"}, map[string]bool{"go": true}, 33},
	}

	for _, tc := range cases {
		if got := matchPercentage(tc.quest, tc.user); got != tc.want {
			t.Errorf("%s: matchPercentage = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	if got := splitCSV("   "); got != nil {
		t.Errorf("splitCSV(blank) = %v, want nil", got)
	}
	got := splitCSV(" go, react ,,figma ")
	if len(got) != 3 || got[0] != "go" || got[1] != "react" || got[2] != "figma" {
		t.Errorf("splitCSV = %v, want [go react figma]", got)
	}
}
