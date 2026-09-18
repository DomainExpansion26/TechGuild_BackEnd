package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

type SettingsController struct {
	cfg *config.Config
}

func NewSettingsController(cfg *config.Config) *SettingsController {
	return &SettingsController{cfg: cfg}
}

func (c *SettingsController) GetAccountSettingsHandler(ctx context.Context, input *dto.GetAccountSettingsInput) (*dto.GetAccountSettingsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	accountSettings, err := profileService.GetAccountSettings(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetAccountSettingsOutput{
		Body: *accountSettings,
	}, nil
}

func (c *SettingsController) UpdateAccountSettingsHandler(ctx context.Context, input *dto.UpdateAccountSettingsInput) (*dto.UpdateAccountSettingsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
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

	return &dto.UpdateAccountSettingsOutput{
		Body: dto.SettingsUpdateResponse{
			Message: "account settings updated successfully",
		},
	}, nil
}

func (c *SettingsController) GetNotificationsHandler(ctx context.Context, input *dto.GetNotificationsInput) (*dto.GetNotificationsOutput, error) {
	return nil, huma.Error501NotImplemented("Notifications settings are not implemented yet")
}

func (c *SettingsController) UpdateNotificationsHandler(ctx context.Context, input *dto.UpdateNotificationsInput) (*dto.UpdateNotificationsOutput, error) {
	return nil, huma.Error501NotImplemented("Notifications settings are not implemented yet")
}

func (c *SettingsController) GetPrivacySettingsHandler(ctx context.Context, input *dto.GetPrivacySettingsInput) (*dto.GetPrivacySettingsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	profileService := services.NewProfileService(c.cfg)
	privacySettings, err := profileService.GetPrivacySettings(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrAccountTypeNotSet) || errors.Is(err, services.ErrInvalidAccountType) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return &dto.GetPrivacySettingsOutput{Body: *privacySettings}, nil
}

func (c *SettingsController) UpdatePrivacySettingsHandler(ctx context.Context, input *dto.UpdatePrivacyInput) (*dto.UpdatePrivacyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.UpdatePrivacySettings(userID, input.Body); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrProfileNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrAccountTypeNotSet) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrInvalidAccountType) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UpdatePrivacyOutput{Body: dto.SettingsUpdateResponse{Message: "privacy settings updated successfully"}}, nil
}

func (c *SettingsController) GetBillingSettingsHandler(ctx context.Context, input *dto.GetBillingSettingsInput) (*dto.GetBillingSettingsOutput, error) {
	return nil, huma.Error501NotImplemented("billing settings are not implemented yet")
}

func (c *SettingsController) UpdatePayoutMethodHandler(ctx context.Context, input *dto.UpdatePayoutMethodInput) (*dto.UpdatePayoutMethodOutput, error) {
	return nil, huma.Error501NotImplemented("payout method is not implemented yet")
}

func (c *SettingsController) UpdatePayoutScheduleHandler(ctx context.Context, input *dto.UpdatePayoutScheduleInput) (*dto.UpdatePayoutScheduleOutput, error) {
	return nil, huma.Error501NotImplemented("payout schedule is not implemented yet")
}

func (c *SettingsController) DeactivateAccountHandler(ctx context.Context, input *dto.DeactivateAccountInput) (*dto.DeactivateAccountOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.DeactivateAccount(userID, input.Body); err != nil {
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
	return &dto.DeactivateAccountOutput{
		Body: dto.SettingsUpdateResponse{
			Message: "account deactivated. log in within 30 days to reactivate, or it will be permanently deleted",
		},
	}, nil
}

func (c *SettingsController) DeleteAccountHandler(ctx context.Context, input *dto.DeleteAccountInput) (*dto.DeleteAccountOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.DeleteAccountPermanently(userID, input.Body.Password); err != nil {
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

	return &dto.DeleteAccountOutput{
		Body: dto.SettingsUpdateResponse{
			Message: "account permanently deleted",
		},
	}, nil
}

func (c *SettingsController) GetSessionsHandler(ctx context.Context, input *dto.GetSessionsInput) (*dto.GetSessionsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)

	sessions, err := profileService.GetSessions(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetSessionsOutput{
		Body: *sessions,
	}, nil
}

func (c *SettingsController) RevokeSessionHandler(ctx context.Context, input *dto.RevokeSessionInput) (*dto.RevokeSessionOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.RevokeSession(userID, input.ID); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrSessionNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrValidation) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return &dto.RevokeSessionOutput{
		Body: dto.SettingsUpdateResponse{
			Message: "session revoked successfully",
		},
	}, nil
}

func (c *SettingsController) SignOutOtherSessionsHandler(ctx context.Context, input *dto.SignOtherSessionsInput) (*dto.SignOtherSessionsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	currentSessionID, err := utils.GetSessionIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	profileService := services.NewProfileService(c.cfg)
	if err := profileService.SignOutOtherSessions(userID, currentSessionID); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrSessionNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return &dto.SignOtherSessionsOutput{
		Body: dto.SettingsUpdateResponse{
			Message: "signed out of all other sessions successfully",
		},
	}, nil
}
