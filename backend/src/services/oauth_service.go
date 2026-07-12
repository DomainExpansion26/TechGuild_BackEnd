package services

import (
	"errors"
	"strings"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/redis/go-redis/v9"
)

type OAuthService struct {
	userRepo repository.UserRepository
}

func NewOAuthService(redisClient *redis.Client) *OAuthService {
	return &OAuthService{
		userRepo: repository.NewUserRepository(),
	}
}

func (s *OAuthService) GoogleLogin(req dto.GoogleLoginRequest) (*dto.GoogleLoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)

	if err != nil {

		user = &models.User{
			Email:          req.Email,
			FullName:       req.FullName,
			PasswordHash:   "",
			Status:         models.StatusActive,
			EmailVerified:  true,
			OAuthProvider:  "google",
			OAuthID:        req.GoogleID,
		}

		err = s.userRepo.CreateUser(user)
		if err != nil {
			return nil, err
		}

		firstName := user.FullName
		if parts := strings.Split(user.FullName, " "); len(parts) > 0 {
			firstName = parts[0]
		}
		profile := &models.UserProfile{
			UserID:    user.ID,
			FirstName: firstName,
		}

		err = s.userRepo.CreateProfile(profile)
		if err != nil {
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
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
		IsRevoked:    false,
	}

	err = s.userRepo.CreateSession(session)
	if err != nil {
		return nil, errors.New("failed to create session")
	}

	return &dto.GoogleLoginResponse{
		Message:      "Google login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
func (s *OAuthService) GitHubLogin(req dto.GitHubLoginRequest) (*dto.GitHubLoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)

	if err != nil {

		user = &models.User{
			Email:           req.Email,
			FullName:        req.FullName,
			PasswordHash:   "",
			Status:          models.StatusActive,
			EmailVerified:   true,
			OAuthProvider:   "github",
			OAuthID:         req.GitHubID,
		}

		err = s.userRepo.CreateUser(user)
		if err != nil {
			return nil, err
		}

		firstName := user.FullName
		if parts := strings.Split(user.FullName, " "); len(parts) > 0 {
			firstName = parts[0]
		}
		profile := &models.UserProfile{
			UserID:    user.ID,
			FirstName: firstName,
		}

		err = s.userRepo.CreateProfile(profile)
		if err != nil {
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
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
		IsRevoked:    false,
	}

	err = s.userRepo.CreateSession(session)
	if err != nil {
		return nil, errors.New("failed to create session")
	}

	return &dto.GitHubLoginResponse{
		Message:      "GitHub login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}