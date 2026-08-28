package controllers

import (
	"context"
	"net/http"
	"strings"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- Register ----------

func RegisterHandler(ctx context.Context, input *dto.RegisterInput) (*dto.RegisterOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.Register(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	out := &dto.RegisterOutput{}
	out.Body.Message = "Registration successful. Please check your email to verify your account."
	return out, nil
}

// ---------- Login ----------

func LoginHandler(ctx context.Context, input *dto.LoginInput) (*dto.LoginOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	res, refreshToken, err := authService.Login(input.Body)
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
		SameSite: http.SameSiteStrictMode,
	}

	return &dto.LoginOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- VerifyEmail ----------

func VerifyEmailHandler(ctx context.Context, input *dto.VerifyEmailInput) (*dto.VerifyEmailOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	res, err := authService.VerifyEmail(dto.VerifyEmailRequest{Token: input.Token})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.VerifyEmailOutput{Body: *res}, nil
}

// ---------- ResendVerificationEmail ----------

func ResendVerificationEmailHandler(ctx context.Context, input *dto.ResendVerificationInput) (*dto.ResendVerificationOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ResendVerificationEmail(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ResendVerificationOutput{
		Body: dto.ResendVerificationResponse{Message: "Verification email sent successfully"},
	}, nil
}

// ---------- Logout ----------

func LogoutHandler(ctx context.Context, input *dto.LogoutInput) (*dto.LogoutOutput, error) {
	token := input.RefreshTokenCookie
	if token == "" {
		token = input.Body.RefreshToken
	}

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.Logout(token); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	if accessToken := strings.TrimPrefix(input.Authorization, "Bearer "); accessToken != "" {
		_ = authService.BlacklistAccessToken(accessToken)
	}

	clearCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	return &dto.LogoutOutput{
		SetCookie: clearCookie.String(),
		Body:      dto.LogoutResponse{Message: "Logout successful"},
	}, nil
}

// ---------- RefreshToken ----------

func RefreshTokenHandler(ctx context.Context, input *dto.RefreshTokenInput) (*dto.RefreshTokenOutput, error) {
	oldToken := input.RefreshTokenCookie
	if oldToken == "" {
		oldToken = input.Body.RefreshToken
	}

	if oldToken == "" {
		return nil, huma.Error401Unauthorized("refresh token missing")
	}

	authService := services.NewAuthService(postgres.RedisDB)

	res, newRefreshToken, err := authService.RefreshToken(oldToken)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	return &dto.RefreshTokenOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- ResetPassword ----------

func ResetPasswordHandler(ctx context.Context, input *dto.ResetPasswordInput) (*dto.ResetPasswordOutput, error) {
	input.Body.Token = input.Token

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ResetPassword(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ResetPasswordOutput{
		Body: dto.ResetPasswordResponse{Message: "Password reset successfully"},
	}, nil
}

// ---------- ForgotPassword ----------

func ForgotPasswordHandler(ctx context.Context, input *dto.ForgotPasswordInput) (*dto.ForgotPasswordOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ForgotPassword(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ForgotPasswordOutput{
		Body: dto.ForgotPasswordResponse{Message: "Password reset link sent successfully"},
	}, nil
}

// ---------- ChangePassword (protected) ----------

func ChangePasswordHandler(ctx context.Context, input *dto.ChangePasswordInput) (*dto.ChangePasswordOutput, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ChangePassword(userID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ChangePasswordOutput{
		Body: dto.ChangePasswordResponse{Message: "Password changed successfully"},
	}, nil
}

// ---------- DeleteAccount (protected) ----------

func DeleteAccountHandler(ctx context.Context, input *dto.DeleteAccountInput) (*dto.DeleteAccountOutput, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.DeleteAccount(userID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	out := &dto.DeleteAccountOutput{}
	out.Body.Message = "Account deleted successfully"
	return out, nil
}
