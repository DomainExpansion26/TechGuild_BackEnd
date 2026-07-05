package migration

import (
	"log"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func Migrate() {

	err := postgres.DB.AutoMigrate(

		&models.User{},
		&models.UserProfile{},
		&models.UserSession{},
		&models.VerificationRecord{},

		&models.GovernmentID{},

		&models.RuleDocument{},
		&models.PolicyChangeNotification{},

		&models.AuditLog{},
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	log.Println("Database Migrated Successfully")
}