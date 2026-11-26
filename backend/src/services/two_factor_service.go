// services/two_factor_service.go
package services

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

var ErrRateLimited = errors.New("too many failed attempts, please try again later")

const (
	twoFAMaxAttempts   = 5
	twoFAAttemptWindow = 5 * time.Minute
)

type TwoFactorService struct {
	userRepo repository.UserRepository
	tfaRepo  repository.TwoFactorRepository
	totp     *utils.TOTPService
	redis    *redis.Client
}

func NewTwoFactorService() *TwoFactorService {
	return &TwoFactorService{
		userRepo: repository.NewUserRepository(),
		tfaRepo:  repository.NewTwoFactorRepository(),
		totp:     utils.NewTOTPService("TechGuild", utils.TwoFAEncryptionKey),
		redis:    postgres.RedisDB,
	}
}

func twoFAAttemptKey(scope string, userID uuid.UUID) string {
	return "2fa_attempts:" + scope + ":" + userID.String()
}

func (s *TwoFactorService) checkRateLimit(ctx context.Context, scope string, userID uuid.UUID) error {
	if s.redis == nil {
		return nil
	}
	count, err := s.redis.Get(ctx, twoFAAttemptKey(scope, userID)).Int()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("2fa rate limit read error (failing open): %v", err)
		}
		return nil
	}
	if count >= twoFAMaxAttempts {
		return ErrRateLimited
	}
	return nil
}

func (s *TwoFactorService) recordFailedAttempt(ctx context.Context, scope string, userID uuid.UUID) {
	if s.redis == nil {
		return
	}
	key := twoFAAttemptKey(scope, userID)
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("2fa rate limit incr error: %v", err)
		return
	}
	if count == 1 {
		_ = s.redis.Expire(ctx, key, twoFAAttemptWindow).Err()
	}
}

func (s *TwoFactorService) clearFailedAttempts(ctx context.Context, scope string, userID uuid.UUID) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, twoFAAttemptKey(scope, userID)).Err()
}

// ---------- Setup ----------

