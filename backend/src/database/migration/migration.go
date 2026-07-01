package migration

import (
	"log"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

func Migrate() error {

	err := postgres.DB.AutoMigrate(
		&models.User{},
	)

	if err != nil {
		return err
	}

	log.Println("✅ Database Migration Completed")

	return nil
}