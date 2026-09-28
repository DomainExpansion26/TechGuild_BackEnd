package migration

import (
	"log"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func Migrate() {

	err := postgres.DB.AutoMigrate(

		// Users & Profiles
		&models.User{},
		&models.IndividualProfile{},
		&models.AgencyProfile{},
		&models.ClientProfile{},
		&models.UserSession{},

		// Verification
		&models.VerificationRecord{},
		&models.VerificationDocument{},
		&models.GovernmentID{},
		&models.GovtIDDedup{},
		&models.BusinessPANDedup{},

		// Rules & Notifications
		&models.RuleDocument{},
		&models.PolicyChangeNotification{},

		// Audit
		&models.AuditLog{},

		// Projects
		&models.Project{},
		&models.ProjectSkill{},
		&models.ProjectAttachment{},
		&models.ProjectApplication{},
		&models.ProjectContract{},
		&models.ProjectMilestone{},
		&models.ProjectSubmission{},

<<<<<<< Updated upstream
		// Team Collaboration
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.TeamPortfolio{},
		&models.TeamSkill{},
=======
		// Party Collaboration
		&models.Party{},
		&models.PartyMember{},
		&models.PartyInvitation{},
		&models.PartyPortfolio{},
		&models.PartySkill{},

		// 2FA Authentication
		&models.UserTwoFactorAuthentication{},
		&models.UserRecoveryCode{},
>>>>>>> Stashed changes
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	log.Println("Database Migrated Successfully")
}