package controllers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

type ProfileController struct {
	cfg *config.Config
}

func NewProfileController(cfg *config.Config) *ProfileController {
	return &ProfileController{cfg: cfg}
}

// ---------- CreateIndividualProfile (JSON body version) ----------

func (c *ProfileController) CreateIndividualProfileHandler(ctx context.Context, input *dto.IndividualProfileInput) (*dto.IndividualProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) UpdateIndividualProfileHandler(ctx context.Context, input *dto.UpdateIndividualProfileInput) (*dto.UpdateIndividualProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) CreateAgencyProfileHandler(ctx context.Context, input *dto.AgencyProfileInput) (*dto.AgencyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) UpdateAgencyProfileHandler(ctx context.Context, input *dto.UpdateAgencyProfileInput) (*dto.UpdateAgencyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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
func (c *ProfileController) CreateClientProfileHandler(ctx context.Context, input *dto.ClientProfileInput) (*dto.ClientProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) UpdateClientProfileHandler(ctx context.Context, input *dto.UpdateClientProfileInput) (*dto.UpdateClientProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) GetMyProfileHandler(ctx context.Context, input *dto.GetMyProfileInput) (*dto.GetMyProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	profile, err := profileService.GetMyProfile(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetMyProfileOutput{Body: *profile}, nil
}

// ---------- UploadResume ----------
func (c *ProfileController) UploadResumeHandler(ctx context.Context, input *dto.UploadResumeInput) (*dto.UploadResumeOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Resume

	if strings.ToLower(filepath.Ext(file.Filename)) != ".pdf" {
		return nil, huma.Error400BadRequest("only PDF files are allowed")
	}

	if file.Size > utils.MaxUploadBytes {
		return nil, huma.Error400BadRequest("file size exceeds 2MB limit")
	}

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	profileService := services.NewProfileService(c.cfg)
	fileURL, err := profileService.UploadAndSaveResume(userID, file, filename)
	if err != nil {
		return nil, mapProfileError(err)
	}

	return &dto.UploadResumeOutput{
		Body: dto.UploadResumeResponse{Message: "resume uploaded successfully", ResumeURL: fileURL},
	}, nil
}

// ---------- UploadAvatar ----------

func (c *ProfileController) UploadAvatarHandler(ctx context.Context, input *dto.UploadAvatarInput) (*dto.UploadAvatarOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Avatar

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, huma.Error400BadRequest("only JPG, PNG, and WebP images are allowed")
	}

	if file.Size > utils.MaxUploadBytes {
		return nil, huma.Error400BadRequest("file size exceeds 2MB limit")
	}

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	profileService := services.NewProfileService(c.cfg)
	fileURL, err := profileService.UploadAndSaveAvatar(userID, file, filename)
	if err != nil {
		return nil, mapProfileError(err)
	}

	return &dto.UploadAvatarOutput{
		Body: dto.UploadAvatarResponse{Message: "avatar uploaded successfully", AvatarURL: fileURL},
	}, nil
}

// ---------- UploadLogo ----------
func (c *ProfileController) UploadLogoHandler(ctx context.Context, input *dto.UploadLogoInput) (*dto.UploadLogoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()
	file := form.Logo

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, huma.Error400BadRequest("only JPG, PNG, and WebP images are allowed")
	}

	if file.Size > utils.MaxUploadBytes {
		return nil, huma.Error400BadRequest("file size exceeds 2MB limit")
	}

	filename := fmt.Sprintf("%s-logo-%d", userID, time.Now().Unix())

	profileService := services.NewProfileService(c.cfg)
	fileURL, err := profileService.UploadAndSaveLogo(userID, file, filename)
	if err != nil {
		return nil, mapProfileError(err)
	}

	return &dto.UploadLogoOutput{
		Body: dto.UploadLogoResponse{Message: "logo uploaded successfully", LogoURL: fileURL},
	}, nil
}

// ---------- DeleteAvatar / DeleteLogo / DeleteResume ----------

func (c *ProfileController) DeleteAvatarHandler(ctx context.Context, input *dto.DeleteAvatarInput) (*dto.DeleteAvatarOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.DeleteAvatar(userID); err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrNothingToDelete) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrForbidden) { // ← YE 3 LINE ADD KARO
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteAvatarOutput{
		Body: dto.MessageResponse{Message: "avatar deleted successfully"},
	}, nil
}

func (c *ProfileController) DeleteLogoHandler(ctx context.Context, input *dto.DeleteLogoInput) (*dto.DeleteLogoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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
		if errors.Is(err, services.ErrForbidden) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteLogoOutput{
		Body: dto.MessageResponse{Message: "logo deleted successfully"},
	}, nil
}

