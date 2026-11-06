package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef" // 32 bytes for AES-256

func TestEncryptDecryptSecret_RoundTrip(t *testing.T) {
	t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", testEncryptionKey)

	const plaintext = "1234567890123456" // bank account number

	ciphertext, err := EncryptSecret(plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret returned an error: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("EncryptSecret returned an empty ciphertext")
	}
	if strings.Contains(ciphertext, plaintext) {
		t.Fatal("ciphertext contains the plaintext")
	}

	decrypted, err := DecryptSecret(ciphertext)
	if err != nil {
		t.Fatalf("DecryptSecret returned an error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptSecret_UsesRandomNonce(t *testing.T) {
	t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", testEncryptionKey)

	first, err := EncryptSecret("same-secret")
	if err != nil {
		t.Fatalf("first EncryptSecret failed: %v", err)
	}
	second, err := EncryptSecret("same-secret")
	if err != nil {
		t.Fatalf("second EncryptSecret failed: %v", err)
	}

	if first == second {
		t.Fatal("two encryptions of the same plaintext produced identical ciphertext (nonce is not random)")
	}
}

func TestDecryptSecret_RejectsTamperedCiphertext(t *testing.T) {
	t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", testEncryptionKey)

	ciphertext, err := EncryptSecret("1234567890123456")
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("could not decode ciphertext: %v", err)
	}

	// Flip a bit in the authentication tag (last byte) — GCM must reject it.
	raw[len(raw)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := DecryptSecret(tampered); err == nil {
		t.Fatal("DecryptSecret accepted a tampered ciphertext, expected an authentication error")
	}
}

func TestDecryptSecret_RejectsTamperedNonce(t *testing.T) {
	t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", testEncryptionKey)

	ciphertext, err := EncryptSecret("1234567890123456")
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("could not decode ciphertext: %v", err)
	}

	// Flip a bit in the nonce (first byte).
	raw[0] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := DecryptSecret(tampered); err == nil {
		t.Fatal("DecryptSecret accepted a ciphertext with a tampered nonce")
	}
}

func TestDecryptSecret_RejectsMalformedInput(t *testing.T) {
	t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", testEncryptionKey)

	cases := map[string]string{
		"not base64": "!!!not-valid-base64!!!",
		"too short":  base64.StdEncoding.EncodeToString([]byte("short")),
		"empty":      "",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecryptSecret(input); err == nil {
				t.Fatalf("DecryptSecret(%q) succeeded, expected an error", input)
			}
		})
	}
}

func TestEncryptSecret_Requires32ByteKey(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", "")
		if _, err := EncryptSecret("secret"); err == nil {
			t.Fatal("EncryptSecret succeeded without BANK_DETAILS_ENCRYPTION_KEY")
		}
	})

	t.Run("short key", func(t *testing.T) {
		t.Setenv("BANK_DETAILS_ENCRYPTION_KEY", "too-short")
		if _, err := EncryptSecret("secret"); err == nil {
			t.Fatal("EncryptSecret succeeded with a short key")
		}
	})
}
