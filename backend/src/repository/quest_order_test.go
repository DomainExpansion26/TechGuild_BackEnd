package repository

import "testing"

func TestQuestOrderClause(t *testing.T) {
	cases := []struct {
		sort string
		want string
	}{
		{"", "created_at DESC, id DESC"},
		{"newest", "created_at DESC, id DESC"},
		{"budget_desc", "budget_minor DESC, id DESC"},
		{"budget_asc", "budget_minor ASC, id DESC"},
		{"emergency_first", "CASE WHEN quest_type = 'emergency' THEN 0 ELSE 1 END ASC, created_at DESC, id DESC"},
		{"unknown", "created_at DESC, id DESC"},
	}

	for _, tc := range cases {
		if got := questOrderClause(tc.sort); got != tc.want {
			t.Errorf("questOrderClause(%q) = %q, want %q", tc.sort, got, tc.want)
		}
	}
}
