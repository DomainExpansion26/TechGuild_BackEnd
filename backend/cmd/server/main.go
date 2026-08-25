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
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"

	"techguild-backend/src/config"
	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/jobs"
	"techguild-backend/src/middleware"
	"techguild-backend/src/routes"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	postgres.ConnectDatabase()

	migration.Migrate()

	jobs.StartCleanupJob()

	router := gin.Default()

	// Allowed origins are env-driven (FRONTEND_URL + ZUDOKU_URL, comma-separated).
	router.Use(middleware.CORS(cfg))

	// ---- public / unversioned routes ----
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

	apiConfig := huma.DefaultConfig("TechGuild Backend API", "1.0.0")
	apiConfig.Info.Description = "Backend API for TechGuild Platform."

	publicURL := os.Getenv("SERVER_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8080"
	}
	apiConfig.Servers = []*huma.Server{{URL: publicURL}}

	// JWT Bearer auth scheme — Zudoku/playground Authorize dialog ke liye.
	apiConfig.Components = &huma.Components{
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"BearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
	}
	// Default: sab routes ko bearerAuth chahiye (protected). Public routes
	apiConfig.Security = []map[string][]string{{"BearerAuth": {}}}

	api := humagin.New(router, apiConfig)

	// naye Huma routes register karo
	routes.RegisterAuthRoutes(api)
	routes.RegisterContractRoutes(api)
	routes.RegisterProfileRoutes(api)
	routes.RegisterOAuthRoutes(api)
	routes.RegisterMilestoneRoutes(api)
	routes.RegisterProjectRoutes(api)
	routes.RegisterProjectApplicationRoutes(api)
	routes.RegisterSubmissionRoutes(api)
	routes.RegisterTeamRoutes(api)
	routes.RegisterVerificationRoutes(api)

	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
