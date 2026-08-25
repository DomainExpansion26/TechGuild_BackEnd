package controllers

import (
	"net/http"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	_ "techguild-backend/src/swagger"

	"github.com/gin-gonic/gin"
)

// UpdateAccountSettings godoc
// @Summary Update account settings
// @Description Updates the authenticated user's account settings.
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.UpdateAccountRequest true "Account settings"
// @Success 200 {object} swagger.UpdateAccountResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Failure 500 {object} swagger.ErrorResponse
// @Router /settings/account [put]
func UpdateAccountSettings(c *gin.Context) {
	var req dto.UpdateAccountRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.UpdateAccountSettings(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account settings updated successfully"})
}

// UpdateNotifications godoc
// @Summary Update notification settings
// @Description Updates the authenticated user's notification preferences.
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.UpdateNotificationsRequest true "Notification settings"
// @Success 200 {object} swagger.UpdateNotificationsResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Failure 500 {object} swagger.ErrorResponse
// @Router /settings/notifications [put]
func UpdateNotifications(c *gin.Context) {
	var req dto.UpdateNotificationsRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.UpdateNotifications(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notifications updated successfully"})
}

// UpdatePrivacySettings godoc
// @Summary Update privacy settings
// @Description Updates the authenticated user's privacy preferences.
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.UpdatePrivacyRequest true "Privacy settings"
// @Success 200 {object} swagger.UpdatePrivacySettingsResponse
// @Failure 400 {object} swagger.ErrorResponse "Invalid request"
// @Failure 401 {object} swagger.ErrorResponse "Unauthorized"
// @Failure 500 {object} swagger.ErrorResponse "Internal server error"
// @Router /settings/privacy [put]
func UpdatePrivacySettings(c *gin.Context) {
	var req dto.UpdatePrivacyRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.UpdatePrivacySettings(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "privacy settings updated successfully"})
}
