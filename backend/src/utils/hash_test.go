package utils

import (
	"encoding/hex"
	"testing"
)

func TestHashGovtID_DeterministicAndDistinct(t *testing.T) {
	t.Setenv("HASH_SALT", "unit-test-salt")

	const input = "ABCDE1234F"

	first := HashGovtID(input)
	second := HashGovtID(input)

	if first != second {
		t.Fatalf("HashGovtID is not deterministic: %q != %q", first, second)
	}
	if first == "" {
		t.Fatal("HashGovtID returned an empty hash")
	}
	if len(first) != 64 {
		t.Fatalf("HashGovtID should return a 64-char SHA-256 hex string, got %d chars", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("HashGovtID returned non-hex output: %v", err)
	}

	other := HashGovtID("ABCDE1234G")
	if first == other {
		t.Fatalf("different government IDs produced the same hash: %q", first)
	}
}

func TestHashGovtID_DependsOnSalt(t *testing.T) {
	const input = "ABCDE1234F"

	t.Setenv("HASH_SALT", "salt-one")
	withSaltOne := HashGovtID(input)

	t.Setenv("HASH_SALT", "salt-two")
	withSaltTwo := HashGovtID(input)

	if withSaltOne == withSaltTwo {
		t.Fatal("HashGovtID output did not change when HASH_SALT changed")
	}
}

func TestHashBusinessPAN_DeterministicAndDistinct(t *testing.T) {
	t.Setenv("HASH_SALT", "unit-test-salt")

	const input = "AAAPZ1234C"

	first := HashBusinessPAN(input)
	second := HashBusinessPAN(input)

	if first != second {
		t.Fatalf("HashBusinessPAN is not deterministic: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("HashBusinessPAN should return a 64-char SHA-256 hex string, got %d chars", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("HashBusinessPAN returned non-hex output: %v", err)
	}

	other := HashBusinessPAN("AAAPZ1234D")
	if first == other {
		t.Fatalf("different PANs produced the same hash: %q", first)
	}
}
