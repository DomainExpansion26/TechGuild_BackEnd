package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- CreateOrUpdateIndividualProfile (JSON body version) ----------

func CreateOrUpdateIndividualProfileHandler(ctx context.Context, input *dto.CreateIndividualProfileInput) (*dto.CreateIndividualProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateIndividualProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CreateIndividualProfileOutput{
		Body: dto.CreateProfileResponse{Message: "Individual profile updated successfully", PublicUrlSlug: slug},
	}, nil
}

// ---------- CreateOrUpdateAgencyProfile ----------

func CreateOrUpdateAgencyProfileHandler(ctx context.Context, input *dto.CreateAgencyProfileInput) (*dto.CreateAgencyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateAgencyProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CreateAgencyProfileOutput{
		Body: dto.CreateProfileResponse{Message: "Agency profile updated successfully", PublicUrlSlug: slug},
	}, nil
}

// ---------- CreateOrUpdateClientProfile ----------

func CreateOrUpdateClientProfileHandler(ctx context.Context, input *dto.CreateClientProfileInput) (*dto.CreateClientProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateClientProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CreateClientProfileOutput{
		Body: dto.CreateProfileResponse{Message: "Client profile updated successfully", PublicUrlSlug: slug},
	}, nil
}

// ---------- GetMyProfile ----------

func GetMyProfileHandler(ctx context.Context, input *dto.GetMyProfileInput) (*dto.GetMyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	profile, err := profileService.GetMyProfile(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetMyProfileOutput{Body: profile}, nil
}

// ---------- SetAccountType ----------

func SetAccountTypeHandler(ctx context.Context, input *dto.SetAccountTypeInput) (*dto.SetAccountTypeOutput, error) {
	profileService := services.NewProfileService()

	if err := profileService.SetAccountType(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SetAccountTypeOutput{
		Body: dto.CreateProfileResponse{Message: "account type set successfully"},
	}, nil
}

// ---------- UploadResume ----------

func UploadResumeHandler(ctx context.Context, input *dto.UploadResumeInput) (*dto.UploadResumeOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Resume

	if filepath.Ext(file.Filename) != ".pdf" {
		return nil, huma.Error400BadRequest("only PDF files are allowed")
	}

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadPDFToCloudinary(file, filename)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to upload to cloudinary: " + err.Error())
	}

	return &dto.UploadResumeOutput{
		Body: struct {
			Message   string `json:"message"`
			ResumeURL string `json:"resume_url"`
		}{Message: "resume uploaded successfully", ResumeURL: fileURL},
	}, nil
}

// ---------- UploadAvatar ----------

func UploadAvatarHandler(ctx context.Context, input *dto.UploadAvatarInput) (*dto.UploadAvatarOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Avatar

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, huma.Error400BadRequest("only JPG, PNG, and WebP images are allowed")
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return nil, huma.Error400BadRequest("file size exceeds 5MB limit")
	}

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadImageToCloudinary(file, filename)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to upload to cloudinary: " + err.Error())
	}

	return &dto.UploadAvatarOutput{
		Body: struct {
			Message   string `json:"message"`
			AvatarURL string `json:"avatar_url"`
		}{Message: "avatar uploaded successfully", AvatarURL: fileURL},
	}, nil
}

// ---------- UploadLogo ----------

func UploadLogoHandler(ctx context.Context, input *dto.UploadLogoInput) (*dto.UploadLogoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Logo

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, huma.Error400BadRequest("only JPG, PNG, and WebP images are allowed")
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return nil, huma.Error400BadRequest("file size exceeds 5MB limit")
	}

	filename := fmt.Sprintf("%s-logo-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadImageToCloudinary(file, filename)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to upload to cloudinary: " + err.Error())
	}

	return &dto.UploadLogoOutput{
		Body: struct {
			Message string `json:"message"`
			LogoURL string `json:"logo_url"`
		}{Message: "logo uploaded successfully", LogoURL: fileURL},
	}, nil
}

// ---------- DeleteAvatar / DeleteLogo / DeleteResume ----------