func (s *TwoFactorService) Setup(userID uuid.UUID) (*dto.Setup2FAResponse, error) {
	user, err := s.userRepo.GetUserByID(userID.String())
	if err != nil {
		return nil, errors.New("user not found")
	}

	// GUARD: if 2fa is already enabled, return error
	existing, err := s.tfaRepo.GetByUserID(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if existing != nil && existing.Status == models.TwoFAStatusEnabled {
		return nil, errors.New("2FA is already enabled; disable it first before setting up again")
	}

	secret, uri, err := s.totp.GenerateSecret(user.Email)
	if err != nil {
		return nil, err
	}

	encrypted, err := s.totp.EncryptSecret(secret)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(10 * time.Minute)

	record := &models.UserTwoFactorAuthentication{
		UserID:           userID,
		SecretEncrypted:  encrypted,
		Status:           models.TwoFAStatusPending,
		PendingExpiresAt: &expiresAt,
	}
	if existing != nil {
		record.ID = existing.ID
	}

	if err := s.tfaRepo.Upsert(record); err != nil {
		return nil, err
	}

	utils.LogSecurityEvent(utils.EventTwoFASetupInitiated, userID.String(), "")

	qrImage, err := s.totp.GenerateQRCodeBase64(uri)
	if err != nil {
		return nil, err
	}
	return &dto.Setup2FAResponse{
		ProvisioningURI: uri,
		QRCodeImage:     qrImage,
		ExpiresAt:       expiresAt.Format(time.RFC3339),
	}, nil
}

// ---------- VerifySetup ----------

func (s *TwoFactorService) VerifySetup(ctx context.Context, userID uuid.UUID, code string) (*dto.VerifySetup2FAResponse, error) {
	if err := s.checkRateLimit(ctx, "setup", userID); err != nil {
		return nil, err
	}

	record, err := s.tfaRepo.GetByUserID(userID)
	if err != nil || record.Status != models.TwoFAStatusPending {
		return nil, errors.New("no pending 2fa setup")
	}

	if record.PendingExpiresAt != nil && time.Now().After(*record.PendingExpiresAt) {
		_ = s.tfaRepo.UpdateStatus(userID, map[string]interface{}{"status": models.TwoFAStatusDisabled})
		return nil, errors.New("setup expired, restart 2fa setup")
	}

	secret, err := s.totp.DecryptSecret(record.SecretEncrypted)
	if err != nil {
		return nil, err
	}

	if !s.totp.VerifyCode(secret, code) {
		s.recordFailedAttempt(ctx, "setup", userID)
		utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=setup")
		return nil, errors.New("invalid code")
	}
	s.clearFailedAttempts(ctx, "setup", userID)

	plainCodes, hashes, err := s.totp.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	codes := make([]models.UserRecoveryCode, 0, len(hashes))
	for _, h := range hashes {
		codes = append(codes, models.UserRecoveryCode{UserID: userID, CodeHash: h})
	}
	if err := s.tfaRepo.ReplaceRecoveryCodes(userID, codes); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.tfaRepo.UpdateStatus(userID, map[string]interface{}{
		"status":             models.TwoFAStatusEnabled,
		"verified_at":        now,
		"pending_expires_at": nil,
	}); err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateTwoFactorEnabled(userID.String(), true); err != nil {
		return nil, err
	}

	utils.LogSecurityEvent(utils.EventTwoFAEnabled, userID.String(), "")

	return &dto.VerifySetup2FAResponse{
		Message:       "2FA enabled",
		RecoveryCodes: plainCodes,
	}, nil
}

func (s *TwoFactorService) VerifyLogin(ctx context.Context, temporaryToken, code string, device string, ipAddress string, userAgent string) (*dto.LoginResponse, string, error) {
	userID, jti, expiresAt, err := utils.ValidateTemporary2FATokenWithJTI(temporaryToken)
	if err != nil || s.isTempTokenUsed(ctx, jti) {
		return nil, "", errors.New("invalid or expired session")
	}

	if err := s.checkRateLimit(ctx, "login", userID); err != nil {
		return nil, "", err
	}

	record, err := s.tfaRepo.GetByUserID(userID)
	if err != nil || record.Status != models.TwoFAStatusEnabled {
		return nil, "", errors.New("2fa is not enabled")
	}

	secret, err := s.totp.DecryptSecret(record.SecretEncrypted)
	if err != nil {
		return nil, "", err
	}

	if !s.totp.VerifyCode(secret, code) {
		s.recordFailedAttempt(ctx, "login", userID)
		utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=login")
		return nil, "", errors.New("invalid code")
	}
	s.clearFailedAttempts(ctx, "login", userID)
	utils.LogSecurityEvent(utils.EventTwoFALoginSuccess, userID.String(), "")
	if !s.consumeTempToken(ctx, jti, expiresAt) {
		return nil, "", errors.New("invalid or expired session")
	}
	return s.issueFinalTokens(userID, device, ipAddress, userAgent)
}

func (s *TwoFactorService) VerifyRecoveryCode(ctx context.Context, temporaryToken, code string, device string, ipAddress string, userAgent string) (*dto.LoginResponse, string, error) {
	userID, err := utils.ValidateTemporary2FAToken(temporaryToken)
	if err != nil {
		return nil, "", errors.New("invalid or expired session")
	}

	if err := s.checkRateLimit(ctx, "recovery", userID); err != nil {
		return nil, "", err
	}

	codes, err := s.tfaRepo.GetUnusedRecoveryCodes(userID)
	if err != nil {
		return nil, "", err
	}

	for _, c := range codes {
		if s.totp.VerifyRecoveryCode(code, c.CodeHash) {
			if err := s.tfaRepo.MarkRecoveryCodeUsed(c.ID); err != nil {
				s.recordFailedAttempt(ctx, "recovery", userID)
				return nil, "", errors.New("invalid or already used recovery code")
			}
			s.clearFailedAttempts(ctx, "recovery", userID)
			utils.LogSecurityEvent(utils.EventRecoveryCodeUsed, userID.String(), "")
			return s.issueFinalTokens(userID, device, ipAddress, userAgent)
		}
	}

	s.recordFailedAttempt(ctx, "recovery", userID)
	utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=recovery")
	return nil, "", errors.New("invalid or already used recovery code")
}

func (s *TwoFactorService) RegenerateRecoveryCodes(ctx context.Context, userID uuid.UUID, password string, code string) ([]string, error) {
	user, err := s.userRepo.GetUserByID(userID.String())
	if err != nil {
		return nil, errors.New("user not found")
	}

	record, err := s.tfaRepo.GetByUserID(userID)
	if err != nil || record.Status != models.TwoFAStatusEnabled {
		return nil, errors.New("2fa is not enabled")
	}

	if user.PasswordHash != "" {
		if !utils.CheckPassword(password, user.PasswordHash) {
			return nil, errors.New("incorrect password")
		}
	} else {
		if err := s.checkRateLimit(ctx, "regen", userID); err != nil {
			return nil, err
		}
		secret, err := s.totp.DecryptSecret(record.SecretEncrypted)
		if err != nil {
			return nil, err
		}
		if !s.totp.VerifyCode(secret, code) {
			s.recordFailedAttempt(ctx, "regen", userID)
			return nil, errors.New("invalid authenticator code")
		}
		s.clearFailedAttempts(ctx, "regen", userID)
	}

	plainCodes, hashes, err := s.totp.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}

	codes := make([]models.UserRecoveryCode, 0, len(hashes))
	for _, h := range hashes {
		codes = append(codes, models.UserRecoveryCode{UserID: userID, CodeHash: h})
	}
	if err := s.tfaRepo.ReplaceRecoveryCodes(userID, codes); err != nil {
		return nil, err
	}

	utils.LogSecurityEvent(utils.EventRecoveryCodesRegen, userID.String(), "")
	return plainCodes, nil
}

