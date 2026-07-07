package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/routes"
)

func main() {
	postgres.ConnectDatabase()
	migration.Migrate()

	router := gin.Default()
	routes.AuthRoutes(router)
	routes.OAuthRoutes(router)
	log.Println("Server running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}