package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

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

func (s *ProfileService) CreateOrUpdateIndividualProfile(userID string, req dto.CreateIndividualProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeIndividual {
		return "", errors.New("unauthorized: account type mismatch")
	}

	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	var isNew bool
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			profile = &models.IndividualProfile{
				UserID: user.ID,
			}
			isNew = true
		} else {
			return "", err
		}
	}

	if isNew || profile.PublicUrlSlug == "" {
		slug, err := s.generateUniqueSlug(user.FirstName + " " + user.LastName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug
	}

	// Update fields
	profile.Phone = req.Phone
	if req.DateOfBirth != nil {
		t, _ := time.Parse(time.RFC3339, *req.DateOfBirth) // simple parsing assumption
		profile.DateOfBirth = &t
	}
	profile.Gender = models.Gender(req.Gender)
	profile.AvatarURL = req.AvatarURL
	profile.Bio = req.Bio
	profile.Country = req.Country
	profile.State = req.State
	profile.City = req.City
	profile.Headline = req.Headline
	profile.PreferredLanguage = req.PreferredLanguage
	profile.TimeZone = req.TimeZone
	profile.CountryCode = req.CountryCode

	profile.ExperienceLevel = req.ExperienceLevel
	profile.Availability = req.Availability
	profile.Skills = req.Skills
	profile.ToolsTechnologies = req.ToolsTechnologies
	profile.ServiceCategories = req.ServiceCategories

	profile.PortfolioURL = req.PortfolioURL
	profile.GithubURL = req.GithubURL
	profile.LinkedinURL = req.LinkedinURL
	profile.ResumeURL = req.ResumeURL

	profile.TermsConfirmed = req.TermsConfirmed
	profile.ProfileVisibility = req.ProfileVisibility

	err = s.userRepo.UpdateIndividualProfile(profile)
	if err != nil {
		return "", err
	}

	if isNew {
		_ = s.userRepo.AddUserPoints(userID, 20)
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

func (s *ProfileService) CreateOrUpdateAgencyProfile(userID string, req dto.CreateAgencyProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeAgencyAdmin {
		return "", errors.New("unauthorized: account type mismatch")
	}

	profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
	var isNew bool
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			profile = &models.AgencyProfile{
				UserID: user.ID,
			}
			isNew = true
		} else {
			return "", err
		}
	}

	if isNew || profile.PublicUrlSlug == "" {
		slug, err := s.generateUniqueSlug(req.AgencyName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug
	}

	profile.AgencyName = req.AgencyName
	profile.LogoURL = req.LogoURL
	profile.Description = req.Description
	profile.WebsiteURL = req.WebsiteURL

	profile.ServicesOffered = req.ServicesOffered
	profile.Industries = req.Industries
	profile.TeamSize = req.TeamSize

	profile.ContactName = req.ContactName
	profile.Phone = req.Phone
	profile.RegistrationNo = req.RegistrationNo
	profile.Country = req.Country
	profile.State = req.State
	profile.City = req.City
	profile.TimeZone = req.TimeZone
	profile.CountryCode = req.CountryCode

	err = s.userRepo.UpdateAgencyProfile(profile)
	if err != nil {
		return "", err
	}

	if isNew {
		_ = s.userRepo.AddUserPoints(userID, 20)
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

func (s *ProfileService) CreateOrUpdateClientProfile(userID string, req dto.CreateClientProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeClientAdmin {
		return "", errors.New("unauthorized: account type mismatch")
	}

	profile, err := s.userRepo.GetClientProfileByUserID(userID)
	var isNew bool
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			profile = &models.ClientProfile{
				UserID: user.ID,
			}
			isNew = true
		} else {
			return "", err
		}
	}

	if isNew || profile.PublicUrlSlug == "" {
		slug, err := s.generateUniqueSlug(req.CompanyName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug
	}

	profile.CompanyName = req.CompanyName
	profile.LogoURL = req.LogoURL
	profile.Industry = req.Industry
	profile.WebsiteURL = req.WebsiteURL

	profile.ProjectTypes = req.ProjectTypes
	profile.BudgetRange = req.BudgetRange
	profile.TeamSize = req.TeamSize

	profile.ContactName = req.ContactName
	profile.Phone = req.Phone
	profile.Country = req.Country
	profile.State = req.State
	profile.City = req.City
	profile.TimeZone = req.TimeZone
	profile.CountryCode = req.CountryCode

	err = s.userRepo.UpdateClientProfile(profile)
	if err != nil {
		return "", err
	}

	if isNew {
		_ = s.userRepo.AddUserPoints(userID, 20)
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

func (s *ProfileService) checkAndActivateUser(userID string) {
	verification, err := s.userRepo.GetVerificationRecordByUserID(userID)
	if err == nil && verification != nil && verification.Status == models.VerificationApproved {
		_ = s.userRepo.UpdateUserStatus(userID, string(models.StatusActive))
	}
}

func (s *ProfileService) generateUniqueSlug(fullName string) (string, error) {
	baseSlug := strings.ToLower(fullName)
	reg := regexp.MustCompile("[^a-z0-9]+")
	baseSlug = reg.ReplaceAllString(baseSlug, "-")
	baseSlug = strings.Trim(baseSlug, "-")

	id, err := gonanoid.New(8)
	if err != nil {
		return "", err
	}

	if baseSlug == "" {
		return id, nil
	}
	return baseSlug + "-" + id, nil
}

func (s *ProfileService) GetMyProfile(userID string) (interface{}, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.AccountType == nil {
		return nil, errors.New("account type not set")
	}

	switch *user.AccountType {
	case models.AccountTypeIndividual:
		return s.userRepo.GetIndividualProfileByUserID(userID)
	case models.AccountTypeAgencyAdmin:
		return s.userRepo.GetAgencyProfileByUserID(userID)
	case models.AccountTypeClientAdmin:
		return s.userRepo.GetClientProfileByUserID(userID)
	default:
		return nil, errors.New("invalid account type")
	}
}

func (s *ProfileService) SetAccountType(userID string, accountType string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.AccountType != nil && *user.AccountType != "" {
		return errors.New("account type already set")
	}

	return s.userRepo.UpdateAccountType(userID, models.AccountType(accountType))
}

func (s *ProfileService) DeleteAvatar(userID string) error {
	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	if err != nil {
		return errors.New("profile not found")
	}

	if profile.AvatarURL == "" {
		return errors.New("no avatar to delete")
	}

	// Extract public ID and delete from Cloudinary
	publicID := extractCloudinaryPublicID(profile.AvatarURL)
	if publicID != "" {
		_ = utils.DeleteFromCloudinary(publicID, "image")
	}

	// Clear the URL in the database
	profile.AvatarURL = ""
	return s.userRepo.UpdateIndividualProfile(profile)
}

func (s *ProfileService) DeleteResume(userID string) error {
	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	if err != nil {
		return errors.New("profile not found")
	}

	if profile.ResumeURL == "" {
		return errors.New("no resume to delete")
	}

	// Extract public ID and delete from Cloudinary
	publicID := extractCloudinaryPublicID(profile.ResumeURL)
	if publicID != "" {
		_ = utils.DeleteFromCloudinary(publicID, "raw")
	}

	// Clear the URL in the database
	profile.ResumeURL = ""
	return s.userRepo.UpdateIndividualProfile(profile)
}

// extractCloudinaryPublicID extracts the public ID from a Cloudinary URL
// e.g. https://res.cloudinary.com/xxx/image/upload/v123/avatars/filename.png -> avatars/filename
// e.g. https://res.cloudinary.com/xxx/raw/upload/v123/resumes/filename.pdf -> resumes/filename
func extractCloudinaryPublicID(url string) string {
	// Find the "/upload/" part and take everything after the version segment
	parts := strings.Split(url, "/upload/")
	if len(parts) != 2 {
		return ""
	}
	// parts[1] is like "v1234567890/avatars/filename.png"
	afterUpload := parts[1]
	// Skip the version segment (starts with "v" followed by digits)
	segments := strings.SplitN(afterUpload, "/", 2)
	if len(segments) != 2 {
		return ""
	}
	// segments[1] is "avatars/filename.png" - remove the extension
	pathWithExt := segments[1]
	lastDot := strings.LastIndex(pathWithExt, ".")
	if lastDot == -1 {
		return pathWithExt
	}
	return pathWithExt[:lastDot]
}

func (s *ProfileService) GetPublicProfile(slug string) (*dto.PublicProfileResponse, error) {
	profile, err := s.userRepo.GetIndividualProfileBySlug(slug)
	if err != nil {
		return nil, errors.New("profile not found")
	}

	// Only show public profiles
	if profile.ProfileVisibility != "public" {
		return nil, errors.New("this profile is private")
	}

	user, err := s.userRepo.GetUserByID(profile.UserID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	accountType := ""
	if user.AccountType != nil {
		accountType = string(*user.AccountType)
	}

	dob := ""
	if profile.DateOfBirth != nil {
		dob = profile.DateOfBirth.Format(time.RFC3339)
	}

	phone := ""
	if profile.Phone != nil {
		phone = *profile.Phone
	}

	resp := &dto.PublicProfileResponse{
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		AccountType:       accountType,
		Points:            user.Points,
		Phone:             phone,
		DateOfBirth:       dob,
		Gender:            string(profile.Gender),
		AvatarURL:         profile.AvatarURL,
		Bio:               profile.Bio,
		Country:           profile.Country,
		State:             profile.State,
		City:              profile.City,
		Headline:          profile.Headline,
		PreferredLanguage: profile.PreferredLanguage,
		TimeZone:          profile.TimeZone,
		CountryCode:       profile.CountryCode,
		PublicUrlSlug:     profile.PublicUrlSlug,
		ExperienceLevel:   profile.ExperienceLevel,
		Availability:      profile.Availability,
		Skills:            profile.Skills,
		ToolsTechnologies: profile.ToolsTechnologies,
		ServiceCategories: profile.ServiceCategories,
		PortfolioURL:      profile.PortfolioURL,
		GithubURL:         profile.GithubURL,
		LinkedinURL:       profile.LinkedinURL,
		ResumeURL:         profile.ResumeURL,
		ProfileVisibility: profile.ProfileVisibility,
		MemberSince:       user.CreatedAt.Format("January 2006"),
	}

	return resp, nil
}

func (s *ProfileService) GetUserPoints(userID string) (*dto.UserPointsResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	accountType := ""
	if user.AccountType != nil {
		accountType = string(*user.AccountType)
	}

	// Check if profile is complete
	profileComplete := false
	if user.AccountType != nil {
		switch *user.AccountType {
		case models.AccountTypeIndividual:
			profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
			if err == nil && profile.Headline != "" && profile.Bio != "" {
				profileComplete = true
			}
		case models.AccountTypeAgencyAdmin:
			profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
			if err == nil && profile.AgencyName != "" {
				profileComplete = true
			}
		case models.AccountTypeClientAdmin:
			profile, err := s.userRepo.GetClientProfileByUserID(userID)
			if err == nil && profile.CompanyName != "" {
				profileComplete = true
			}
		}
	}

	return &dto.UserPointsResponse{
		Points:          user.Points,
		AccountType:     accountType,
		ProfileComplete: profileComplete,
	}, nil
}

func (s *ProfileService) ExportUserData(userID string) (*dto.ExportResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Collect all available profile data
	exportData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":             user.ID,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"email":          user.Email,
			"account_type":   user.AccountType,
			"status":         user.Status,
			"email_verified": user.EmailVerified,
			"points":         user.Points,
			"created_at":     user.CreatedAt,
			"updated_at":     user.UpdatedAt,
		},
	}

	if user.AccountType != nil {
		switch *user.AccountType {
		case models.AccountTypeIndividual:
			if profile, err := s.userRepo.GetIndividualProfileByUserID(userID); err == nil {
				exportData["profile"] = profile
			}
		case models.AccountTypeAgencyAdmin:
			if profile, err := s.userRepo.GetAgencyProfileByUserID(userID); err == nil {
				exportData["profile"] = profile
			}
		case models.AccountTypeClientAdmin:
			if profile, err := s.userRepo.GetClientProfileByUserID(userID); err == nil {
				exportData["profile"] = profile
			}
		}
	}

	exportData["exported_at"] = time.Now().UTC().Format(time.RFC3339)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, errors.New("failed to generate export file")
	}

	// Upload to Cloudinary
	filename := fmt.Sprintf("export-%s-%d", userID, time.Now().Unix())
	downloadURL, err := utils.UploadJSONToCloudinary(jsonData, filename)
	if err != nil {
		return nil, errors.New("failed to upload export file")
	}

	// Email the download link to the user
	go utils.SendDataExportEmail(user.Email, user.FirstName, downloadURL)

	return &dto.ExportResponse{
		Message:     "Your data export is ready. A download link has been sent to your email.",
		DownloadURL: downloadURL,
		ExpiresIn:   "never (Cloudinary raw files are permanent — delete manually if needed)",
	}, nil
}

