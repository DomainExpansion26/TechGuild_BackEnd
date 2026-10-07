// services/two_factor_service.go
package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

type TwoFactorService struct {
	userRepo repository.UserRepository
	tfaRepo  repository.TwoFactorRepository
	totp     *utils.TOTPService
}

func NewTwoFactorService() *TwoFactorService {
	return &TwoFactorService{
		userRepo: repository.NewUserRepository(),
		tfaRepo:  repository.NewTwoFactorRepository(),
		totp:     utils.NewTOTPService("TechGuild", utils.TwoFAEncryptionKey),
	}
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

func (s *TwoFactorService) VerifySetup(userID uuid.UUID, code string) (*dto.VerifySetup2FAResponse, error) {
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
		utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=setup")
		return nil, errors.New("invalid code")
	}

	utils.LogSecurityEvent(utils.EventTwoFAEnabled, userID.String(), "")

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

	plainCodes, hashes, err := s.totp.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}

	var codes []models.UserRecoveryCode
	for _, h := range hashes {
		codes = append(codes, models.UserRecoveryCode{UserID: userID, CodeHash: h})
	}
	if err := s.tfaRepo.CreateRecoveryCodes(codes); err != nil {
		return nil, err
	}

	return &dto.VerifySetup2FAResponse{
		Message:       "2FA enabled",
		RecoveryCodes: plainCodes,
	}, nil
}

// ---------- VerifyLogin ----------

func (s *TwoFactorService) VerifyLogin(temporaryToken, code string, device string, ipAddress string, userAgent string) (*dto.LoginResponse, string, error) {
	userID, err := utils.ValidateTemporary2FAToken(temporaryToken)
	if err != nil {
		return nil, "", errors.New("invalid or expired session")
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
		utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=login")
		return nil, "", errors.New("invalid code")
	}
	utils.LogSecurityEvent(utils.EventTwoFALoginSuccess, userID.String(), "")
	return s.issueFinalTokens(userID, device, ipAddress, userAgent)
}

// ---------- VerifyRecoveryCode ----------

func (s *TwoFactorService) VerifyRecoveryCode(temporaryToken, code string, device string, ipAddress string, userAgent string) (*dto.LoginResponse, string, error) {
	userID, err := utils.ValidateTemporary2FAToken(temporaryToken)
	if err != nil {
		return nil, "", errors.New("invalid or expired session")
	}

	codes, err := s.tfaRepo.GetUnusedRecoveryCodes(userID)
	if err != nil {
		return nil, "", err
	}

	for _, c := range codes {
		if s.totp.VerifyRecoveryCode(code, c.CodeHash) {
			_ = s.tfaRepo.MarkRecoveryCodeUsed(c.ID)
			utils.LogSecurityEvent(utils.EventRecoveryCodeUsed, userID.String(), "")
			return s.issueFinalTokens(userID, device, ipAddress, userAgent)
		}
	}

	utils.LogSecurityEvent(utils.EventTwoFAVerifyFailed, userID.String(), "stage=recovery") // ye line missing thi
	return nil, "", errors.New("invalid or already used recovery code")
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

// ---------- RegenerateRecoveryCodes ----------

func (s *TwoFactorService) RegenerateRecoveryCodes(userID uuid.UUID, password string, code string) ([]string, error) {
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
		secret, err := s.totp.DecryptSecret(record.SecretEncrypted)
		if err != nil {
			return nil, err
		}
		if !s.totp.VerifyCode(secret, code) {
			return nil, errors.New("invalid authenticator code")
		}
	}

	if err := s.tfaRepo.InvalidateRecoveryCodes(userID); err != nil {
		return nil, err
	}

	plainCodes, hashes, err := s.totp.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}

	var codes []models.UserRecoveryCode
	for _, h := range hashes {
		codes = append(codes, models.UserRecoveryCode{UserID: userID, CodeHash: h})
	}
	if err := s.tfaRepo.CreateRecoveryCodes(codes); err != nil {
		return nil, err
	}

	utils.LogSecurityEvent(utils.EventRecoveryCodesRegen, userID.String(), "")
	return plainCodes, nil
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
