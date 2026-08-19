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
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-contrib/cors"
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

	// CORS — Zudoku (ya kisi bhi frontend) se requests allow karne ke liye
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001", "https://your-zudoku-domain.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	config := huma.DefaultConfig("TechGuild Backend API", "1.0.0")
	config.Info.Description = "Backend API for TechGuild Platform."
	config.Servers = []*huma.Server{
		{URL: "http://localhost:8080"},
	}
	api := humagin.New(router, config)

	// naye Huma routes register karo
	routes.RegisterAuthRoutes(api)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Static("/uploads", "./uploads")

	router.HEAD("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "TechGuild API is running"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Authentication (SetAccountType — Gin, baaki sab Huma me migrate ho chuka)
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
