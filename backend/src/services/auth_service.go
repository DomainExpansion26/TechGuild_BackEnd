package services

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMy.MrqQKBrEmYq5YoZLxs6VJ1J7bDVU1Aa"

type AuthService struct {
	cfg              *config.Config
	userRepo         repository.UserRepository
	verificationRepo *repository.VerificationRepository
	blacklistRepo    *repository.TokenBlacklistRepository
	tfaRepo          repository.TwoFactorRepository
}

func NewAuthService(redisClient *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{
		cfg:              cfg,
		userRepo:         repository.NewUserRepository(),
		verificationRepo: repository.NewVerificationRepository(redisClient),
		blacklistRepo:    repository.NewTokenBlacklistRepository(redisClient),
		tfaRepo:          repository.NewTwoFactorRepository(),
	}
}

func (s *AuthService) Register(req dto.RegisterRequest) error {
	req.Email = utils.NormalizeEmail(req.Email)
	req.FirstName = utils.NormalizeName(req.FirstName)
	req.LastName = utils.NormalizeName(req.LastName)

	existingUser, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("something went wrong while checking for existing user")
	}

	if existingUser != nil {
		return errors.New("email already exists")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}
	user := &models.User{
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		Status:        models.StatusPendingVerification,
		EmailVerified: false,
	}

	err = s.userRepo.CreateUser(user)

	if err != nil {
		if utils.IsDuplicateKeyError(err) {
			return errors.New("email already exists")
		}
		return errors.New("something went wrong please try again later")
	}

	// verification := &models.VerificationRecord{
	// 	UserID: user.ID,
	// 	Type:   "email",
	// 	Status: "pending",
	// }

	// err = s.userRepo.CreateVerification(verification)
	// if err != nil {
	// 	return err
	// }

	err = s.SendVerificationEmail(user.ID.String(), user.Email)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) SendVerificationEmail(userID string, email string) error {

	token, err := utils.GenerateVerificationToken(userID)
	if err != nil {
		return err
	}

	err = s.verificationRepo.SaveVerificationToken(userID, token)
	if err != nil {
		return err
	}

	go s.sendWithRetry(email, token, userID)

	// Send email asynchronously
	// go func(email, token string) {
	// 	log.Printf("Sending verification email to %s", email)

	// 	if err := utils.SendVerificationEmail(email, token); err != nil {
	// 		log.Printf("Failed to send verification email to %s: %v", email, err)
	// 		return
	// 	}

	// 	log.Printf("Verification email sent successfully to %s", email)
	// }(email, token)

	return nil
}

func (s *AuthService) sendWithRetry(email, token, userID string) {
	const maxAttempts = 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := utils.SendVerificationEmail(s.cfg, email, token)
		if err == nil {
			log.Printf("Verification email sent to %s (attempt %d)", email, attempt)
			return
		}

		log.Printf("Attempt %d failed to send verification email to %s: %v", attempt, email, err)

		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff 2, 4s
		}
	}
	// Saare attempts fail — ab isse "silently lost" nahi hone denge
	log.Printf("CRITICAL: verification email permanently failed for user_id=%s email=%s", userID, email)
	// TODO: yaha ek persistent failure record daalo (DB table ya monitoring alert)
}

