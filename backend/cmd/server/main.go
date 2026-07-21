// @title TechGuild Backend API
// @version 1.0
// @description Backend API for TechGuild Platform.
// @termsOfService http://swagger.io/terms/

// @contact.name TechGuild Team
// @contact.email support@techguild.com

// @license.name MIT

// @host https://techguild-backend.onrender.com
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_"techguild-backend/docs"

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

	// Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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