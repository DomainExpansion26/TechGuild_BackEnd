package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/routes"
)

func main() {

	// Connect Database & Redis
	postgres.ConnectDatabase()

	// Run Migrations
	migration.Migrate()

	// Create Gin Router
	router := gin.Default()

	// Register Routes
	routes.AuthRoutes(router)

	// Start Server
	log.Println("Server running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}