func (c *ProfileController) DeleteResumeHandler(ctx context.Context, input *dto.DeleteResumeInput) (*dto.DeleteResumeOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) GetPublicProfileHandler(ctx context.Context, input *dto.GetPublicProfileInput) (*dto.GetPublicProfileOutput, error) {
	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) GetUserPointsHandler(ctx context.Context, input *dto.GetUserPointsInput) (*dto.GetUserPointsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

func (c *ProfileController) ExportProfileHandler(ctx context.Context, input *dto.ExportProfileInput) (*dto.ExportProfileOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	result, err := profileService.ExportUserData(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrAccountTypeNotSet) {
			return nil, huma.Error400BadRequest("account type not set, cannot export profile")
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound("profile not found, create it first")
		}
		if errors.Is(err, services.ErrInvalidAccountType) {
			return nil, huma.Error400BadRequest("invalid account type, cannot export profile")
		}
		if errors.Is(err, services.ErrInternal) {
			return nil, huma.Error500InternalServerError("failed to export user data")
		}
		return nil, huma.Error500InternalServerError("failed to export user data: " + err.Error())
	}

	return &dto.ExportProfileOutput{Body: *result}, nil
}

// ---------- CheckSlug ----------

func (c *ProfileController) CheckSlugHandler(ctx context.Context, input *dto.CheckSlugInput) (*dto.CheckSlugOutput, error) {
	profileService := services.NewProfileService(c.cfg)
	resp, err := profileService.CheckSlugAvailability(input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CheckSlugOutput{Body: *resp}, nil
}

// // ---------- DeprecatedProfileCreate ----------

// func (c *ProfileController) DeprecatedProfileCreateHandler(ctx context.Context, input *dto.DeprecatedProfileCreateInput) (*dto.DeprecatedProfileCreateOutput, error) {
// 	userID, err := utils.GetUserIDFromHumaContext(ctx)
// 	if err != nil {
// 		return nil, huma.Error401Unauthorized(err.Error())
// 	}

// 	profileService := services.NewProfileService(c.cfg)

// 	var slug string
// 	switch input.Body.AccountType {
// 	case "individual":
// 		var req dto.CreateIndividualProfileRequest
// 		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
// 			return nil, huma.Error400BadRequest("invalid request body")
// 		}
// 		slug, err = profileService.CreateIndividualProfile(userID, req)
// 	case "agency":
// 		var req dto.CreateAgencyProfileRequest
// 		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
// 			return nil, huma.Error400BadRequest("invalid request body")
// 		}
// 		slug, err = profileService.CreateAgencyProfile(userID, req)
// 	case "client":
// 		var req dto.CreateClientProfileRequest
// 		if err := json.Unmarshal(input.Body.ProfileData, &req); err != nil {
// 			return nil, huma.Error400BadRequest("invalid request body")
// 		}
// 		slug, err = profileService.CreateClientProfile(userID, req)
// 	default:
// 		return nil, huma.Error400BadRequest("unknown account_type: must be individual, agency, or client")
// 	}

// 	if err != nil {
// 		if errors.Is(err, services.ErrUserNotFound) {
// 			return nil, huma.Error404NotFound(err.Error())
// 		}
// 		if errors.Is(err, services.ErrProfileAlreadyExists) {
// 			return nil, huma.Error409Conflict(err.Error())
// 		}
// 		if errors.Is(err, services.ErrForbidden) {
// 			return nil, huma.Error403Forbidden(err.Error())
// 		}
// 		if errors.Is(err, services.ErrValidation) {
// 			return nil, huma.Error400BadRequest(err.Error())
// 		}
// 		if errors.Is(err, services.ErrInternal) {
// 			return nil, huma.Error500InternalServerError(err.Error())
// 		}
// 		return nil, huma.Error500InternalServerError(err.Error())
// 	}

// 	return &dto.DeprecatedProfileCreateOutput{
// 		Body: dto.CreateProfileResponse{
// 			Message:       "Profile created successfully (deprecated: use POST /v1/profile/{type} instead)",
// 			PublicUrlSlug: slug,
// 		},
// 	}, nil
// }

// mapProfileError centralizes mapping of service-layer sentinel errors
// to HTTP status codes, avoiding duplication across handlers.
func mapProfileError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrUserNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, services.ErrProfileNotFound):
		return huma.Error404NotFound("profile not found, create it first")
	case errors.Is(err, services.ErrProfileAlreadyExists):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, services.ErrForbidden):
		return huma.Error403Forbidden(err.Error())
	case errors.Is(err, services.ErrValidation):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, services.ErrNothingToDelete):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, services.ErrAccountTypeNotSet):
		return huma.Error400BadRequest("account type not set")
	case errors.Is(err, services.ErrInvalidAccountType):
		return huma.Error400BadRequest("invalid account type")
	case errors.Is(err, services.ErrInvalidPassword):
		return huma.Error401Unauthorized("invalid password")
	case errors.Is(err, services.ErrSessionNotFound):
		return huma.Error404NotFound("session not found")
	default:
		return huma.Error500InternalServerError(err.Error())
	}
}
