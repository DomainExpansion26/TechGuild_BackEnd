package jobs

import (
	"log"
	"strings"
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"
)

func extractCloudinaryPublicID(url string) string {
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "/upload/")
	if len(parts) != 2 {
		return ""
	}
	subParts := strings.SplitN(parts[1], "/", 2)
	if len(subParts) == 2 {
		fileName := subParts[1]
		if idx := strings.LastIndex(fileName, "."); idx != -1 {
			return fileName[:idx]
		}
		return fileName
	}
	return ""
}

func StartCleanupJob() {
	log.Println("Starting background cleanup job worker...")
	// Run immediately on startup
	runCleanup()

	// Then run periodically
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			<-ticker.C
			runCleanup()
		}
	}()
}

func runCleanup() {
	log.Println("[Cleanup Job] Running daily cleanup for pending_deletion accounts...")

	var users []models.User
	// Find users where Status == pending_deletion and ScheduledDeletionDate < Now
	err := postgres.DB.Where("status = ? AND scheduled_deletion_date < ?", models.StatusPendingDeletion, time.Now()).Find(&users).Error
	if err != nil {
		log.Println("[Cleanup Job] Error querying users:", err)
		return
	}

	for _, user := range users {
		log.Printf("[Cleanup Job] Hard deleting account: %s", user.ID)

		// 1. Delete associated profile and Cloudinary files
		if user.AccountType != nil {
			switch *user.AccountType {
			case models.AccountTypeIndividual:
				var profile models.IndividualProfile
				if err := postgres.DB.Where("user_id = ?", user.ID).First(&profile).Error; err == nil {
					// Delete Avatar
					if publicID := extractCloudinaryPublicID(profile.AvatarURL); publicID != "" {
						_ = utils.DeleteFromCloudinary(publicID, "image")
					}
					// Delete Resume
					if publicID := extractCloudinaryPublicID(profile.ResumeURL); publicID != "" {
						_ = utils.DeleteFromCloudinary(publicID, "image")
					}
					// Hard delete profile
					postgres.DB.Unscoped().Delete(&profile)
				}
			case models.AccountTypeAgencyAdmin:
				var profile models.AgencyProfile
				if err := postgres.DB.Where("user_id = ?", user.ID).First(&profile).Error; err == nil {
					// Delete Logo
					if publicID := extractCloudinaryPublicID(profile.LogoURL); publicID != "" {
						_ = utils.DeleteFromCloudinary(publicID, "image")
					}
					// Hard delete profile
					postgres.DB.Unscoped().Delete(&profile)
				}
			case models.AccountTypeClientAdmin:
				var profile models.ClientProfile
				if err := postgres.DB.Where("user_id = ?", user.ID).First(&profile).Error; err == nil {
					// Delete Logo
					if publicID := extractCloudinaryPublicID(profile.LogoURL); publicID != "" {
						_ = utils.DeleteFromCloudinary(publicID, "image")
					}
					// Hard delete profile
					postgres.DB.Unscoped().Delete(&profile)
				}
			}
		}

		// 2. Anonymize user record
		user.FirstName = "Deleted"
		user.LastName = "User"
		user.Email = "deleted_" + user.ID.String() + "@deleted.local"
		user.PasswordHash = ""
		user.Status = models.StatusDeleted
		user.ScheduledDeletionDate = nil

		postgres.DB.Save(&user)
		log.Printf("[Cleanup Job] Successfully wiped and anonymized account: %s", user.ID)
	}
}