func (s *AuthService) VerifyEmail(req dto.VerifyEmailRequest, device, ipAddress, userAgent string) (*dto.VerifyEmailResponse, string, error) {
	_, err := s.verificationRepo.GoConsumeVerificationToken(req.Token)
	if err == nil {
		return &dto.VerifyEmailResponse{
			Message: "This verification link has already been used...",
		}, "", nil
	}
	userID, err := s.verificationRepo.GetVerificationToken(req.Token)
	if err != nil {
		return nil, "", errors.New("invalid or expired verification link")
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, "", errors.New("user not found")
	}

	// ✅ EDGE CASE FIX — Already verified user
	if user.EmailVerified {
		_ = s.verificationRepo.SaveConsumedVerificationToken(req.Token, userID)
		_ = s.verificationRepo.DeleteVerificationToken(req.Token)

		requiresAccountType := user.AccountType == nil || *user.AccountType == ""

		refreshToken, err := utils.GenerateRefreshToken(userID)
		if err != nil {
			return &dto.VerifyEmailResponse{
				Message:             "Email already verified. Please login to continue.",
				RequiresAccountType: requiresAccountType,
			}, "", nil
		}

		session := &models.UserSession{
			UserID:       user.ID,
			RefreshToken: refreshToken,
			Device:       device,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			IsRevoked:    false,
			ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
		}
		if err := s.userRepo.CreateSession(session); err != nil {
			return &dto.VerifyEmailResponse{
				Message:             "Email already verified. Please login to continue.",
				RequiresAccountType: requiresAccountType,
			}, "", nil
		}

		accessToken, err := utils.GenerateAccessToken(userID, session.ID.String())
		if err != nil {
			return &dto.VerifyEmailResponse{
				Message:             "Email already verified. Please login to continue.",
				RequiresAccountType: requiresAccountType,
			}, "", nil
		}

		return &dto.VerifyEmailResponse{
			Message:             "Email already verified.",
			AccessToken:         accessToken,
			ExpiresIn:           int(utils.AccessTokenTTL.Seconds()),
			RequiresAccountType: requiresAccountType,
		}, refreshToken, nil
	}

	// Baaki flow same
	err = s.userRepo.UpdateEmailVerified(userID, true)
	if err != nil {
		return nil, "", err
	}

	if user.Status == models.StatusPendingVerification {
		if err := s.userRepo.UpdateUserStatus(userID, string(models.StatusActive)); err != nil {
			return nil, "", err
		}
	}

	err = s.verificationRepo.SaveConsumedVerificationToken(req.Token, userID)
	if err != nil {
		return nil, "", err
	}
	_ = s.verificationRepo.DeleteVerificationToken(req.Token)

	_ = s.userRepo.AddUserPoints(userID, 10)

	requiresAccountType := user.AccountType == nil || *user.AccountType == ""

	refreshToken, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return &dto.VerifyEmailResponse{
			Message:             "Email verified successfully. Please login to continue.",
			RequiresAccountType: requiresAccountType,
		}, "", nil
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		Device:       device,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return &dto.VerifyEmailResponse{
			Message:             "Email verified successfully. Please login to continue.",
			RequiresAccountType: requiresAccountType,
		}, "", nil
	}

	accessToken, err := utils.GenerateAccessToken(userID, session.ID.String())
	if err != nil {
		return &dto.VerifyEmailResponse{
			Message:             "Email verified successfully. Please login to continue.",
			RequiresAccountType: requiresAccountType,
		}, "", nil
	}

	return &dto.VerifyEmailResponse{
		Message:             "Email verified successfully. Please select your account type to proceed.",
		AccessToken:         accessToken,
		ExpiresIn:           int(utils.AccessTokenTTL.Seconds()),
		RequiresAccountType: requiresAccountType,
	}, refreshToken, nil
}

func (s *AuthService) ResendVerificationEmail(req dto.ResendVerificationRequest) error {	req.Email = utils.NormalizeEmail(req.Email)

	// Rate limit: max 1 resend per 60s per email
	if s.verificationRepo.Redis != nil {
		cooldownKey := "resend_verify_cooldown:" + req.Email
		ctx := context.Background()
		exists, err := s.verificationRepo.Redis.Exists(ctx, cooldownKey).Result()
		if err == nil && exists > 0 {
			log.Printf("ResendVerification: rate-limited: %s", req.Email)
			return nil
		}
		_ = s.verificationRepo.Redis.Set(ctx, cooldownKey, "1", 60*time.Second).Err()
	}

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		log.Printf("ResendVerification: email not found: %s", req.Email)
		return nil
	}

	if user.EmailVerified {
		log.Printf("ResendVerification: already verified: %s", req.Email)
		return nil
	}

	return s.SendVerificationEmail(user.ID.String(), user.Email)
}

