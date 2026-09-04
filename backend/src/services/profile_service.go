package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("User is not found")
var ErrProfileNotFound = errors.New("profile not found, create it first")
var ErrProfileAlreadyExists = errors.New("profile already exists, use update instead")
var ErrNothingToDelete = errors.New("nothing to delete")
var ErrInternal = errors.New("internal server error")
var ErrForbidden = errors.New("forbidden: account type mismatch")
var ErrValidation = errors.New("validation error")
var ErrAccountTypeNotSet = errors.New("account type not set")
var ErrInvalidAccountType = errors.New("invalid account type")
var ErrInvalidPassword = errors.New("invalid password")

const maxSlugRetries = 3

type ProfileService struct {
	userRepo repository.UserRepository
}

func NewProfileService() *ProfileService {
	return &ProfileService{
		userRepo: repository.NewUserRepository(),
	}
}

// ================= INDIVIDUAL =================
func (s *ProfileService) CreateIndividualProfile(userID string, req dto.CreateIndividualProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeIndividual {
		return "", ErrForbidden
	}

	_, err = s.userRepo.GetIndividualProfileByUserID(userID)
	if err == nil {
		return "", ErrProfileAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	profile := &models.IndividualProfile{UserID: user.ID}
	profile.Phone = req.Phone

	if err := applyIndividualFields(profile, dto.UpdateIndividualProfileRequest{
		Country:           req.Country,
		City:              req.City,
		TimeZone:          req.TimeZone,
		Headline:          req.Headline,
		Bio:               req.Bio,
		ExperienceLevel:   req.ExperienceLevel,
		Availability:      req.Availability,
		Skills:            req.Skills,
		ToolsTechnologies: req.ToolsTechnologies,
		ServiceCategories: req.ServiceCategories,
		PortfolioURL:      req.PortfolioURL,
		AvatarURL:         req.AvatarURL,
		GithubURL:         req.GithubURL,
		LinkedinURL:       req.LinkedinURL,
		DateOfBirth:       req.DateOfBirth,
		Gender:            req.Gender,
		PreferredLanguage: req.PreferredLanguage,
		TermsConfirmed:    req.TermsConfirmed,
		ResumeURL:         req.ResumeURL,
	}); err != nil {
		return "", err
	}

	for attempt := 1; attempt <= maxSlugRetries; attempt++ {
		slug, err := s.generateUniqueSlug(user.FirstName + " " + user.LastName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug

		err = s.userRepo.UpdateIndividualProfile(profile)
		if err == nil {
			s.checkAndActivateUser(userID)
			return profile.PublicUrlSlug, nil
		}

		constraint := utils.GetDuplicateKeyConstraint(err)
		switch {
		case constraint == "":
			log.Printf("profile save failed for user_id=%s: %v", userID, err)
			return "", ErrInternal
		case strings.Contains(constraint, "phone"):
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		case strings.Contains(constraint, "slug"), strings.Contains(constraint, "public_url"):
			log.Printf("slug collision for user_id=%s, retrying (attempt %d/%d)", userID, attempt, maxSlugRetries)
			continue // race condition on slug — retry with a fresh one
		default:
			log.Printf("profile save failed (unhandled constraint %s) for user_id=%s: %v", constraint, userID, err)
			return "", ErrInternal
		}
	}

	return "", fmt.Errorf("%w: unable to create a unique profile URL, please try again", ErrValidation)
}

func (s *ProfileService) UpdateIndividualProfile(userID string, req dto.UpdateIndividualProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeIndividual {
		return "", ErrForbidden
	}

	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrProfileNotFound
		}
		return "", err
	}

	if err := applyIndividualFields(profile, req); err != nil {
		return "", err
	}

	if err := s.userRepo.UpdateIndividualProfile(profile); err != nil {
		if utils.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		}
		log.Printf("profile save failed for user_id=%s: %v", userID, err)
		return "", ErrInternal
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

// applyIndividualFields selectively updates only the fields present (non-nil) in req.
func applyIndividualFields(profile *models.IndividualProfile, req dto.UpdateIndividualProfileRequest) error {
	if req.DateOfBirth != nil {
		t, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: invalid date_of_birth format, expected YYYY-MM-DD", ErrValidation)
		}
		profile.DateOfBirth = &t
	}
	if req.Gender != nil {
		profile.Gender = models.Gender(*req.Gender)
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = *req.AvatarURL
	}
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.Country != nil {
		profile.Country = *req.Country
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.Headline != nil {
		profile.Headline = *req.Headline
	}
	if req.PreferredLanguage != nil {
		profile.PreferredLanguage = *req.PreferredLanguage
	}
	if req.TimeZone != nil {
		profile.TimeZone = *req.TimeZone
	}
	if req.ExperienceLevel != nil {
		profile.ExperienceLevel = *req.ExperienceLevel
	}
	if req.Availability != nil {
		profile.Availability = *req.Availability
	}
	if req.Skills != nil {
		profile.Skills = *req.Skills
	}
	if req.ToolsTechnologies != nil {
		profile.ToolsTechnologies = *req.ToolsTechnologies
	}
	if req.ServiceCategories != nil {
		profile.ServiceCategories = *req.ServiceCategories
	}
	if req.PortfolioURL != nil {
		profile.PortfolioURL = *req.PortfolioURL
	}
	if req.GithubURL != nil {
		profile.GithubURL = *req.GithubURL
	}
	if req.LinkedinURL != nil {
		profile.LinkedinURL = *req.LinkedinURL
	}
	if req.ResumeURL != nil {
		profile.ResumeURL = *req.ResumeURL
	}
	if req.TermsConfirmed != nil {
		profile.TermsConfirmed = *req.TermsConfirmed
	}
	return nil
}

// ================= AGENCY =================

func (s *ProfileService) CreateAgencyProfile(userID string, req dto.CreateAgencyProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeAgencyAdmin {
		return "", ErrForbidden
	}

	_, err = s.userRepo.GetAgencyProfileByUserID(userID)
	if err == nil {
		return "", ErrProfileAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	if req.AgencyName == "" {
		return "", fmt.Errorf("%w: agency_name is required to create a profile", ErrValidation)
	}

	profile := &models.AgencyProfile{UserID: user.ID}
	profile.AgencyName = req.AgencyName
	profile.Phone = req.Phone

	applyAgencyFields(profile, dto.UpdateAgencyProfileRequest{
		LogoURL:         req.LogoURL,
		Description:     req.Description,
		WebsiteURL:      req.WebsiteURL,
		ServicesOffered: req.ServicesOffered,
		Industries:      req.Industries,
		TeamSize:        req.TeamSize,
		ContactName:     req.ContactName,
		Country:         req.Country,
		City:            req.City,
		TimeZone:        req.TimeZone,
	})

	for attempt := 1; attempt <= maxSlugRetries; attempt++ {
		slug, err := s.generateUniqueSlug(req.AgencyName) // FIXED: was user.FirstName+LastName — agency slug must come from the agency's own name
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug

		err = s.userRepo.UpdateAgencyProfile(profile)
		if err == nil {
			s.checkAndActivateUser(userID)
			return profile.PublicUrlSlug, nil
		}

		constraint := utils.GetDuplicateKeyConstraint(err)
		switch {
		case constraint == "":
			log.Printf("profile save failed for user_id=%s: %v", userID, err)
			return "", ErrInternal
		case strings.Contains(constraint, "phone"):
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		case strings.Contains(constraint, "slug"), strings.Contains(constraint, "public_url"):
			log.Printf("slug collision for user_id=%s, retrying (attempt %d/%d)", userID, attempt, maxSlugRetries)
			continue // race condition on slug — retry with a fresh one
		default:
			log.Printf("profile save failed (unhandled constraint %s) for user_id=%s: %v", constraint, userID, err)
			return "", ErrInternal
		}
	}

	return "", fmt.Errorf("%w: unable to create a unique profile URL, please try again", ErrValidation)
}

func (s *ProfileService) UpdateAgencyProfile(userID string, req dto.UpdateAgencyProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeAgencyAdmin {
		return "", ErrForbidden
	}

	profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrProfileNotFound
		}
		return "", err
	}

	applyAgencyFields(profile, req)

	if err := s.userRepo.UpdateAgencyProfile(profile); err != nil {
		if utils.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		}
		log.Printf("agency profile save failed for user_id=%s: %v", userID, err)
		return "", ErrInternal
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

func applyAgencyFields(profile *models.AgencyProfile, req dto.UpdateAgencyProfileRequest) {
	if req.AgencyName != nil {
		profile.AgencyName = *req.AgencyName
	}
	if req.LogoURL != nil {
		profile.LogoURL = *req.LogoURL
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.WebsiteURL != nil {
		profile.WebsiteURL = *req.WebsiteURL
	}
	if req.ServicesOffered != nil {
		profile.ServicesOffered = *req.ServicesOffered
	}
	if req.Industries != nil {
		profile.Industries = *req.Industries
	}
	if req.TeamSize != nil {
		profile.TeamSize = *req.TeamSize
	}
	if req.ContactName != nil {
		profile.ContactName = *req.ContactName
	}
	if req.Country != nil {
		profile.Country = *req.Country
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.TimeZone != nil {
		profile.TimeZone = *req.TimeZone
	}
}

// ================= CLIENT =================

func (s *ProfileService) CreateClientProfile(userID string, req dto.CreateClientProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeClientAdmin {
		return "", ErrForbidden
	}

	_, err = s.userRepo.GetClientProfileByUserID(userID)
	if err == nil {
		return "", ErrProfileAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	if req.CompanyName == "" {
		return "", fmt.Errorf("%w: company_name is required to create a profile", ErrValidation)
	}

	profile := &models.ClientProfile{UserID: user.ID}
	profile.CompanyName = req.CompanyName
	profile.Phone = req.Phone

	applyClientFields(profile, dto.UpdateClientProfileRequest{
		LogoURL:      req.LogoURL,
		Industry:     req.Industry,
		WebsiteURL:   req.WebsiteURL,
		ProjectTypes: req.ProjectTypes,
		BudgetRange:  req.BudgetRange,
		TeamSize:     req.TeamSize,
		Country:      req.Country,
		City:         req.City,
		TimeZone:     req.TimeZone,
	})

	for attempt := 1; attempt <= maxSlugRetries; attempt++ {
		slug, err := s.generateUniqueSlug(req.CompanyName)
		if err != nil {
			return "", err
		}
		profile.PublicUrlSlug = slug

		err = s.userRepo.UpdateClientProfile(profile)
		if err == nil {
			s.checkAndActivateUser(userID)
			return profile.PublicUrlSlug, nil
		}

		constraint := utils.GetDuplicateKeyConstraint(err)
		switch {
		case constraint == "":
			log.Printf("client profile save failed for user_id=%s: %v", userID, err)
			return "", ErrInternal
		case strings.Contains(constraint, "phone"):
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		case strings.Contains(constraint, "slug"), strings.Contains(constraint, "public_url"):
			log.Printf("slug collision for user_id=%s, retrying (attempt %d/%d)", userID, attempt, maxSlugRetries)
			continue
		default:
			log.Printf("client profile save failed (unhandled constraint %s) for user_id=%s: %v", constraint, userID, err)
			return "", ErrInternal
		}
	}

	return "", fmt.Errorf("%w: unable to create a unique profile URL, please try again", ErrValidation)
}

func (s *ProfileService) UpdateClientProfile(userID string, req dto.UpdateClientProfileRequest) (string, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if user.AccountType == nil || *user.AccountType != models.AccountTypeClientAdmin {
		return "", ErrForbidden
	}

	profile, err := s.userRepo.GetClientProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrProfileNotFound
		}
		return "", err
	}

	applyClientFields(profile, req)

	if err := s.userRepo.UpdateClientProfile(profile); err != nil {
		if utils.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: phone number already in use", ErrValidation)
		}
		log.Printf("client profile save failed for user_id=%s: %v", userID, err)
		return "", ErrInternal
	}

	s.checkAndActivateUser(userID)
	return profile.PublicUrlSlug, nil
}

func applyClientFields(profile *models.ClientProfile, req dto.UpdateClientProfileRequest) {
	if req.CompanyName != nil {
		profile.CompanyName = *req.CompanyName
	}
	if req.LogoURL != nil {
		profile.LogoURL = *req.LogoURL
	}
	if req.Industry != nil {
		profile.Industry = *req.Industry
	}
	if req.WebsiteURL != nil {
		profile.WebsiteURL = *req.WebsiteURL
	}
	if req.ProjectTypes != nil {
		profile.ProjectTypes = *req.ProjectTypes
	}
	if req.BudgetRange != nil {
		profile.BudgetRange = *req.BudgetRange
	}
	if req.TeamSize != nil {
		profile.TeamSize = *req.TeamSize
	}
	if req.Country != nil {
		profile.Country = *req.Country
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.TimeZone != nil {
		profile.TimeZone = *req.TimeZone
	}
}

// ================= SHARED HELPERS =================

func (s *ProfileService) checkAndActivateUser(userID string) {
	verification, err := s.userRepo.GetVerificationRecordByUserID(userID)
	if err != nil {
		log.Printf("checkAndActivateUser: failed to load verification for user_id=%s: %v", userID, err)
		return
	}
	if verification == nil || verification.Status != models.VerificationApproved {
		return
	}
	if err := s.userRepo.UpdateUserStatus(userID, string(models.StatusActive)); err != nil {
		log.Printf("checkAndActivateUser: failed to activate user_id=%s: %v", userID, err)
	}
}

func (s *ProfileService) generateUniqueSlug(fullName string) (string, error) {
	baseSlug := slugify(fullName)

	if baseSlug == "" {
		baseSlug = "user"
	}

	// Attempt 1: clean name, no suffix.
	taken, err := s.isSlugTaken(baseSlug)
	if err != nil {
		return "", err
	}
	if !taken {
		return baseSlug, nil
	}

	// Attempt 2-6: sequential number suffix (e.g. fuzel-patel-2, fuzel-patel-3)
	for suffixNum := 2; suffixNum <= 6; suffixNum++ {
		candidate := fmt.Sprintf("%s-%d", baseSlug, suffixNum)
		taken, err := s.isSlugTaken(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}

	// Fallback: random suffix
	for attempt := 0; attempt < 5; attempt++ {
		id, err := gonanoid.New(6)
		if err != nil {
			return "", err
		}
		candidate := baseSlug + "-" + id
		taken, err := s.isSlugTaken(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}

	return "", errors.New("failed to generate unique slug after multiple attempts")
}

func slugify(input string) string {
	s := strings.ToLower(input)
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (s *ProfileService) GetMyProfile(userID string) (*dto.GetMyProfileResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.AccountType == nil {
		return nil, ErrAccountTypeNotSet
	}

	switch *user.AccountType {
	case models.AccountTypeIndividual:
		profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
		if err != nil {
			return nil, err
		}
		return &dto.GetMyProfileResponse{
			AccountType: string(*user.AccountType),
			Individual:  s.buildMyIndividualProfile(profile),
		}, nil
	case models.AccountTypeAgencyAdmin:
		profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
		if err != nil {
			return nil, err
		}
		return &dto.GetMyProfileResponse{
			AccountType: string(*user.AccountType),
			Agency:      s.buildMyAgencyProfile(profile),
		}, nil
	case models.AccountTypeClientAdmin:
		profile, err := s.userRepo.GetClientProfileByUserID(userID)
		if err != nil {
			return nil, err
		}
		return &dto.GetMyProfileResponse{
			AccountType: string(*user.AccountType),
			Client:      s.buildMyClientProfile(profile),
		}, nil
	default:
		return nil, ErrInvalidAccountType
	}
}

func (s *ProfileService) buildMyIndividualProfile(profile *models.IndividualProfile) *dto.MyIndividualProfile {
	return &dto.MyIndividualProfile{
		PublicUrlSlug:     profile.PublicUrlSlug,
		Phone:             profile.Phone,
		DateOfBirth:       formatDate(profile.DateOfBirth),
		Gender:            string(profile.Gender),
		AvatarURL:         profile.AvatarURL,
		Bio:               profile.Bio,
		Country:           profile.Country,
		City:              profile.City,
		Headline:          profile.Headline,
		PreferredLanguage: profile.PreferredLanguage,
		TimeZone:          profile.TimeZone,
		CountryCode:       profile.CountryCode,
		ExperienceLevel:   profile.ExperienceLevel,
		Availability:      profile.Availability,
		Skills:            profile.Skills,
		ToolsTechnologies: profile.ToolsTechnologies,
		ServiceCategories: profile.ServiceCategories,
		PortfolioURL:      profile.PortfolioURL,
		GithubURL:         profile.GithubURL,
		LinkedinURL:       profile.LinkedinURL,
		ResumeURL:         profile.ResumeURL,
	}
}

func (s *ProfileService) buildMyAgencyProfile(profile *models.AgencyProfile) *dto.MyAgencyProfile {
	return &dto.MyAgencyProfile{
		PublicUrlSlug:   profile.PublicUrlSlug,
		AgencyName:      profile.AgencyName,
		LogoURL:         profile.LogoURL,
		Description:     profile.Description,
		WebsiteURL:      profile.WebsiteURL,
		ServicesOffered: profile.ServicesOffered,
		Industries:      profile.Industries,
		TeamSize:        profile.TeamSize,
		ContactName:     profile.ContactName,
		Phone:           profile.Phone,
		RegistrationNo:  profile.RegistrationNo,
		Country:         profile.Country,
		City:            profile.City,
		TimeZone:        profile.TimeZone,
		CountryCode:     profile.CountryCode,
	}
}

func (s *ProfileService) buildMyClientProfile(profile *models.ClientProfile) *dto.MyClientProfile {
	return &dto.MyClientProfile{
		PublicUrlSlug: profile.PublicUrlSlug,
		CompanyName:   profile.CompanyName,
		LogoURL:       profile.LogoURL,
		Industry:      profile.Industry,
		WebsiteURL:    profile.WebsiteURL,
		ProjectTypes:  profile.ProjectTypes,
		BudgetRange:   profile.BudgetRange,
		TeamSize:      profile.TeamSize,
		ContactName:   profile.ContactName,
		Phone:         profile.Phone,
		Country:       profile.Country,
		City:          profile.City,
		TimeZone:      profile.TimeZone,
		CountryCode:   profile.CountryCode,
	}
}

// formatDate renders a time as YYYY-MM-DD to keep the public/API contract
// consistent with the accepted input format.
func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func (s *ProfileService) SetAccountType(req dto.SetAccountTypeRequest) error {
	user, err := s.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		return ErrInvalidPassword
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return ErrInvalidPassword
	}

	if !user.EmailVerified {
		return fmt.Errorf("%w: please verify your email first", ErrValidation)
	}

	if user.AccountType != nil && *user.AccountType != "" {
		return ErrProfileAlreadyExists
	}

	return s.userRepo.WithTransaction(func(txRepo repository.UserRepository) error {
		if err := txRepo.UpdateAccountType(user.ID.String(), models.AccountType(req.AccountType)); err != nil {
			return err
		}
		if err := txRepo.AddUserPoints(user.ID.String(), 20); err != nil {
			return err
		}
		return txRepo.UpdateUserStatus(user.ID.String(), string(models.StatusActive))
	})
}

func (s *ProfileService) DeleteAvatar(userID string) error {
	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProfileNotFound
		}
		return err
	}

	if profile.AvatarURL == "" {
		return ErrNothingToDelete
	}

	publicID := extractCloudinaryPublicID(profile.AvatarURL)
	if publicID != "" {
		_ = utils.DeleteFromCloudinary(publicID, "image")
	}

	profile.AvatarURL = ""
	return s.userRepo.UpdateIndividualProfile(profile)
}

func (s *ProfileService) DeleteLogo(userID string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.AccountType != nil && *user.AccountType == models.AccountTypeAgencyAdmin {
		profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProfileNotFound
			}
			return err
		}
		if profile.LogoURL == "" {
			return ErrNothingToDelete
		}
		publicID := extractCloudinaryPublicID(profile.LogoURL)
		if publicID != "" {
			_ = utils.DeleteFromCloudinary(publicID, "image")
		}
		profile.LogoURL = ""
		return s.userRepo.UpdateAgencyProfile(profile)
	} else if user.AccountType != nil && *user.AccountType == models.AccountTypeClientAdmin {
		profile, err := s.userRepo.GetClientProfileByUserID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProfileNotFound
			}
			return err
		}
		if profile.LogoURL == "" {
			return ErrNothingToDelete
		}
		publicID := extractCloudinaryPublicID(profile.LogoURL)
		if publicID != "" {
			_ = utils.DeleteFromCloudinary(publicID, "image")
		}
		profile.LogoURL = ""
		return s.userRepo.UpdateClientProfile(profile)
	}

	return ErrForbidden
}

