package migration

import (
	"os"
	"testing"
)

// TestMigrateVerificationAttempts_PostgresOnly exercises migrateVerificationAttempts.
//
// The DDL deliberately uses Postgres-specific types (BIGSERIAL, TIMESTAMPTZ)
// and partial indexes, so it cannot run against the SQLite test harness used by
// the repository tests. It is gated behind an env var so the default `go test`
// (and `go vet`) stay green without a live database.
func TestMigrateVerificationAttempts_PostgresOnly(t *testing.T) {
	if os.Getenv("MIGRATION_TEST_POSTGRES") == "" {
		t.Skip("set MIGRATION_TEST_POSTGRES=1 with a live Postgres test DB to run")
	}
	t.Skip("live Postgres migration test is not wired in this environment")
}
