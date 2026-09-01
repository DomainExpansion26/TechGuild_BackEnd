package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

func UpdateAccountSettingsHandler(ctx context.Context, input *dto.UpdateAccountSettingsInput) (*dto.UpdateAccountSettingsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.UpdateAccountSettings(userID, input.Body); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdateAccountSettingsOutput{Body: dto.SettingsUpdateResponse{Message: "account settings updated successfully"}}, nil
}

func UpdateNotificationsHandler(ctx context.Context, input *dto.UpdateNotificationsInput) (*dto.UpdateNotificationsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.UpdateNotifications(userID, input.Body); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdateNotificationsOutput{Body: dto.SettingsUpdateResponse{Message: "notifications updated successfully"}}, nil
}

func UpdatePrivacySettingsHandler(ctx context.Context, input *dto.UpdatePrivacyInput) (*dto.UpdatePrivacyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService()
	if err := profileService.UpdatePrivacySettings(userID, input.Body); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdatePrivacyOutput{Body: dto.SettingsUpdateResponse{Message: "privacy settings updated successfully"}}, nil
}
