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
	"gorm.io/gorm"
)

type OAuthService struct {
	userRepo repository.UserRepository
	tfaRepo  repository.TwoFactorRepository
	redis    *redis.Client
}

func NewOAuthService(redisClient *redis.Client) *OAuthService {
	return &OAuthService{
		userRepo: repository.NewUserRepository(),
		tfaRepo:  repository.NewTwoFactorRepository(),
		redis:    redisClient,
	}
}

func stringPtr(s string) *string {
	return &s
}

// checkTwoFactor returns (temporaryToken, requiresTwoFactor, error).
// FAILS CLOSED: any real DB error is returned as error, not treated as "no 2FA".
func (s *OAuthService) checkTwoFactor(user *models.User) (string, bool, error) {
	if !user.TwoFactorEnabled {
		return "", false, nil
	}

	record, err := s.tfaRepo.GetByUserID(user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if record.Status != models.TwoFAStatusEnabled {
		return "", false, nil
	}

	tempToken, err := utils.GenerateTemporary2FAToken(user.ID.String())
	if err != nil {
		return "", false, err
	}
	return tempToken, true, nil
}

func splitName(fullName string) (string, string) {
	firstName := fullName
	lastName := ""
	if parts := strings.Split(fullName, " "); len(parts) > 1 {
		firstName = parts[0]
		lastName = strings.Join(parts[1:], " ")
	}
	return firstName, lastName
}

func (s *OAuthService) GoogleLogin(req dto.GoogleLoginRequest, device, ipAddress, userAgent string) (*dto.GoogleLoginResponse, string, error) {

	req.Email = utils.NormalizeEmail(req.Email)

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", errors.New("something went wrong please try again")
	}

	if user == nil {
		firstName, lastName := splitName(req.FullName)

		newUser := &models.User{
			Email:         req.Email,
			FirstName:     firstName,
			LastName:      lastName,
			PasswordHash:  "",
			Status:        models.StatusActive,
			EmailVerified: true,
			OAuthProvider: stringPtr("google"),
			OAuthID:       stringPtr(req.GoogleID),
		}

		err = s.userRepo.CreateUser(newUser)
		if err != nil {
			if utils.IsDuplicateKeyError(err) {
				// race: someone else created this user between our GetUserByEmail and CreateUser.
				// self-heal by re-fetching instead of erroring.
				user, err = s.userRepo.GetUserByEmail(req.Email)
				if err != nil {
					return nil, "", errors.New("something went wrong please try again")
				}
			} else {
				return nil, "", err
			}
		} else {
			user = newUser
			profile := &models.IndividualProfile{
				UserID:        user.ID,
				PublicUrlSlug: utils.GenerateSlug(firstName),
			}
			if err := s.userRepo.CreateProfile(profile); err != nil {
				return nil, "", err
			}
		}
	}

	// user now guaranteed non-nil (new, raced-and-refetched, or pre-existing)
	if user.OAuthID == nil {
		user.OAuthProvider = stringPtr("google")
		user.OAuthID = stringPtr(req.GoogleID)
	}
	if user.Status == models.StatusPendingDeletion {
		user.Status = models.StatusActive
		user.ScheduledDeletionDate = nil
	}
	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, "", err
	}
	if user.Status != models.StatusActive {
		return nil, "", errors.New("user account is not active")
	}

	tempToken, requires2FA, err := s.checkTwoFactor(user)
	if err != nil {
		return nil, "", err
	}
	if requires2FA {
		return &dto.GoogleLoginResponse{
			Message:           "2FA verification required",
			RequiresTwoFactor: true,
			TemporaryToken:    tempToken,
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
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
		IsRevoked:    false,
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, "", errors.New("failed to create session")
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String(), session.ID.String())
	if err != nil {
		return nil, "", err
	}

	return &dto.GoogleLoginResponse{
		Message:     "Google login successful",
		AccessToken: accessToken,
		ExpiresIn:   int(utils.AccessTokenTTL.Seconds()),
	}, refreshToken, nil
}

func (s *OAuthService) GitHubLogin(req dto.GitHubLoginRequest, device, ipAddress, userAgent string) (*dto.GitHubLoginResponse, string, error) {

	req.Email = utils.NormalizeEmail(req.Email)

	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", errors.New("something went wrong please try again")
	}

	if user == nil {
		firstName, lastName := splitName(req.FullName)

		newUser := &models.User{
			Email:         req.Email,
			FirstName:     firstName,
			LastName:      lastName,
			PasswordHash:  "",
			Status:        models.StatusActive,
			EmailVerified: true,
			OAuthProvider: stringPtr("github"),
			OAuthID:       stringPtr(req.GitHubID),
		}

		err = s.userRepo.CreateUser(newUser)
		if err != nil {
			if utils.IsDuplicateKeyError(err) {
				user, err = s.userRepo.GetUserByEmail(req.Email)
				if err != nil {
					return nil, "", errors.New("something went wrong please try again")
				}
			} else {
				return nil, "", err
			}
		} else {
			user = newUser
			profile := &models.IndividualProfile{
				UserID:        user.ID,
				PublicUrlSlug: utils.GenerateSlug(firstName),
				AvatarURL:     req.Avatar,
			}
			if err := s.userRepo.CreateProfile(profile); err != nil {
				return nil, "", err
			}
		}
	}

	if user.OAuthID == nil {
		user.OAuthProvider = stringPtr("github")
		user.OAuthID = stringPtr(req.GitHubID)
	}
	if user.Status == models.StatusPendingDeletion {
		user.Status = models.StatusActive
		user.ScheduledDeletionDate = nil
	}
	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, "", err
	}
	if user.Status != models.StatusActive {
		return nil, "", errors.New("user account is not active")
	}

	tempToken, requires2FA, err := s.checkTwoFactor(user)
	if err != nil {
		return nil, "", err
	}
	if requires2FA {
		return &dto.GitHubLoginResponse{
			Message:           "2FA verification required",
			RequiresTwoFactor: true,
			TemporaryToken:    tempToken,
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
		ExpiresAt:    time.Now().Add(15 * 24 * time.Hour),
		IsRevoked:    false,
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, "", errors.New("failed to create session")
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String(), session.ID.String())
	if err != nil {
		return nil, "", err
	}

	return &dto.GitHubLoginResponse{
		Message:     "GitHub login successful",
		AccessToken: accessToken,
		ExpiresIn:   int(utils.AccessTokenTTL.Seconds()),
	}, refreshToken, nil
}
