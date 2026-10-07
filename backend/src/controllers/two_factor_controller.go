// controllers/two_factor_controller.go
package controllers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

// ---------- Setup ----------

func Setup2FAHandler(ctx context.Context, input *dto.Setup2FAInput) (*dto.Setup2FAOutput, error) {
	userIDStr, _ := ctx.Value(utils.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid user")
	}

	res, err := services.NewTwoFactorService().Setup(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.Setup2FAOutput{Body: *res}, nil
}

// ---------- VerifySetup ----------

func VerifySetup2FAHandler(ctx context.Context, input *dto.VerifySetup2FAInput) (*dto.VerifySetup2FAOutput, error) {
	userIDStr, _ := ctx.Value(utils.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid user")
	}

	res, err := services.NewTwoFactorService().VerifySetup(userID, input.Body.Code)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &dto.VerifySetup2FAOutput{Body: *res}, nil
}

// ---------- VerifyLogin ----------

func VerifyLogin2FAHandler(ctx context.Context, input *dto.VerifyLogin2FAInput) (*dto.VerifyLogin2FAOutput, error) {
	res, refreshToken, err := services.NewTwoFactorService().VerifyLogin(input.Body.TemporaryToken, input.Body.Code, input.UserAgent, utils.GetClientIP(input.ForwardedFor), input.UserAgent)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	return &dto.VerifyLogin2FAOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- VerifyRecoveryCode ----------

func VerifyRecoveryCodeHandler(ctx context.Context, input *dto.VerifyRecoveryCodeInput) (*dto.VerifyRecoveryCodeOutput, error) {
	res, refreshToken, err := services.NewTwoFactorService().VerifyRecoveryCode(
		input.Body.TemporaryToken,
		input.Body.Code,
		input.UserAgent,
		utils.GetClientIP(input.ForwardedFor),
		input.UserAgent,
	)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	return &dto.VerifyRecoveryCodeOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- Disable ----------

func Disable2FAHandler(ctx context.Context, input *dto.Disable2FAInput) (*dto.Disable2FAOutput, error) {
	userIDStr, _ := ctx.Value(utils.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid user")
	}

	if err := services.NewTwoFactorService().Disable(userID, input.Body.Password, input.Body.Code); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.Disable2FAOutput{
		Body: dto.Message2FAResponse{Message: "2FA disabled"},
	}, nil
}

// ---------- RegenerateRecoveryCodes ----------

func RegenerateRecoveryCodesHandler(ctx context.Context, input *dto.RegenerateRecoveryCodesInput) (*dto.RegenerateRecoveryCodesOutput, error) {
	userIDStr, _ := ctx.Value(utils.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid user")
	}

	codes, err := services.NewTwoFactorService().RegenerateRecoveryCodes(userID, input.Body.Password, input.Body.Code)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RegenerateRecoveryCodesOutput{
		Body: dto.RegenerateRecoveryCodesResponse{
			Message:       "Recovery codes regenerated",
			RecoveryCodes: codes,
		},
	}, nil
}
