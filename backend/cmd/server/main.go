package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/jobs"
	"techguild-backend/src/routes"
)

func main() {
	postgres.ConnectDatabase()
	migration.Migrate()

	jobs.StartCleanupJob()

	router := gin.Default()
	router.Static("/uploads", "./uploads")
	routes.AuthRoutes(router)
	routes.OAuthRoutes(router)
	routes.ProfileRoutes(router)
	routes.VerificationRoutes(router)
	log.Println("Server running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
