package services

import (
	"errors"
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

	existingPhone, _ := s.userRepo.GetUserByPhone(req.Phone)
	if existingPhone != nil {
		return errors.New("phone already exists")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}

	user := &models.User{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		AccountType:  models.AccountType(req.AccountType),
		Status:       models.StatusPendingVerification,
	}

	err = s.userRepo.CreateUser(user)
	if err != nil {
		return err
	}

	profile := &models.UserProfile{
		UserID: user.ID,
	}

	err = s.userRepo.CreateProfile(profile)
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

	otp, err := utils.GenerateOTP()
	if err != nil {
		return err
	}

	err = s.verificationRepo.SaveOTP(req.Email, otp)
	if err != nil {
		return err
	}

	err = utils.SendOTPEmail(req.Email, otp)
	if err != nil {
		return err
	}

	return nil
}
func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "active" {
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
func (s *AuthService) SendVerificationOTP(email string) error {

	otp, err := utils.GenerateOTP()
	if err != nil {
		return err
	}

	err = s.verificationRepo.SaveOTP(email, otp)
	if err != nil {
		return err
	}

	err = utils.SendOTPEmail(email, otp)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) VerifyEmail(req dto.VerifyEmailRequest) error {

	otp, err := s.verificationRepo.GetOTP(req.Email)
	if err != nil {
		return errors.New("otp expired or invalid")
	}

	if otp != req.OTP {
		return errors.New("invalid otp")
	}

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdateUserStatus(user.ID.String(), "active")
	if err != nil {
		return err
	}

	_ = s.verificationRepo.DeleteOTP(req.Email)

	return nil
}

func (s *AuthService) ResendOTP(req dto.ResendOTPRequest) error {
	return s.SendVerificationOTP(req.Email)
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

	if user.Status != "active" {
		return errors.New("account is not active")
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		return err
	}

	err = s.verificationRepo.SaveOTP("reset:"+req.Email, otp)
	if err != nil {
		return err
	}

	err = utils.SendOTPEmail(req.Email, otp)
	if err != nil {
		return err
	}

	return nil
}
func (s *AuthService) VerifyResetOTP(req dto.VerifyResetOTPRequest) error {

	otp, err := s.verificationRepo.GetOTP("reset:" + req.Email)
	if err != nil {
		return errors.New("otp expired or invalid")
	}

	if otp != req.OTP {
		return errors.New("invalid otp")
	}

	err = s.verificationRepo.SaveOTP("verified:"+req.Email, "true")
	if err != nil {
		return err
	}

	_ = s.verificationRepo.DeleteOTP("reset:" + req.Email)

	return nil
}
func (s *AuthService) ResetPassword(req dto.ResetPasswordRequest) error {

	verified, err := s.verificationRepo.GetOTP("verified:" + req.Email)
	if err != nil || verified != "true" {
		return errors.New("otp verification required")
	}

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return errors.New("user not found")
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

	_ = s.verificationRepo.DeleteOTP("verified:" + req.Email)

	return nil
}