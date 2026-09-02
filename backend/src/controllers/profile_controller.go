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
	"gorm.io/gorm"
)

// ---------- CreateIndividualProfile (JSON body version) ----------

func CreateIndividualProfileHandler(ctx context.Context, input *dto.IndividualProfileInput) (*dto.IndividualProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateIndividualProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.IndividualProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Individual profile created successfully",
			PublicUrlSlug: slug},
	}, nil
}

// ---------- UpdateIndividualProfile ----------

func UpdateIndividualProfileHandler(ctx context.Context, input *dto.UpdateIndividualProfileInput) (*dto.UpdateIndividualProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.UpdateIndividualProfile(userID, input.Body)

	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound("profile not found, create it first")
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdateIndividualProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Individual profile updated successfully",
			PublicUrlSlug: slug},
	}, nil
}

// ---------- CreateOrUpdateAgencyProfile ----------

func CreateAgencyProfileHandler(ctx context.Context, input *dto.AgencyProfileInput) (*dto.AgencyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateAgencyProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.AgencyProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Agency profile created successfully",
			PublicUrlSlug: slug},
	}, nil
}

func UpdateAgencyProfileHandler(ctx context.Context, input *dto.UpdateAgencyProfileInput) (*dto.UpdateAgencyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.UpdateAgencyProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound("profile not found, create it first")
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdateAgencyProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Agency profile updated successfully",
			PublicUrlSlug: slug},
	}, nil
}

// ---------- CreateOrUpdateClientProfile ----------
func CreateClientProfileHandler(ctx context.Context, input *dto.ClientProfileInput) (*dto.ClientProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateClientProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ClientProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Client profile created successfully",
			PublicUrlSlug: slug},
	}, nil
}

func UpdateClientProfileHandler(ctx context.Context, input *dto.UpdateClientProfileInput) (*dto.UpdateClientProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	slug, err := profileService.UpdateClientProfile(userID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound("profile not found, create it first")
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdateClientProfileOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Client profile updated successfully",
			PublicUrlSlug: slug},
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.Error404NotFound("profile not found, create it first")
		}
		if errors.Is(err, services.ErrAccountTypeNotSet) || errors.Is(err, services.ErrInvalidAccountType) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetMyProfileOutput{Body: *profile}, nil
}

// ---------- SetAccountType ----------

func SetAccountTypeHandler(ctx context.Context, input *dto.SetAccountTypeInput) (*dto.SetAccountTypeOutput, error) {
	profileService := services.NewProfileService()

	if err := profileService.SetAccountType(input.Body); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInvalidAccountType) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.SetAccountTypeOutput{
		Body: dto.SetAccountTypeResponse{Message: "account type set successfully"},
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

	const maxResumeSize = 5 * 1024 * 1024
	if file.Size > maxResumeSize {
		return nil, huma.Error400BadRequest("file size exceeds 5MB limit")
	}

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadPDFToCloudinary(file, filename)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to upload to cloudinary: " + err.Error())
	}

	return &dto.UploadResumeOutput{
		Body: dto.UploadResumeResponse{Message: "resume uploaded successfully", ResumeURL: fileURL},
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
		Body: dto.UploadAvatarResponse{Message: "avatar uploaded successfully", AvatarURL: fileURL},
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
		Body: dto.UploadLogoResponse{Message: "logo uploaded successfully", LogoURL: fileURL},
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
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrNothingToDelete) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteAvatarOutput{
		Body: dto.MessageResponse{Message: "avatar deleted successfully"},
	}, nil
}

func DeleteLogoHandler(ctx context.Context, input *dto.DeleteLogoInput) (*dto.DeleteLogoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteLogo(userID); err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrNothingToDelete) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteLogoOutput{
		Body: dto.MessageResponse{Message: "logo deleted successfully"},
	}, nil
}

func DeleteResumeHandler(ctx context.Context, input *dto.DeleteResumeInput) (*dto.DeleteResumeOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteResume(userID); err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrNothingToDelete) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteResumeOutput{
		Body: dto.MessageResponse{Message: "resume deleted successfully"},
	}, nil
}

// ---------- GetPublicProfile ----------

func GetPublicProfileHandler(ctx context.Context, input *dto.GetPublicProfileInput) (*dto.GetPublicProfileOutput, error) {
	profileService := services.NewProfileService()
	profile, err := profileService.GetPublicProfile(input.Slug)
	if err != nil {
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
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
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
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
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ExportProfileOutput{Body: *result}, nil
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

// ---------- DeprecatedProfileCreate ----------

func DeprecatedProfileCreateHandler(ctx context.Context, input *dto.DeprecatedProfileCreateInput) (*dto.DeprecatedProfileCreateOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()

	var slug string
	switch input.Body.AccountType {
	case "individual":
		var req dto.CreateIndividualProfileRequest
		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
			return nil, huma.Error400BadRequest("invalid request body")
		}
		slug, err = profileService.CreateIndividualProfile(userID, req)
	case "agency":
		var req dto.CreateAgencyProfileRequest
		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
			return nil, huma.Error400BadRequest("invalid request body")
		}
		slug, err = profileService.CreateAgencyProfile(userID, req)
	case "client":
		var req dto.CreateClientProfileRequest
		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
			return nil, huma.Error400BadRequest("invalid request body")
		}
		slug, err = profileService.CreateClientProfile(userID, req)
	default:
		return nil, huma.Error400BadRequest("unknown account_type: must be individual, agency, or client")
	}

	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeprecatedProfileCreateOutput{
		Body: dto.CreateProfileResponse{
			Message:       "Profile created successfully (deprecated: use POST /v1/profile/{type} instead)",
			PublicUrlSlug: slug,
		},
	}, nil
}

// ---------- DeleteProfileAccount ----------

func DeleteProfileAccountHandler(ctx context.Context, input *dto.DeleteProfileAccountInput) (*dto.DeleteProfileAccountOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.DeleteAccount(userID, input.Body.Password); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteProfileAccountOutput{
		Body: dto.MessageResponse{Message: "account successfully scheduled for deletion"},
	}, nil
}
