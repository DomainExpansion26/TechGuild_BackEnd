package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		profileData := c.PostForm("profile_data")
		if profileData != "" {
			if err := json.Unmarshal([]byte(profileData), &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile_data JSON: " + err.Error()})
				return
			}
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

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
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
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

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		profileData := c.PostForm("profile_data")
		if profileData != "" {
			if err := json.Unmarshal([]byte(profileData), &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile_data JSON: " + err.Error()})
				return
			}
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		// Handle Logo upload
		logoFileHeader, err := c.FormFile("logo")
		if err == nil && logoFileHeader != nil {
			logoFile, _ := logoFileHeader.Open()
			defer logoFile.Close()
			logoURL, uploadErr := utils.UploadImageToCloudinary(logoFile, logoFileHeader.Filename)
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload logo: " + uploadErr.Error()})
				return
			}
			req.LogoURL = logoURL
		}
	} else {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		profileData := c.PostForm("profile_data")
		if profileData != "" {
			if err := json.Unmarshal([]byte(profileData), &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile_data JSON: " + err.Error()})
				return
			}
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		// Handle Logo upload
		logoFileHeader, err := c.FormFile("logo")
		if err == nil && logoFileHeader != nil {
			logoFile, _ := logoFileHeader.Open()
			defer logoFile.Close()
			logoURL, uploadErr := utils.UploadImageToCloudinary(logoFile, logoFileHeader.Filename)
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload logo: " + uploadErr.Error()})
				return
			}
			req.LogoURL = logoURL
		}
	} else {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err := profileService.SetAccountType(req)
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

func UploadLogo(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "logo image is required"})
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

	filename := fmt.Sprintf("%s-logo-%d", userID, time.Now().Unix())

	fileURL, err := utils.UploadImageToCloudinary(file, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload to cloudinary: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "logo uploaded successfully",
		"logo_url": fileURL,
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

func DeleteLogo(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.DeleteLogo(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logo deleted successfully"})
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

func CreateOrUpdateProfile(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	accountType, err := profileService.GetUserAccountType(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch accountType {
	case "individual":
		CreateOrUpdateIndividualProfile(c)
	case "agency":
		CreateOrUpdateAgencyProfile(c)
	case "client":
		CreateOrUpdateClientProfile(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account type"})
	}
}

func CheckSlug(c *gin.Context) {
	slug := c.Query("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug query parameter is required"})
		return
	}

	profileService := services.NewProfileService()
	resp, err := profileService.CheckSlugAvailability(slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func DeleteProfileAccount(c *gin.Context) {
	var req dto.DeleteAccountRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profileService := services.NewProfileService()
	err = profileService.DeleteAccount(userID, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account successfully scheduled for deletion"})
}
