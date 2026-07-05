package controllers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

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
		"message": "Registration successful. OTP sent to your email.",
	})
}
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
func VerifyEmail(c *gin.Context) {

	var req dto.VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.VerifyEmail(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

func ResendOTP(c *gin.Context) {

	var req dto.ResendOTPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService(postgres.RedisDB)

	err := authService.ResendOTP(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
}
func RefreshToken(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error": "invalid request",
		})
		return
	}
	claims, err := utils.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(401, gin.H{
			"error": "invalid or expired refresh token",
		})
		return
	}
	newAccessToken, err := utils.GenerateAccessToken(claims["user_id"].(string))
	if err != nil {
		c.JSON(500, gin.H{
			"error": "failed to generate token",
		})
		return
	}
	c.JSON(200, dto.RefreshResponse{
		AccessToken: newAccessToken,
	})
}