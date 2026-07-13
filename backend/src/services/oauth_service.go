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

func stringPtr(s string) *string {
	return &s
}

func (s *OAuthService) GoogleLogin(req dto.GoogleLoginRequest) (*dto.GoogleLoginResponse, error) {

	user, err := s.userRepo.GetUserByEmail(req.Email)

	if err != nil {

		firstName := req.FullName
		lastName := ""
		if parts := strings.Split(req.FullName, " "); len(parts) > 1 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}

		user = &models.User{
			Email:         req.Email,
			FirstName:     firstName,
			LastName:      lastName,
			PasswordHash:  "",
			Status:        models.StatusActive,
			EmailVerified: true,
			OAuthProvider: stringPtr("google"),
			OAuthID:       stringPtr(req.GoogleID),
		}

		err = s.userRepo.CreateUser(user)
		if err != nil {
			return nil, err
		}

		// Not using user.FullName here as it doesn't exist.
		// we already have firstName from above
		profile := &models.IndividualProfile{
			UserID: user.ID,
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

		firstName := req.FullName
		lastName := ""
		if parts := strings.Split(req.FullName, " "); len(parts) > 1 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}

		user = &models.User{
			Email:         req.Email,
			FirstName:     firstName,
			LastName:      lastName,
			PasswordHash:  "",
			Status:        models.StatusActive,
			EmailVerified: true,
			OAuthProvider: stringPtr("github"),
			OAuthID:       stringPtr(req.GitHubID),
		}

		err = s.userRepo.CreateUser(user)
		if err != nil {
			return nil, err
		}

		profile := &models.IndividualProfile{
			UserID: user.ID,
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
