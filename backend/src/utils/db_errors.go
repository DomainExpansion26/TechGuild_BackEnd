package utils

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // Unique violation error code
	}
	return false
}

// GetDuplicateKeyConstraint returns the constraint name that caused a
// unique-violation error, or "" if err is not a duplicate-key error.
func GetDuplicateKeyConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}
	return ""
}
