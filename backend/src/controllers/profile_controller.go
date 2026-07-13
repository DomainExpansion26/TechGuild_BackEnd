package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
	"time"

	"github.com/gin-gonic/gin"
)

func getUserIDFromContext(c *gin.Context) (string, error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return "", errors.New("user is not authenticated")
	}
	userID, ok := userIDVal.(string)
	if !ok {
		return "", errors.New("invalid user id format")
	}
	return userID, nil
}

func CreateOrUpdateIndividualProfile(c *gin.Context) {
	var req dto.CreateIndividualProfileRequest

	// Support both JSON (if no files) and multipart form (if uploading files)
	contentType := c.GetHeader("Content-Type")
	
	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Multipart form data
		profileData := c.PostForm("profile_data")
		if profileData != "" {
			if err := json.Unmarshal([]byte(profileData), &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile_data JSON: " + err.Error()})
				return
			}
		}

		// Handle Avatar upload
		avatarFileHeader, err := c.FormFile("avatar")
		if err == nil && avatarFileHeader != nil {
			avatarFile, _ := avatarFileHeader.Open()
			defer avatarFile.Close()
			avatarURL, uploadErr := utils.UploadImageToCloudinary(avatarFile, avatarFileHeader.Filename)
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload avatar: " + uploadErr.Error()})
				return
			}
			req.AvatarURL = avatarURL
		}

		// Handle Resume upload
		resumeFileHeader, err := c.FormFile("resume")
		if err == nil && resumeFileHeader != nil {
			resumeFile, _ := resumeFileHeader.Open()
			defer resumeFile.Close()
			resumeURL, uploadErr := utils.UploadPDFToCloudinary(resumeFile, resumeFileHeader.Filename)
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload resume: " + uploadErr.Error()})
				return
			}
			req.ResumeURL = resumeURL
		}
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateIndividualProfile(userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreateProfileResponse{
		Message:       "Individual profile updated successfully",
		PublicUrlSlug: slug,
	})
}

func CreateOrUpdateAgencyProfile(c *gin.Context) {
	var req dto.CreateAgencyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateAgencyProfile(userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreateProfileResponse{
		Message:       "Agency profile updated successfully",
		PublicUrlSlug: slug,
	})
}

func CreateOrUpdateClientProfile(c *gin.Context) {
	var req dto.CreateClientProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	slug, err := profileService.CreateOrUpdateClientProfile(userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreateProfileResponse{
		Message:       "Client profile updated successfully",
		PublicUrlSlug: slug,
	})
}

func GetMyProfile(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	profile, err := profileService.GetMyProfile(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func SetAccountType(c *gin.Context) {
	var req dto.SetAccountTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	profileService := services.NewProfileService()
	err := profileService.SetAccountType(userID.(string), req.AccountType)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "account type set successfully"})
}

func UploadResume(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("resume")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resume file is required"})
		return
	}

	if filepath.Ext(fileHeader.Filename) != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only PDF files are allowed"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadPDFToCloudinary(file, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload to cloudinary: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "resume uploaded successfully",
		"resume_url": fileURL,
	})
}

func UploadAvatar(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar image is required"})
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only JPG, PNG, and WebP images are allowed"})
		return
	}

	const maxFileSize = 5 * 1024 * 1024 // 5MB
	if fileHeader.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 5MB limit"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%s-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadImageToCloudinary(file, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload to cloudinary: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "avatar uploaded successfully",
		"avatar_url": fileURL,
	})
}

func DeleteAvatar(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.DeleteAvatar(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "avatar deleted successfully"})
}

func DeleteResume(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.DeleteResume(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "resume deleted successfully"})
}

func GetPublicProfile(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
		return
	}

	profileService := services.NewProfileService()
	profile, err := profileService.GetPublicProfile(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func GetUserPoints(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	points, err := profileService.GetUserPoints(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, points)
}

func ExportProfile(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	result, err := profileService.ExportUserData(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

