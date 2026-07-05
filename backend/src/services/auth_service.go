package services

import (
	"errors"
	"techguild-backend/src/models"
	"techguild-backend/src/dto"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
	"github.com/redis/go-redis/v9"
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