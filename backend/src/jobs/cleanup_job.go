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

	// Then run periodically every hour
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for {
			<-ticker.C
			runCleanup()
		}
	}()
}

func runCleanup() {
	log.Println("[Cleanup Job] Running cleanup jobs...")

	// ============================================================
	// 1. Cleanup accounts scheduled for deletion (anonymize)
	// ============================================================

	var users []models.User

	err := postgres.DB.
		Where(
			"status IN ? AND scheduled_deletion_date < ?",
			[]models.UserStatus{models.StatusDeactivated, models.StatusPendingDeletion},
			time.Now(),
		).
		Find(&users).Error

	if err != nil {
		log.Println(
			"[Cleanup Job] Error querying pending deletion users:",
			err,
		)
	} else {
		for _, user := range users {
			log.Printf(
				"[Cleanup Job] Anonymizing account: %s",
				user.ID,
			)

			// Delete associated profile and Cloudinary files
			if user.AccountType != nil {
				switch *user.AccountType {

				case models.AccountTypeIndividual:
					var profile models.IndividualProfile

					if err := postgres.DB.
						Where("user_id = ?", user.ID).
						First(&profile).Error; err == nil {

						// Delete Avatar
						if publicID := extractCloudinaryPublicID(profile.AvatarURL); publicID != "" {
							_ = utils.DeleteFromCloudinary(
								publicID,
								"image",
							)
						}

						// Delete Resume
						if publicID := extractCloudinaryPublicID(profile.ResumeURL); publicID != "" {
							_ = utils.DeleteFromCloudinary(
								publicID,
								"image",
							)
						}

						// Hard delete profile
						postgres.DB.
							Unscoped().
							Delete(&profile)
					}

				case models.AccountTypeAgencyAdmin:
					var profile models.AgencyProfile

					if err := postgres.DB.
						Where("user_id = ?", user.ID).
						First(&profile).Error; err == nil {

						// Delete Logo
						if publicID := extractCloudinaryPublicID(profile.LogoURL); publicID != "" {
							_ = utils.DeleteFromCloudinary(
								publicID,
								"image",
							)
						}

						// Hard delete profile
						postgres.DB.
							Unscoped().
							Delete(&profile)
					}

				case models.AccountTypeClientAdmin:
					var profile models.ClientProfile

					if err := postgres.DB.
						Where("user_id = ?", user.ID).
						First(&profile).Error; err == nil {

						// Delete Logo
						if publicID := extractCloudinaryPublicID(profile.LogoURL); publicID != "" {
							_ = utils.DeleteFromCloudinary(
								publicID,
								"image",
							)
						}

						// Hard delete profile
						postgres.DB.
							Unscoped().
							Delete(&profile)
					}
				}
			}

			// ---------- Security data cleanup (new) ----------
			// Anonymized account should not retain a usable TOTP secret,
			// unused recovery codes, or active sessions.
			postgres.DB.Where("user_id = ?", user.ID).Delete(&models.UserTwoFactorAuthentication{})
			postgres.DB.Where("user_id = ?", user.ID).Delete(&models.UserRecoveryCode{})
			postgres.DB.Model(&models.UserSession{}).
				Where("user_id = ?", user.ID).
				Update("is_revoked", true)
			// ---------- End security data cleanup ----------

			// Anonymize user record
			user.FirstName = "Deleted"
			user.LastName = "User"
			user.Email = "deleted_" + user.ID.String() + "@deleted.local"
			user.PasswordHash = ""
			user.TwoFactorEnabled = false
			user.Status = models.StatusDeleted
			user.ScheduledDeletionDate = nil

			if err := postgres.DB.Save(&user).Error; err != nil {
				log.Printf(
					"[Cleanup Job] Failed to anonymize account %s: %v",
					user.ID,
					err,
				)
				continue
			}

			log.Printf(
				"[Cleanup Job] Successfully wiped and anonymized account: %s",
				user.ID,
			)
		}
	}

	// ============================================================
	// 2. Cleanup unverified accounts older than 48 hours (hard delete)
	// ============================================================
	// Note: UserTwoFactorAuthentication and UserRecoveryCode have
	// OnDelete:CASCADE foreign keys, so hard-deleting the user
	// automatically removes any related 2FA rows. UserSession's
	// cascade behavior should be verified separately if not already
	// confirmed.

	cutoff := time.Now().Add(-48 * time.Hour)

	var unverifiedUsers []models.User

	err = postgres.DB.
		Where(
			"email_verified = ? AND created_at < ?",
			false,
			cutoff,
		).
		Find(&unverifiedUsers).Error

	if err != nil {
		log.Println(
			"[Cleanup Job] Error querying unverified accounts:",
			err,
		)
		return
	}

	for _, user := range unverifiedUsers {
		log.Printf(
			"[Cleanup Job] Deleting unverified account: %s",
			user.ID,
		)

		// Hard delete unverified account
		if err := postgres.DB.
			Unscoped().
			Delete(&user).Error; err != nil {

			log.Printf(
				"[Cleanup Job] Failed to delete unverified account %s: %v",
				user.ID,
				err,
			)

			continue
		}

		log.Printf(
			"[Cleanup Job] Successfully deleted unverified account: %s",
			user.ID,
		)
	}
}
