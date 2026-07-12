package controllers

import (
	"errors"
	"net/http"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"

	"github.com/gin-gonic/gin"
)

func CreateProfile(c *gin.Context) {
	var req dto.CreateProfileRequest

	// parse the req data and validate
	err := c.ShouldBindJSON(&req)
	if err != nil {
		// write json to client, need status code and error(gin helps us to map the key and value)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		// err.Error() -> converts the object to plain text
		return
	}

	// fetch the user id from contex
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user is not authenticated",
		})
		return
	}
	// cast to string because the userIDVal is type of Any
	userID, ok := userIDVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid user id format",
		})

		return
	}

	// create the profile service
	profileService := services.NewProfileService()

	// call te service to createOrUpdate the profile

	slug, err := profileService.CreateOrUpdateProfile(userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// return the response

	c.JSON(http.StatusOK, dto.CreateProfileResponse{
		Message:       "Profile update successfuly",
		PublicUrlSlug: slug,
	})
}