func DeleteAvatarHandler(ctx context.Context, input *dto.DeleteAvatarInput) (*dto.DeleteAvatarOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteAvatar(userID); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteAvatarOutput{
		Body: struct {
			Message string `json:"message"`
		}{Message: "avatar deleted successfully"},
	}, nil
}

func DeleteLogoHandler(ctx context.Context, input *dto.DeleteLogoInput) (*dto.DeleteLogoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteLogo(userID); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteLogoOutput{
		Body: struct {
			Message string `json:"message"`
		}{Message: "logo deleted successfully"},
	}, nil
}

func DeleteResumeHandler(ctx context.Context, input *dto.DeleteResumeInput) (*dto.DeleteResumeOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteResume(userID); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteResumeOutput{
		Body: struct {
			Message string `json:"message"`
		}{Message: "resume deleted successfully"},
	}, nil
}

// ---------- GetPublicProfile ----------

func GetPublicProfileHandler(ctx context.Context, input *dto.GetPublicProfileInput) (*dto.GetPublicProfileOutput, error) {
	profileService := services.NewProfileService()
	profile, err := profileService.GetPublicProfile(input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetPublicProfileOutput{Body: *profile}, nil
}

// ---------- GetUserPoints ----------

func GetUserPointsHandler(ctx context.Context, input *dto.GetUserPointsInput) (*dto.GetUserPointsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	points, err := profileService.GetUserPoints(userID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetUserPointsOutput{Body: *points}, nil
}

// ---------- ExportProfile ----------

func ExportProfileHandler(ctx context.Context, input *dto.ExportProfileInput) (*dto.ExportProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	result, err := profileService.ExportUserData(userID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ExportProfileOutput{Body: *result}, nil
}

// ---------- CreateOrUpdateProfile (dispatcher, multipart) ----------

func CreateOrUpdateProfileHandler(ctx context.Context, input *dto.CreateOrUpdateProfileInput) (*dto.CreateOrUpdateProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	accountType, err := profileService.GetUserAccountType(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	profileData := input.RawBody.Data().ProfileData

	switch accountType {
	case "individual":
		var req dto.CreateIndividualProfileRequest
		if err := json.Unmarshal([]byte(profileData), &req); err != nil {
			return nil, huma.Error400BadRequest("invalid profile data: " + err.Error())
		}
		slug, err := profileService.CreateOrUpdateIndividualProfile(userID, req)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &dto.CreateOrUpdateProfileOutput{Body: dto.CreateProfileResponse{Message: "Profile updated successfully", PublicUrlSlug: slug}}, nil
	case "agency":
		var req dto.CreateAgencyProfileRequest
		if err := json.Unmarshal([]byte(profileData), &req); err != nil {
			return nil, huma.Error400BadRequest("invalid profile data: " + err.Error())
		}
		slug, err := profileService.CreateOrUpdateAgencyProfile(userID, req)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &dto.CreateOrUpdateProfileOutput{Body: dto.CreateProfileResponse{Message: "Profile updated successfully", PublicUrlSlug: slug}}, nil
	case "client":
		var req dto.CreateClientProfileRequest
		if err := json.Unmarshal([]byte(profileData), &req); err != nil {
			return nil, huma.Error400BadRequest("invalid profile data: " + err.Error())
		}
		slug, err := profileService.CreateOrUpdateClientProfile(userID, req)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &dto.CreateOrUpdateProfileOutput{Body: dto.CreateProfileResponse{Message: "Profile updated successfully", PublicUrlSlug: slug}}, nil
	default:
		return nil, huma.Error400BadRequest("invalid account type")
	}
}

// ---------- CheckSlug ----------

func CheckSlugHandler(ctx context.Context, input *dto.CheckSlugInput) (*dto.CheckSlugOutput, error) {
	profileService := services.NewProfileService()
	resp, err := profileService.CheckSlugAvailability(input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CheckSlugOutput{Body: *resp}, nil
}

// ---------- DeleteProfileAccount ----------

func DeleteProfileAccountHandler(ctx context.Context, input *dto.DeleteProfileAccountInput) (*dto.DeleteProfileAccountOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteAccount(userID, input.Body.Password); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteProfileAccountOutput{
		Body: struct {
			Message string `json:"message"`
		}{Message: "account successfully scheduled for deletion"},
	}, nil
}