// ---------- Login rate limiting (H1) ----------

const (
	loginMaxAttempts   = 5
	loginAttemptWindow = 5 * time.Minute
)

func loginAttemptKey(email, ip string) string {
	return "login_attempts:" + email + ":" + ip
}

// checkLoginRateLimit returns ErrRateLimited when the user+IP pair has too many
// failed attempts. Fails open when Redis is unavailable (login continues).
func (s *AuthService) checkLoginRateLimit(email, ip string) error {
	if s.verificationRepo.Redis == nil {
		return nil
	}
	key := loginAttemptKey(email, ip)
	count, err := s.verificationRepo.Redis.Get(context.Background(), key).Int()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		log.Printf("login rate limit check error: %v", err)
		return nil
	}
	if count >= loginMaxAttempts {
		return ErrRateLimited
	}
	return nil
}

func (s *AuthService) recordFailedLogin(email, ip string) {
	if s.verificationRepo.Redis == nil {
		return
	}
	key := loginAttemptKey(email, ip)
	count, err := s.verificationRepo.Redis.Incr(context.Background(), key).Result()
	if err != nil {
		log.Printf("login rate limit incr error: %v", err)
		return
	}
	if count == 1 {
		_ = s.verificationRepo.Redis.Expire(context.Background(), key, loginAttemptWindow).Err()
	}
}

func (s *AuthService) clearFailedLogins(email, ip string) {
	if s.verificationRepo.Redis == nil {
		return
	}
	_ = s.verificationRepo.Redis.Del(context.Background(), loginAttemptKey(email, ip)).Err()
}

func (s *AuthService) Login(req dto.LoginRequest, device, ipAddress, userAgent string) (*dto.LoginResponse, string, error) {
	req.Email = utils.NormalizeEmail(req.Email)

	if err := s.checkLoginRateLimit(req.Email, ipAddress); err != nil {
		return nil, "", err
	}

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Perform a dummy password check to mitigate timing attacks
			utils.CheckPassword(req.Password, dummyHash)
			s.recordFailedLogin(req.Email, ipAddress)
			return nil, "", errors.New("invalid email or password")
		}
		s.recordFailedLogin(req.Email, ipAddress)
		return nil, "", errors.New("invalid email or password")
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		s.recordFailedLogin(req.Email, ipAddress)
		return nil, "", errors.New("invalid email or password")
	}

	if !user.EmailVerified {
		// M6: uniform message (no account-state enumeration); silently resend
		// verification email (existing 60s cooldown prevents spam)
		_ = s.ResendVerificationEmail(dto.ResendVerificationRequest{Email: req.Email})
		s.recordFailedLogin(req.Email, ipAddress)
		return nil, "", errors.New("invalid email or password")
	}

	if user.Status == models.StatusPendingDeletion || user.Status == models.StatusDeactivated {
		user.Status = models.StatusActive
		user.ScheduledDeletionDate = nil
		if err := s.userRepo.UpdateUser(user); err != nil {
			return nil, "", err
		}
	}

	if user.Status != models.StatusActive {
		s.recordFailedLogin(req.Email, ipAddress)
		return nil, "", errors.New("invalid email or password")
	}

	requiresAccountType := user.AccountType == nil || *user.AccountType == ""

	// ---------- 2FA CHECK (new) ----------
	if user.TwoFactorEnabled {
		record, err := s.tfaRepo.GetByUserID(user.ID)
		if err != nil || record.Status != models.TwoFAStatusEnabled {
			return nil, "", errors.New("two-factor authentication configuration invalid")
		}
		tempToken, err := utils.GenerateTemporary2FAToken(user.ID.String())
		if err != nil {
			return nil, "", err
		}
		return &dto.LoginResponse{
			Message:             "Two-factor authentication required",
			RequiresTwoFactor:   true,
			TemporaryToken:      tempToken,
			RequiresAccountType: requiresAccountType,
		}, "", nil
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, "", err
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		Device:       device,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}

	err = s.userRepo.CreateSession(session)
	if err != nil {
		return nil, "", err
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String(), session.ID.String())
	if err != nil {
		return nil, "", err
	}

	s.clearFailedLogins(req.Email, ipAddress)

	return &dto.LoginResponse{
		Message:             "Login successful",
		AccessToken:         accessToken,
		ExpiresIn:           int(utils.AccessTokenTTL.Seconds()),
		RequiresAccountType: requiresAccountType,
	}, refreshToken, nil
}