func (s *ProfileService) DeleteResume(userID string) error {
	profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProfileNotFound
		}
		return err
	}

	if profile.ResumeURL == "" {
		return ErrNothingToDelete
	}

	publicID := extractCloudinaryPublicID(profile.ResumeURL)
	if publicID != "" {
		_ = utils.DeleteFromCloudinary(publicID, "raw")
	}

	profile.ResumeURL = ""
	return s.userRepo.UpdateIndividualProfile(profile)
}

func extractCloudinaryPublicID(url string) string {
	parts := strings.Split(url, "/upload/")
	if len(parts) != 2 {
		return ""
	}
	afterUpload := parts[1]
	segments := strings.SplitN(afterUpload, "/", 2)
	if len(segments) != 2 {
		return ""
	}
	pathWithExt := segments[1]
	lastDot := strings.LastIndex(pathWithExt, ".")
	if lastDot == -1 {
		return pathWithExt
	}
	return pathWithExt[:lastDot]
}

func (s *ProfileService) GetPublicProfile(slug string) (*dto.GetPublicProfileResponse, error) {
	// Individual
	if profile, err := s.userRepo.GetIndividualProfileBySlug(slug); err == nil {
		built, err := s.buildIndividualPublicProfileResponse(profile)
		if err != nil {
			return nil, err
		}
		return &dto.GetPublicProfileResponse{
			AccountType: "individual",
			Individual:  built,
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("GetPublicProfile: individual lookup failed for slug=%s: %v", slug, err)
		return nil, ErrInternal
	}

	// Agency
	if profile, err := s.userRepo.GetAgencyProfileBySlug(slug); err == nil {
		built, err := s.buildAgencyPublicProfileResponse(profile)
		if err != nil {
			return nil, err
		}
		return &dto.GetPublicProfileResponse{
			AccountType: "agency",
			Agency:      built,
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("GetPublicProfile: agency lookup failed for slug=%s: %v", slug, err)
		return nil, ErrInternal
	}

	// Client
	if profile, err := s.userRepo.GetClientProfileBySlug(slug); err == nil {
		built, err := s.buildClientPublicProfileResponse(profile)
		if err != nil {
			return nil, err
		}
		return &dto.GetPublicProfileResponse{
			AccountType: "client",
			Client:      built,
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("GetPublicProfile: client lookup failed for slug=%s: %v", slug, err)
		return nil, ErrInternal
	}

	return nil, ErrProfileNotFound
}

func (s *ProfileService) buildIndividualPublicProfileResponse(profile *models.IndividualProfile) (*dto.PublicIndividualProfile, error) {
	if profile.ProfileVisibility != "public" {
		return nil, errors.New("this profile is private")
	}

	user, err := s.userRepo.GetUserByID(profile.UserID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	resp := &dto.PublicIndividualProfile{
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		AvatarURL:         profile.AvatarURL,
		Bio:               profile.Bio,
		Country:           profile.Country,
		Headline:          profile.Headline,
		PreferredLanguage: profile.PreferredLanguage,
		TimeZone:          profile.TimeZone,
		PublicUrlSlug:     profile.PublicUrlSlug,
		ExperienceLevel:   profile.ExperienceLevel,
		Availability:      profile.Availability,
		Skills:            profile.Skills,
		ToolsTechnologies: profile.ToolsTechnologies,
		ServiceCategories: profile.ServiceCategories,
		PortfolioURL:      profile.PortfolioURL,
		GithubURL:         profile.GithubURL,
		LinkedinURL:       profile.LinkedinURL,
		MemberSince:       user.CreatedAt.Format("January 2006"),
	}

	return resp, nil
}

func (s *ProfileService) buildAgencyPublicProfileResponse(profile *models.AgencyProfile) (*dto.PublicAgencyProfile, error) {
	if profile.ProfileVisibility != "public" { // ADDED — was missing, agency privacy leak
		return nil, errors.New("this profile is private")
	}

	user, err := s.userRepo.GetUserByID(profile.UserID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &dto.PublicAgencyProfile{
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		AgencyName:      profile.AgencyName,
		LogoURL:         profile.LogoURL,
		Description:     profile.Description,
		WebsiteURL:      profile.WebsiteURL,
		ServicesOffered: profile.ServicesOffered,
		Industries:      profile.Industries,
		TeamSize:        profile.TeamSize,
		Country:         profile.Country,
		TimeZone:        profile.TimeZone,
		PublicUrlSlug:   profile.PublicUrlSlug,
		MemberSince:     user.CreatedAt.Format("January 2006"),
	}, nil
}

func (s *ProfileService) buildClientPublicProfileResponse(profile *models.ClientProfile) (*dto.PublicClientProfile, error) {
	if profile.ProfileVisibility != "public" { // ADDED — was missing, client privacy leak
		return nil, errors.New("this profile is private")
	}

	user, err := s.userRepo.GetUserByID(profile.UserID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &dto.PublicClientProfile{
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		CompanyName:   profile.CompanyName,
		LogoURL:       profile.LogoURL,
		Industry:      profile.Industry,
		WebsiteURL:    profile.WebsiteURL,
		ProjectTypes:  profile.ProjectTypes,
		TeamSize:      profile.TeamSize,
		Country:       profile.Country,
		TimeZone:      profile.TimeZone,
		PublicUrlSlug: profile.PublicUrlSlug,
		MemberSince:   user.CreatedAt.Format("January 2006"),
	}, nil
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

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, errors.New("failed to generate export file")
	}

	filename := fmt.Sprintf("export-%s-%d", userID, time.Now().Unix())
	downloadURL, err := utils.UploadJSONToCloudinary(jsonData, filename)
	if err != nil {
		return nil, errors.New("failed to upload export file")
	}

	go utils.SendDataExportEmail(user.Email, user.FirstName, downloadURL)

	return &dto.ExportResponse{
		Message:     "Your data export is ready. A download link has been sent to your email.",
		DownloadURL: downloadURL,
		ExpiresIn:   "never (Cloudinary raw files are permanent — delete manually if needed)",
	}, nil
}

func (s *ProfileService) CheckSlugAvailability(slug string) (*dto.CheckSlugResponse, error) {
	taken, err := s.isSlugTaken(slug)
	if err != nil {
		return nil, err
	}

	if taken {
		alternatives := []string{
			slug + "-pro",
			slug + "-1",
			slug + "-official",
		}
		return &dto.CheckSlugResponse{
			Available:    false,
			Alternatives: alternatives,
		}, nil
	}

	return &dto.CheckSlugResponse{
		Available: true,
	}, nil
}

// isSlugTaken checks all three profile tables (individual, agency, client)
// since all public slugs share the same namespace.
func (s *ProfileService) isSlugTaken(slug string) (bool, error) {
	if _, err := s.userRepo.GetIndividualProfileBySlug(slug); err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if _, err := s.userRepo.GetAgencyProfileBySlug(slug); err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if _, err := s.userRepo.GetClientProfileBySlug(slug); err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	return false, nil
}

func (s *ProfileService) DeleteAccount(userID string, password string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return ErrInvalidPassword
	}

	user.Status = models.StatusPendingDeletion
	deletionDate := time.Now().Add(30 * 24 * time.Hour)
	user.ScheduledDeletionDate = &deletionDate

	return s.userRepo.WithTransaction(func(txRepo repository.UserRepository) error {
		if err := txRepo.UpdateUser(user); err != nil {
			return err
		}
		return txRepo.RevokeAllSessions(userID)
	})
}

func (s *ProfileService) UpdateAccountSettings(userID string, req dto.UpdateAccountRequest) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if req.NewPassword != "" {
		if !utils.CheckPassword(req.Password, user.PasswordHash) {
			return ErrInvalidPassword
		}
		hashed, err := utils.HashPassword(req.NewPassword)
		if err != nil {
			return errors.New("failed to hash password")
		}
		user.PasswordHash = hashed
	}

	return s.userRepo.UpdateUser(user)
}

func (s *ProfileService) UpdateNotifications(userID string, req dto.UpdateNotificationsRequest) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	b, _ := json.Marshal(req.Preferences)
	user.NotificationPreferences = datatypes.JSON(b)

	return s.userRepo.UpdateUser(user)
}

func (s *ProfileService) UpdatePrivacySettings(userID string, req dto.UpdatePrivacyRequest) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.AccountType == nil {
		return ErrAccountTypeNotSet
	}

	switch *user.AccountType {
	case models.AccountTypeIndividual:
		profile, err := s.userRepo.GetIndividualProfileByUserID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProfileNotFound
			}
			log.Printf("UpdatePrivacySettings: failed to fetch individual profile for user=%s: %v", userID, err)
			return ErrInternal
		}
		profile.ProfileVisibility = req.ProfileVisibility
		if err := s.userRepo.UpdateIndividualProfile(profile); err != nil {
			log.Printf("UpdatePrivacySettings: failed to save individual profile for user=%s: %v", userID, err)
			return ErrInternal
		}

	case models.AccountTypeAgencyAdmin:
		profile, err := s.userRepo.GetAgencyProfileByUserID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProfileNotFound
			}
			log.Printf("UpdatePrivacySettings: failed to fetch agency profile for user=%s: %v", userID, err)
			return ErrInternal
		}
		profile.ProfileVisibility = req.ProfileVisibility
		if err := s.userRepo.UpdateAgencyProfile(profile); err != nil {
			log.Printf("UpdatePrivacySettings: failed to save agency profile for user=%s: %v", userID, err)
			return ErrInternal
		}

	case models.AccountTypeClientAdmin:
		profile, err := s.userRepo.GetClientProfileByUserID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProfileNotFound
			}
			log.Printf("UpdatePrivacySettings: failed to fetch client profile for user=%s: %v", userID, err)
			return ErrInternal
		}
		profile.ProfileVisibility = req.ProfileVisibility
		if err := s.userRepo.UpdateClientProfile(profile); err != nil {
			log.Printf("UpdatePrivacySettings: failed to save client profile for user=%s: %v", userID, err)
			return ErrInternal
		}
	default:
		return ErrInvalidAccountType
	}

	pref := map[string]interface{}{"profile_visibility": req.ProfileVisibility}
	b, _ := json.Marshal(pref)
	user.PrivacySettings = datatypes.JSON(b)
	return s.userRepo.UpdateUser(user)
}
