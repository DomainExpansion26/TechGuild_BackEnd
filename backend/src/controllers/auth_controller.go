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

type RegisterInput struct {
	Body dto.RegisterRequest
}
type RegisterOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func RegisterHandler(ctx context.Context, input *RegisterInput) (*RegisterOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.Register(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	out := &RegisterOutput{}
	out.Body.Message = "Registration successful. Please check your email to verify your account."
	return out, nil
}

// ---------- Login ----------

type LoginInput struct {
	Body dto.LoginRequest
}
type LoginOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      dto.LoginResponse
}

func LoginHandler(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
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

	return &LoginOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- VerifyEmail ----------

type VerifyEmailInput struct {
	Token string `query:"token" required:"true" doc:"Email verification token"`
}
type VerifyEmailOutput struct {
	Body dto.VerifyEmailResponse
}

func VerifyEmailHandler(ctx context.Context, input *VerifyEmailInput) (*VerifyEmailOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	res, err := authService.VerifyEmail(dto.VerifyEmailRequest{Token: input.Token})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &VerifyEmailOutput{Body: *res}, nil
}

// ---------- ResendVerificationEmail ----------

type ResendVerificationInput struct {
	Body dto.ResendVerificationRequest
}
type ResendVerificationOutput struct {
	Body dto.ResendVerificationResponse
}

func ResendVerificationEmailHandler(ctx context.Context, input *ResendVerificationInput) (*ResendVerificationOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ResendVerificationEmail(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &ResendVerificationOutput{
		Body: dto.ResendVerificationResponse{Message: "Verification email sent successfully"},
	}, nil
}

// ---------- Logout ----------

type LogoutInput struct {
	RefreshTokenCookie string `cookie:"refresh_token"`
	Authorization      string `header:"Authorization"`
	Body               dto.LogoutRequest
}
type LogoutOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      dto.LogoutResponse
}

func LogoutHandler(ctx context.Context, input *LogoutInput) (*LogoutOutput, error) {
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

	return &LogoutOutput{
		SetCookie: clearCookie.String(),
		Body:      dto.LogoutResponse{Message: "Logout successful"},
	}, nil
}

// ---------- RefreshToken ----------

type RefreshTokenInput struct {
	RefreshTokenCookie string `cookie:"refresh_token"`
	Body               dto.RefreshTokenRequest
}
type RefreshTokenOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      dto.RefreshTokenResponse
}

func RefreshTokenHandler(ctx context.Context, input *RefreshTokenInput) (*RefreshTokenOutput, error) {
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

	return &RefreshTokenOutput{
		SetCookie: cookie.String(),
		Body:      *res,
	}, nil
}

// ---------- ResetPassword ----------

type ResetPasswordInput struct {
	Token string `query:"token" required:"true" doc:"Password reset token"`
	Body  dto.ResetPasswordRequest
}
type ResetPasswordOutput struct {
	Body dto.ResetPasswordResponse
}

func ResetPasswordHandler(ctx context.Context, input *ResetPasswordInput) (*ResetPasswordOutput, error) {
	input.Body.Token = input.Token

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ResetPassword(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &ResetPasswordOutput{
		Body: dto.ResetPasswordResponse{Message: "Password reset successfully"},
	}, nil
}

// ---------- ForgotPassword ----------

type ForgotPasswordInput struct {
	Body dto.ForgotPasswordRequest
}
type ForgotPasswordOutput struct {
	Body dto.ForgotPasswordResponse
}

func ForgotPasswordHandler(ctx context.Context, input *ForgotPasswordInput) (*ForgotPasswordOutput, error) {
	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ForgotPassword(input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &ForgotPasswordOutput{
		Body: dto.ForgotPasswordResponse{Message: "Password reset link sent successfully"},
	}, nil
}

// ---------- ChangePassword (protected) ----------

type ChangePasswordInput struct {
	Body dto.ChangePasswordRequest
}
type ChangePasswordOutput struct {
	Body dto.ChangePasswordResponse
}

func ChangePasswordHandler(ctx context.Context, input *ChangePasswordInput) (*ChangePasswordOutput, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.ChangePassword(userID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &ChangePasswordOutput{
		Body: dto.ChangePasswordResponse{Message: "Password changed successfully"},
	}, nil
}

// ---------- DeleteAccount (protected) ----------

type DeleteAccountInput struct{}
type DeleteAccountOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func DeleteAccountHandler(ctx context.Context, input *DeleteAccountInput) (*DeleteAccountOutput, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	authService := services.NewAuthService(postgres.RedisDB)

	if err := authService.DeleteAccount(userID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	out := &DeleteAccountOutput{}
	out.Body.Message = "Account deleted successfully"
	return out, nil
}
