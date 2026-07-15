package services

import (
	"errors"
	"regexp"
	"strings"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"techguild-backend/src/repository"
)

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
		return "", errors.New("User is not found")
	}

	// generate a unique public url slug from their fullname
	slug, err := s.generateUniqueSlug(user.FullName)
	if err != nil {
		return "", err
	}

	// if profile is not existed, so need to make it
	profile, err := s.userRepo.GetProfileByUserID(userID)
	if err != nil {
		profile = &models.UserProfile{
			UserID: user.ID,
		}
	}

	// populate the profile data
	profile.Headline = req.Headline
	profile.Bio = req.Bio
	profile.PreferredLanguage = req.PreferredLanguage
	profile.TimeZone = req.TimeZone
	profile.CountryCode = req.CountryCode
	profile.PublicUrlSlug = slug

	// save the update profile in db
	err = s.userRepo.UpdateProfile(profile)
	if err != nil {
		return "", err
	}
	// Check if verification is already , so we have to make it active
	verification, err := s.userRepo.GetVerificationRecordByUserID(userID)
	if err == nil && verification != nil && verification.Status == models.VerificationApproved {
		// If verification is approved, update user status to active
		_ = s.userRepo.UpdateUserStatus(userID, string(models.StatusActive))
	}
	return slug, nil
}
func (s *ProfileService) generateUniqueSlug(fullName string) (string, error) {
	// slugify the name (lowercase, replace non-alphanumeric with hyphens)
	baseSlug := strings.ToLower(fullName)
	// detects that the emoji is NOT a lowercase letter or number.
	reg := regexp.MustCompile("[^a-z0-9]+")
	baseSlug = reg.ReplaceAllString(baseSlug, "-")
	baseSlug = strings.Trim(baseSlug, "-")

	// generate a random 8-character nanoid
	id, err := gonanoid.New(8)
	if err != nil {
		return "", err
	}

	// combine them: "rahul-gupta-x9z2p8q1"
	slug := baseSlug + "-" + id
	return slug, nil
}
