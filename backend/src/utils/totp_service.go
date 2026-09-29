package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

func getEnvOrPanic(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("environment variable %s is required but not set", key))
	}
	return val
}

var TwoFAEncryptionKey = mustDecodeBase64Key(getEnvOrPanic("TWO_FACTOR_ENCRYPTION_KEY")) // must be 32 bytes (AES-256)

type TOTPService struct {
	issuer        string
	encryptionKey []byte
}

func NewTOTPService(issuer string, encryptionKey []byte) *TOTPService {
	return &TOTPService{issuer: issuer, encryptionKey: encryptionKey}
}

func mustDecodeBase64Key(encoded string) []byte {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic("invalid TWO_FACTOR_ENCRYPTION_KEY: must be valid base64")
	}
	if len(key) != 32 {
		panic(fmt.Sprintf("invalid TWO_FACTOR_ENCRYPTION_KEY: must decode to 32 bytes (AES-256), got %d", len(key)))
	}
	return key
}

func (s *TOTPService) GenerateSecret(userEmail string) (secret string, provisioningURI string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: userEmail,
		Period:      30,
		Digits:      otp.DigitsSix,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

func (s *TOTPService) VerifyCode(secret, code string) bool {
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1, // ±30s clock-drift tolerance
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return valid
}

func (s *TOTPService) EncryptSecret(plain string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *TOTPService) DecryptSecret(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *TOTPService) GenerateRecoveryCodes(n int) (plainCodes []string, hashes []string, err error) {
	for i := 0; i < n; i++ {
		code, err := randomRecoveryCode()
		if err != nil {
			return nil, nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		plainCodes = append(plainCodes, code)
		hashes = append(hashes, string(hash))
	}
	return plainCodes, hashes, nil
}

func (s *TOTPService) VerifyRecoveryCode(code, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil
}

func randomRecoveryCode() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	code := base64.RawURLEncoding.EncodeToString(sum[:])[:8]
	return fmt.Sprintf("%s-%s", code[:4], code[4:]), nil
}

func (s *TOTPService) GenerateQRCodeBase64(provisioningURI string) (string, error) {
	png, err := qrcode.Encode(provisioningURI, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
