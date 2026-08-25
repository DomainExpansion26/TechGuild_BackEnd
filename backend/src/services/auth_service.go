package services

import (
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMy.MrqQKBrEmYq5YoZLxs6VJ1J7bDVU1Aa"

type AuthService struct {
	userRepo         repository.UserRepository
	verificationRepo *repository.VerificationRepository
	blacklistRepo    *repository.TokenBlacklistRepository
}

func NewAuthService(redisClient *redis.Client) *AuthService {
	return &AuthService{
		userRepo:         repository.NewUserRepository(),
		verificationRepo: repository.NewVerificationRepository(redisClient),
		blacklistRepo:    repository.NewTokenBlacklistRepository(redisClient),
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

	go sendWithRetry(email, token, userID)

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

func sendWithRetry(email, token, userID string) {
	const maxAttempts = 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := utils.SendVerificationEmail(email, token)
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

func (s *AuthService) VerifyEmail(req dto.VerifyEmailRequest) (*dto.VerifyEmailResponse, error) {

	_, err := s.verificationRepo.GoConsumeVerificationToken(req.Token)
	if err == nil {
		// Token was already used.
		// This can happen because of an email security scanner
		// or because the user clicked the same link again.
		return &dto.VerifyEmailResponse{
			Message: "This verification link has already been used. If you haven't verified your email yet, please request a new verification email.",
		}, nil
	}
	userID, err := s.verificationRepo.GetVerificationToken(req.Token)
	if err != nil {
		return nil, errors.New("invalid or expired verification link")
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// check email already verified
	if user.EmailVerified {
		_ = s.verificationRepo.SaveConsumedVerificationToken(req.Token, userID)

		return &dto.VerifyEmailResponse{
			Message: "Email Already Verified.Please Select Your Account type to proceed",
		}, nil
	}

	err = s.userRepo.UpdateEmailVerified(userID, true)
	if err != nil {
		return nil, err
	}

	if user.Status == models.StatusPendingVerification {
		if err := s.userRepo.UpdateUserStatus(userID, string(models.StatusActive)); err != nil {
			return nil, err
		}
	}

	// 7. Save consumed marker.
	// DON'T immediately delete the original token.
	err = s.verificationRepo.SaveConsumedVerificationToken(req.Token, userID)
	if err != nil {
		return nil, err
	}

	// _ = s.verificationRepo.DeleteVerificationToken(req.Token)

	// Add +10 points for verifying the email
	_ = s.userRepo.AddUserPoints(userID, 10)

	return &dto.VerifyEmailResponse{
		Message: "Email verified successfully. Please select your account type to proceed.",
	}, nil
}

func (s *AuthService) ResendVerificationEmail(req dto.ResendVerificationRequest) error {
	req.Email = utils.NormalizeEmail(req.Email)
	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return errors.New("user not found")
	}

	if user.EmailVerified {
		return errors.New("email already verified")
	}

	return s.SendVerificationEmail(user.ID.String(), user.Email)
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, string, error) {
	req.Email = utils.NormalizeEmail(req.Email)
	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Perform a dummy password check to mitigate timing attacks
			utils.CheckPassword(req.Password, dummyHash)
			return nil, "", errors.New("invalid email or password")
		}
		return nil, "", errors.New("invalid email or password")
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, "", errors.New("invalid email or password")
	}

	if !user.EmailVerified {
		return nil, "", errors.New("please verify your email first")
	}

	if user.AccountType == nil || *user.AccountType == "" {
		return nil, "", errors.New("please select an account type first")
	}

	if user.Status == models.StatusPendingDeletion {
		user.Status = models.StatusActive
		user.ScheduledDeletionDate = nil
		if err := s.userRepo.UpdateUser(user); err != nil {
			return nil, "", err
		}
	}

	if user.Status != models.StatusActive {
		return nil, "", errors.New("account is not active")
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, "", err
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}

	err = s.userRepo.CreateSession(session)
	if err != nil {
		return nil, "", err
	}

	return &dto.LoginResponse{
		Message:     "Login successful",
		AccessToken: accessToken,
		ExpiresIn:   int(utils.AccessTokenTTL.Seconds()),
	}, refreshToken, nil
}
func (s *AuthService) Logout(refreshToken string) error {

	if refreshToken == "" {
		return errors.New("refresh token is missing")
	}
	return s.userRepo.RevokeSession(refreshToken)
}

func (s *AuthService) RefreshToken(oldToken string) (*dto.RefreshResponse, string, error) {

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
	if !ok {
		return nil, "", errors.New("invalid token")
	}

	newAccessToken, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return nil, "", err
	}

	newRefreshToken, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, "", err
	}

	err = s.userRepo.UpdateRefreshToken(oldToken, newRefreshToken)
	if err != nil {
		return nil, "", err
	}

	// UpdateRefreshToken (overwrite) ki jagah revoke + naya session
	if err := s.userRepo.RevokeSessionByID(session.ID); err != nil {
		return nil, "", err
	}

	newSession := &models.UserSession{
		UserID:       session.UserID,
		RefreshToken: newRefreshToken,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(utils.RefreshTokenTTL),
	}
	if err := s.userRepo.CreateSession(newSession); err != nil {
		return nil, "", err
	}

	return &dto.RefreshResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   int(utils.AccessTokenTTL.Seconds()),
	}, newRefreshToken, nil
}

func (s *AuthService) ForgotPassword(req dto.ForgotPasswordRequest) error {

	req.Email = utils.NormalizeEmail(req.Email)
	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return errors.New("user not found")
	}

	if !user.EmailVerified {
		return errors.New("email not verified")
	}

	token, err := utils.GenerateResetPasswordToken(user.ID.String())
	if err != nil {
		return err
	}

	err = utils.SendResetPasswordEmail(req.Email, token)
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) ResetPassword(req dto.ResetPasswordRequest) error {

	claims, err := utils.ValidateResetPasswordToken(req.Token)
	if err != nil {
		return errors.New("invalid or expired reset link")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return errors.New("Invalid reset token payload")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(userID, hashedPassword)
	if err != nil {
		return err
	}

	err = s.userRepo.RevokeAllSessions(userID)
	if err != nil {
		return err
	}
	return nil
}
func (s *AuthService) ChangePassword(userID string, req dto.ChangePasswordRequest) error {

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.New("old password is incorrect")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(user.ID.String(), hashedPassword)
	if err != nil {
		return err
	}

	err = s.userRepo.RevokeAllSessions(user.ID.String())
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) DeleteAccount(userID string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	return s.userRepo.DeleteUser(user.ID.String())
}

func (s *AuthService) BlacklistAccessToken(accessToken string) error {
	claims, err := utils.ParseAccessTokenUnverifiedExpiry(accessToken)
	if err != nil {
		return err
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	return s.blacklistRepo.Blacklist(utils.HashToken(accessToken), ttl)
}