// ---------- Disable ----------

func (s *TwoFactorService) Disable(userID uuid.UUID, password, code string) error {
	user, err := s.userRepo.GetUserByID(userID.String())
	if err != nil {
		return errors.New("user not found")
	}

	// OAuth-only users have no password — skip password check, rely on TOTP code alone for verification
	if user.PasswordHash != "" {
		if !utils.CheckPassword(password, user.PasswordHash) {
			return errors.New("incorrect password")
		}
	}

	record, err := s.tfaRepo.GetByUserID(userID)
	if err != nil || record.Status != models.TwoFAStatusEnabled {
		return errors.New("2fa is not enabled")
	}

	secret, err := s.totp.DecryptSecret(record.SecretEncrypted)
	if err != nil {
		return err
	}
	if !s.totp.VerifyCode(secret, code) {
		return errors.New("invalid authenticator code")
	}

	if err := s.tfaRepo.UpdateStatus(userID, map[string]interface{}{
		"status":      models.TwoFAStatusDisabled,
		"verified_at": nil,
	}); err != nil {
		return err
	}

	if err := s.userRepo.UpdateTwoFactorEnabled(userID.String(), false); err != nil {
		return err
	}

	utils.LogSecurityEvent(utils.EventTwoFADisabled, userID.String(), "")
	return s.tfaRepo.InvalidateRecoveryCodes(userID)
}

// ---------- helper ----------

func (s *TwoFactorService) issueFinalTokens(userID uuid.UUID, device, ipAddress, userAgent string) (*dto.LoginResponse, string, error) {
	user, err := s.userRepo.GetUserByID(userID.String())
	if err != nil {
		return nil, "", errors.New("user not found")
	}

	refreshToken, err := utils.GenerateRefreshToken(userID.String())
	if err != nil {
		return nil, "", err
	}

	session := &models.UserSession{
		UserID:       userID,
		RefreshToken: refreshToken,
		Device:       device,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, "", err
	}

	accessToken, err := utils.GenerateAccessToken(userID.String(), session.ID.String())
	if err != nil {
		return nil, "", err
	}

	requiresAccountType := user.AccountType == nil || *user.AccountType == ""

	return &dto.LoginResponse{
		Message:             "Login successful",
		AccessToken:         accessToken,
		ExpiresIn:           int(utils.AccessTokenTTL.Seconds()),
		RequiresAccountType: requiresAccountType, // ✅ ADD
	}, refreshToken, nil
}

func (s *TwoFactorService) isTempTokenUsed(ctx context.Context, jti string) bool {
	if s.redis == nil || jti == "" {
		return false
	}
	exists, err := s.redis.Exists(ctx, "2fa_used_jti:"+jti).Result()
	if err != nil {
		log.Printf("2fa jti check error (failing open): %v", err)
		return false
	}
	return exists > 0
}

func (s *TwoFactorService) consumeTempToken(ctx context.Context, jti string, expiresAt time.Time) bool {
	if s.redis == nil || jti == "" {
		return true
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = utils.Temporary2FATokenTTL
	}
	ok, err := s.redis.SetNX(ctx, "2fa_used_jti:"+jti, "1", ttl).Result()
	if err != nil {
		log.Printf("2fa jti consume error (failing open): %v", err)
		return true
	}
	return ok
}