func (s *AuthService) Logout(refreshToken string) error {

	if refreshToken == "" {
		return errors.New("refresh token is missing")
	}
	return s.userRepo.RevokeSession(refreshToken)
}

func (s *AuthService) RefreshToken(oldToken string, device string, ipAddress string, userAgent string) (*dto.RefreshResponse, string, error) {

	session, err := s.userRepo.GetSession(oldToken)
	if err != nil {
		return nil, "", errors.New("invalid refresh token")
	}

	if session.IsRevoked {
		_ = s.userRepo.RevokeAllSessions(session.UserID.String())
		return nil, "", errors.New("refresh token revoked")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, "", errors.New("refresh token expired")
	}

	claims, err := utils.ValidateRefreshToken(oldToken)
	if err != nil {
		return nil, "", errors.New("invalid refresh token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID != session.UserID.String() {
		// M8: claims user_id must match the session's owner — otherwise the
		// token belongs to a different account; revoke this session.
		_ = s.userRepo.RevokeSessionByID(session.ID)
		return nil, "", errors.New("invalid refresh token")
	}

	newRefreshToken, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, "", err
	}

	newSession := &models.UserSession{
		UserID:       session.UserID,
		RefreshToken: newRefreshToken,
		Device:       device,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}
	if err := s.userRepo.WithTransaction(func(txRepo repository.UserRepository) error {
		if err := txRepo.RevokeSessionByID(session.ID); err != nil {
			return err
		}
		return txRepo.CreateSession(newSession)
	}); err != nil {
		return nil, "", err
	}

	newAccessToken, err := utils.GenerateAccessToken(userID, newSession.ID.String())
	if err != nil {
		return nil, "", err
	}

	// H4: mark the old session's access tokens as revoked immediately
	if s.verificationRepo.Redis != nil {
		_ = s.verificationRepo.Redis.Set(context.Background(), "session_revoked:"+session.ID.String(), "1", utils.AccessTokenTTL).Err()
	}

	return &dto.RefreshResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   int(utils.AccessTokenTTL.Seconds()),
	}, newRefreshToken, nil
}

func (s *AuthService) ForgotPassword(req dto.ForgotPasswordRequest) error {
	req.Email = utils.NormalizeEmail(req.Email)

	// Rate limit: max 1 forgot-password request per 60s per email
	if s.verificationRepo.Redis == nil {
		return errors.New("service temporarily unavailable")
	}
	cooldownKey := "forgot_pwd_cooldown:" + req.Email
	ctx := context.Background()
	exists, err := s.verificationRepo.Redis.Exists(ctx, cooldownKey).Result()
	if err == nil && exists > 0 {
		log.Printf("ForgotPassword: rate-limited: %s", req.Email)
		return nil
	}
	_ = s.verificationRepo.Redis.Set(ctx, cooldownKey, "1", 60*time.Second).Err()

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		log.Printf("ForgotPassword: email not found: %s", req.Email)
		return nil
	}

	if !user.EmailVerified {
		log.Printf("ForgotPassword: email not verified: %s", req.Email)
		return nil
	}

	token, err := utils.GenerateResetPasswordToken(user.ID.String())
	if err != nil {
		return err
	}

	// ✅ Store token in Redis — single-use enforcement.
	// TTL must match JWT expiry (30 min).
	resetKey := "password_reset:" + token
	if err := s.verificationRepo.Redis.Set(ctx, resetKey, user.ID.String(), 30*time.Minute).Err(); err != nil {
		log.Printf("ForgotPassword: failed to store reset token for user=%s: %v", user.ID.String(), err)
		return errors.New("failed to process request, please try again")
	}

	if err = utils.SendResetPasswordEmail(s.cfg, req.Email, token); err != nil {
		log.Printf("ForgotPassword: failed to send email to %s: %v", req.Email, err)
		_ = s.verificationRepo.Redis.Del(ctx, resetKey).Err()
		return errors.New("failed to process request, please try again")
	}
	return nil
}

