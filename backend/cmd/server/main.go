package main

import (
	"log"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"

	"techguild-backend/src/config"
	"techguild-backend/src/controllers"
	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/jobs"
	"techguild-backend/src/middleware"
	"techguild-backend/src/repository"
	"techguild-backend/src/routes"
	"techguild-backend/src/services"
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

	// JWT Bearer auth scheme
	apiConfig.Components = &huma.Components{
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"BearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
	}
	apiConfig.Security = []map[string][]string{{"BearerAuth": {}}}

	api := humagin.New(router, apiConfig)

	// --------------------------------------------------
	// Payment & Payout (provider-neutral)
	// --------------------------------------------------

	// Provider — abhi mock, baad me Razorpay/Stripe/Payoneer adapter swap karenge.
	paymentProvider := services.NewMockPaymentProvider()

	paymentMethodService := services.NewPaymentMethodService(cfg, paymentProvider)
	payoutMethodService := services.NewPayoutMethodService(cfg, paymentProvider)

	paymentMethodController := controllers.NewPaymentMethodController(paymentMethodService)
	payoutMethodController := controllers.NewPayoutMethodController(payoutMethodService)

	// --------------------------------------------------
	// Quest Milestone (freelancer creates, client reviews)
	// --------------------------------------------------

	questMilestoneController := controllers.NewQuestMilestoneController(cfg)

	// --------------------------------------------------
	// Webhook infrastructure
	// --------------------------------------------------

	webhookEventRepo := repository.NewPaymentWebhookEventRepository()
	payoutMethodRepoForWebhook := repository.NewPayoutMethodRepository()
	webhookService := services.NewWebhookService(webhookEventRepo, payoutMethodRepoForWebhook)

	// Register provider verifiers (only if credentials configured)
	if cfg.RazorpayWebhookSecret != "" {
		webhookService.RegisterVerifier(
			services.NewRazorpayWebhookVerifier(cfg.RazorpayWebhookSecret),
		)
	}

	webhookController := controllers.NewWebhookController(webhookService)

	// --------------------------------------------------
	// Other controllers
	// --------------------------------------------------

	authController := controllers.NewAuthController(cfg)
	profileController := controllers.NewProfileController(cfg)
	settingsController := controllers.NewSettingsController(cfg)
	oauthController := controllers.NewOAuthController(cfg)

	// --------------------------------------------------
	// Route registration
	// --------------------------------------------------

	// Core
	routes.RegisterAuthRoutes(api, authController)
	routes.RegisterProfileRoutes(api, profileController)
	routes.RegisterSettingsRoutes(api, settingsController)
	routes.RegisterOAuthRoutes(api, oauthController)

	// Party, Verification, 2FA
	routes.RegisterPartyRoutes(api)
	routes.RegisterVerificationRoutes(api)
	routes.RegisterTwoFactorRoutes(api)

	// Quest flow
	routes.RegisterQuestRoutes(api)
	routes.RegisterTaskManagementRoutes(api)

	// Quest Milestones (freelancer CRUD + client review)
	routes.RegisterQuestMilestoneRoutes(api, questMilestoneController)

	// Payment & Payout methods (client-side + freelancer payout methods)
	routes.RegisterPaymentMethodRoutes(api, paymentMethodController)
	routes.RegisterPayoutMethodRoutes(api, payoutMethodController)

	// Webhooks
	routes.RegisterWebhookRoutes(api, webhookController)

	// -------- DISABLED (project-based, will be replaced with Quest-based) --------
	// routes.RegisterContractRoutes(api)
	// routes.RegisterMilestoneRoutes(api)
	// routes.RegisterProjectRoutes(api)
	// routes.RegisterProjectApplicationRoutes(api)
	// routes.RegisterSubmissionRoutes(api)
	// routes.RegisterMilestonePaymentRoutes(api, milestonePaymentController)

	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
