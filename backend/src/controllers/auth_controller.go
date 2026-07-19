package controllers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	_"techguild-backend/src/swagger"
)
// Register godoc
//
//	@Summary		Register a new user
//	@Description	Register a new TechGuild user account and send an email verification link.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Registration Request"
//	@Success		201		{object}	swagger.RegisterSuccessResponse
//	@Failure		400		{object}	swagger.RegisterErrorResponse
//	@Router			/auth/register [post]
func Register(c *gin.Context) {

	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

// Login godoc
//
//	@Summary		User Login
//	@Description	Authenticate a user using email and password and return JWT access and refresh tokens.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Login Request"
//	@Success		200		{object}	swagger.LoginSuccessResponse
//	@Failure		400		{object}	swagger.LoginBadRequestResponse
//	@Failure		401		{object}	swagger.LoginUnauthorizedResponse
//	@Router			/auth/login [post]
func Login(c *gin.Context) {

	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	res, err := authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// VerifyEmail godoc
// @Summary Verify email
// @Description Verifies a user's email address using the verification token sent via email.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param token query string true "Email verification token"
// @Success 200 {object} dto.VerifyEmailResponse
// @Failure 400 {object} map[string]string "Invalid or expired token"
// @Router /auth/verify-email [get]
func VerifyEmail(c *gin.Context) {

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "verification token is required",
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	res, err := authService.VerifyEmail(dto.VerifyEmailRequest{
		Token: token,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ResendVerificationEmail godoc
// @Summary Resend verification email
// @Description Sends a new email verification link to an unverified user.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationRequest true "Resend verification request"
// @Success 200 {object} dto.ResendVerificationResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Router /auth/resend-verification [post]
func ResendVerificationEmail(c *gin.Context) {

	var req dto.ResendVerificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.ResendVerificationEmail(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ResendVerificationResponse{
		Message: "Verification email sent successfully",
	})
}

// Logout godoc
// @Summary Logout user
// @Description Invalidates the refresh token and logs the user out.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.LogoutRequest true "Logout request"
// @Success 200 {object} dto.LogoutResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/logout [post]
func Logout(c *gin.Context) {

	var req dto.LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.Logout(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.LogoutResponse{
		Message: "Logout successful",
	})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generates a new access token using a valid refresh token.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} dto.RefreshTokenResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /auth/refresh-token [post]
func RefreshToken(c *gin.Context) {

	var req dto.RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	res, err := authService.RefreshToken(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ResetPassword godoc
// @Summary Reset password
// @Description Resets the user's password using the password reset token.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param token query string true "Password reset token"
// @Param request body dto.ResetPasswordRequest true "Reset password request"
// @Success 200 {object} dto.ResetPasswordResponse
// @Failure 400 {object} map[string]string "Invalid or expired token"
// @Router /auth/reset-password [post]
func ResetPassword(c *gin.Context) {

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "reset token is required",
		})
		return
	}

	var req dto.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	req.Token = token

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.ResetPassword(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ResetPasswordResponse{
		Message: "Password reset successfully",
	})
}

// ForgotPassword godoc
// @Summary Forgot password
// @Description Sends a password reset link to the user's registered email address.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Forgot password request"
// @Success 200 {object} dto.ForgotPasswordResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Router /auth/forgot-password [post]
func ForgotPassword(c *gin.Context) {

	var req dto.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.ForgotPassword(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ForgotPasswordResponse{
		Message: "Password reset link sent successfully",
	})
}

// ChangePassword godoc
// @Summary Change password
// @Description Changes the password of the authenticated user.
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Change password request"
// @Success 200 {object} dto.ChangePasswordResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /auth/change-password [post]
func ChangePassword(c *gin.Context) {

	userID := c.GetString("user_id")

	var req dto.ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.ChangePassword(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ChangePasswordResponse{
		Message: "Password changed successfully",
	})
}

// DeleteAccount godoc
// @Summary Delete account
// @Description Permanently deletes the authenticated user's account.
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /auth/delete-account [delete]
func DeleteAccount(c *gin.Context) {
	userID := c.GetString("user_id")

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.DeleteAccount(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deleted successfully",
	})
}