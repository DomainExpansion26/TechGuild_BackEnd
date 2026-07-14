package services

import (
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

type AuthService struct {
	userRepo         repository.UserRepository
	verificationRepo *repository.VerificationRepository
}

func NewAuthService(redisClient *redis.Client) *AuthService {
	return &AuthService{
		userRepo:         repository.NewUserRepository(),
		verificationRepo: repository.NewVerificationRepository(redisClient),
	}
}

func (s *AuthService) Register(req dto.RegisterRequest) error {

	existingUser, _ := s.userRepo.GetUserByEmail(req.Email)
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
		return err
	}

	verification := &models.VerificationRecord{
		UserID: user.ID,
		Type:   "email",
		Status: "pending",
	}

	err = s.userRepo.CreateVerification(verification)
	if err != nil {
		return err
	}

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

	err = utils.SendVerificationEmail(email, token)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) VerifyEmail(req dto.VerifyEmailRequest) (*dto.LoginResponse, error) {

	userID, err := s.verificationRepo.GetVerificationToken(req.Token)
	if err != nil {
		return nil, errors.New("invalid or expired verification link")
	}

	err = s.userRepo.UpdateEmailVerified(userID, true)
	if err != nil {
		return nil, err
	}

	_ = s.verificationRepo.DeleteVerificationToken(req.Token)

	// Add +10 points for verifying the email
	_ = s.userRepo.AddUserPoints(userID, 10)

	return &dto.LoginResponse{
		Message: "Email verified successfully. Please select your account type to proceed.",
	}, nil
}

func (s *AuthService) ResendVerificationEmail(req dto.ResendVerificationRequest) error {

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return errors.New("user not found")
	}

	if user.EmailVerified {
		return errors.New("email already verified")
	}

	return s.SendVerificationEmail(user.ID.String(), user.Email)
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	if !user.EmailVerified {
		return nil, errors.New("please verify your email first")
	}

	if user.AccountType == nil || *user.AccountType == "" {
		return nil, errors.New("please select an account type first")
	}

	if user.Status == models.StatusPendingDeletion {
		user.Status = models.StatusActive
		user.ScheduledDeletionDate = nil
		if err := s.userRepo.UpdateUser(user); err != nil {
			return nil, err
		}
	}

	if user.Status != models.StatusActive {
		return nil, errors.New("account is not active")
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
	}

	err = s.userRepo.CreateSession(session)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Message:      "Login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
func (s *AuthService) Logout(req dto.LogoutRequest) error {

	session, err := s.userRepo.GetSession(req.RefreshToken)
	if err != nil {
		return errors.New("invalid session")
	}

	if session.IsRevoked {
		return errors.New("already logged out")
	}

	err = s.userRepo.RevokeSession(req.RefreshToken)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) RefreshToken(req dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {

	session, err := s.userRepo.GetSession(req.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if session.IsRevoked {
		return nil, errors.New("refresh token revoked")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("refresh token expired")
	}

	claims, err := utils.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid token")
	}

	newAccessToken, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	err = s.userRepo.UpdateRefreshToken(req.RefreshToken, newRefreshToken)
	if err != nil {
		return nil, err
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) ForgotPassword(req dto.ForgotPasswordRequest) error {

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

	userID := claims["user_id"].(string)

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

func (s *AuthService) GoogleLogin(req dto.GoogleLoginRequest) (*dto.GoogleLoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)

	if err != nil {

		// Parse FullName into FirstName and LastName
		firstName := req.FullName
		lastName := ""
		if parts := strings.Split(req.FullName, " "); len(parts) > 1 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}

		user = &models.User{
			FirstName:     firstName,
			LastName:      lastName,
			Email:         req.Email,
			EmailVerified: true,
			OAuthProvider: stringPtr("google"),
			OAuthID:       stringPtr(req.GoogleID),
			Status:        models.StatusActive,
		}

		if err := s.userRepo.CreateUser(user); err != nil {
			return nil, err
		}

	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
	}

	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, err
	}

	return &dto.GoogleLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "Google login successful",
	}, nil
}

func (s *AuthService) GitHubLogin(req dto.GitHubLoginRequest) (*dto.GitHubLoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)

	if err != nil {

		// Parse FullName into FirstName and LastName
		firstName := req.FullName
		lastName := ""
		if parts := strings.Split(req.FullName, " "); len(parts) > 1 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}

		user = &models.User{
			FirstName:     firstName,
			LastName:      lastName,
			Email:         req.Email,
			EmailVerified: true,
			OAuthProvider: stringPtr("github"),
			OAuthID:       stringPtr(req.GitHubID),
			Status:        models.StatusActive,
		}

		if err := s.userRepo.CreateUser(user); err != nil {
			return nil, err
		}

	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
	}

	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, err
	}

	return &dto.GitHubLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "GitHub login successful",
	}, nil
}

func (s *AuthService) DeleteAccount(userID string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	return s.userRepo.DeleteUser(user.ID.String())
}
