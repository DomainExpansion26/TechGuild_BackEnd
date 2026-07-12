package services

import (
	"errors"
	"regexp"
	"strings"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("User is not found")

type ProfileService struct {
	userRepo repository.UserRepository
}

func NewProfileService() *ProfileService {
	return &ProfileService{
		userRepo: repository.NewUserRepository(),
	}
}

func (s *ProfileService) CreateOrUpdateProfile(userID string, req dto.CreateProfileRequest) (string, error) {
	// get the user record to retrieve their full name
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	// if profile is not existed, so need to make it
	profile, err := s.userRepo.GetProfileByUserID(userID)
	var isNew bool
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			firstName := user.FullName
			if parts := strings.Split(user.FullName, " "); len(parts) > 0 {
				firstName = parts[0]
			}
			profile = &models.UserProfile{
				UserID:    user.ID,
				FirstName: firstName,
			}
			isNew = true
		} else {
			return "", err
		}
	}

	// generate a unique public url slug from their fullname only if creating a new profile or if empty
	if isNew || profile.PublicUrlSlug == "" {
		slug, err := s.generateUniqueSlug(user.FullName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug
	}

	// populate the profile data
	profile.Headline = req.Headline
	profile.Bio = req.Bio
	profile.PreferredLanguage = req.PreferredLanguage
	profile.TimeZone = req.TimeZone
	profile.CountryCode = req.CountryCode

	// save the update profile in db
	err = s.userRepo.UpdateProfile(profile)
	if err != nil {
		return "", err
	}

	// Check if verification is already, so we have to make it active
	verification, err := s.userRepo.GetVerificationRecordByUserID(userID)
	if err == nil && verification != nil && verification.Status == models.VerificationApproved {
		// If verification is approved, update user status to active
		err = s.userRepo.UpdateUserStatus(userID, string(models.StatusActive))
		if err != nil {
			return "", err
		}
	}
	return profile.PublicUrlSlug, nil
}

func (s *ProfileService) generateUniqueSlug(fullName string) (string, error) {
	// slugify the name (lowercase, replace non-alphanumeric with hyphens)
	baseSlug := strings.ToLower(fullName)
	reg := regexp.MustCompile("[^a-z0-9]+")
	baseSlug = reg.ReplaceAllString(baseSlug, "-")
	baseSlug = strings.Trim(baseSlug, "-")

	// generate a random 8-character nanoid
	id, err := gonanoid.New(8)
	if err != nil {
		return "", err
	}

	if baseSlug == "" {
		return id, nil
	}

	// combine them: "rahul-gupta-x9z2p8q1"
	slug := baseSlug + "-" + id
	return slug, nil
}
