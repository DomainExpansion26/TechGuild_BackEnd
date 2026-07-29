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

		// Team Collaboration
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.TeamPortfolio{},
		&models.TeamSkill{},
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	log.Println("Database Migrated Successfully")
}