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

	_ "techguild-backend/docs"

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

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	
	router.Static("/uploads", "./uploads")

	// Authentication
	routes.AuthRoutes(router)

	// OAuth
	routes.OAuthRoutes(router)

	// Profile
	routes.ProfileRoutes(router)

	// Verification
	routes.VerificationRoutes(router)

	// Projects
	routes.ProjectRoutes(router)

	// Project Applications
	routes.ProjectApplicationRoutes(router)

	// Contracts
	routes.ContractRoutes(router)

	// Milestones
	routes.MilestoneRoutes(router)

	// Submissions
	routes.SubmissionRoutes(router)

	//team
	routes.TeamRoutes(router)

	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}