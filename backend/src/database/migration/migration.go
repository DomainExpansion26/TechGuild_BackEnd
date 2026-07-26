package migration

import (
	"log"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func Migrate() {

	err := postgres.DB.AutoMigrate(

		// =========================
		// Users
		// =========================
		&models.User{},
		&models.IndividualProfile{},
		&models.AgencyProfile{},
		&models.ClientProfile{},
		&models.UserSession{},

		// =========================
		// Verification
		// =========================
		&models.VerificationRecord{},
		&models.VerificationDocument{},
		&models.GovernmentID{},
		&models.GovtIDDedup{},
		&models.BusinessPANDedup{},

		// =========================
		// Rules
		// =========================
		&models.RuleDocument{},
		&models.PolicyChangeNotification{},

		// =========================
		// Audit
		// =========================
		&models.AuditLog{},

		// =========================
		// Projects
		// =========================
		&models.Project{},
		&models.ProjectSkill{},
		&models.ProjectAttachment{},
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	log.Println("Database Migrated Successfully")
}