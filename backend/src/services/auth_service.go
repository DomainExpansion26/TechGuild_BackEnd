package services

import (
	"errors"

	"techguild-backend/src/dto"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"
)

type AuthService struct {
	repo repository.UserRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		repo: repository.NewUserRepository(),
	}
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {

	user, err := s.repo.GetUserByEmail(req.Email)
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