func (s *AuthService) ResetPassword(token string, req dto.ResetPasswordRequest) error {
	// 1. JWT validity check (signature + expiry)
	claims, err := utils.ValidateResetPasswordToken(token)
	if err != nil {
		return errors.New("invalid or expired reset link")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return errors.New("invalid reset token payload")
	}

	// 2. Single-use check — token must exist in Redis
	ctx := context.Background()
	resetKey := "password_reset:" + token
	storedUserID, err := s.verificationRepo.Redis.Get(ctx, resetKey).Result()
	if err != nil {
		// Token either never issued or already consumed
		return errors.New("invalid or already used reset link")
	}
	if storedUserID != userID {
		return errors.New("invalid reset link")
	}

	// 3. User must still exist (avoid silent no-op on deleted users)
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.New("invalid reset link")
	}

	// 4. New password must differ from current
	if utils.CheckPassword(req.NewPassword, user.PasswordHash) {
		return errors.New("new password must be different from current password")
	}

	// 5. Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// 6. Consume the token FIRST — single-use enforcement before any state
	// change (if the Del fails, abort; the user can retry with a fresh link)
	if err := s.verificationRepo.Redis.Del(ctx, resetKey).Err(); err != nil {
		return errors.New("failed to process reset, please try again")
	}

	// 7. Update password + revoke all sessions atomically
	if err := s.userRepo.WithTransaction(func(txRepo repository.UserRepository) error {
		if err := txRepo.UpdatePassword(userID, hashedPassword); err != nil {
			return err
		}
		return txRepo.RevokeAllSessions(userID)
	}); err != nil {
		return err
	}

	// 8. H4: mark all access tokens issued before now as revoked
	_ = s.verificationRepo.Redis.Set(ctx, "user_revoked_before:"+userID, time.Now().Unix(), utils.AccessTokenTTL).Err()

	return nil
}

func (s *AuthService) ChangePassword(userID string, sessionID string, req dto.ChangePasswordRequest) error {
	// 1. Confirm password must match new password
	if req.NewPassword != req.ConfirmPassword {
		return errors.New("new password and confirm password do not match")
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.New("old password is incorrect")
	}

	// New password must differ from old
	if req.OldPassword == req.NewPassword {
		return errors.New("new password must be different from old password")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(user.ID.String(), hashedPassword); err != nil {
		return err
	}

	// Revoke all OTHER sessions — keep current session alive
	currentSessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return errors.New("invalid session")
	}

	if err := s.userRepo.RevokeOtherSessions(user.ID.String(), currentSessionUUID); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) BlacklistAccessToken(accessToken string) error {
	claims, err := utils.ParseAccessTokenUnverifiedExpiry(accessToken)
	if err != nil {
		return err
	}

	if claims.ExpiresAt == nil {
		return nil
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return s.blacklistRepo.Blacklist(utils.HashToken(accessToken), ttl)
}

func (s *AuthService) SetAccountTypeAuthenticated(userID string, accountType models.AccountType) (*models.User, error) {
	validTypes := map[models.AccountType]bool{
		models.AccountTypeIndividual:  true,
		models.AccountTypeAgencyAdmin: true,
		models.AccountTypeClientAdmin: true,
	}
	if !validTypes[accountType] {
		return nil, errors.New("invalid account type")
	}

	err := s.userRepo.WithTransaction(func(txRepo repository.UserRepository) error {
		if err := txRepo.UpdateAccountType(userID, accountType); err != nil {
			return err
		}
		// ✅ +20 points for selecting account type
		if err := txRepo.AddUserPoints(userID, 20); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.userRepo.GetUserByID(userID)
}
