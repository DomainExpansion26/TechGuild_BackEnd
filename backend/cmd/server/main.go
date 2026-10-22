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

	router.Use(middleware.CORS(cfg))

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
	// Payment Provider (selected via PAYMENT_PROVIDER env)
	// --------------------------------------------------

	paymentProvider := buildPaymentProvider(cfg)

	// --------------------------------------------------
	// Payment & Payout services
	// --------------------------------------------------

	paymentMethodService := services.NewPaymentMethodService(cfg, paymentProvider)
	payoutMethodService := services.NewPayoutMethodService(cfg, paymentProvider)

	paymentMethodController := controllers.NewPaymentMethodController(paymentMethodService)
	payoutMethodController := controllers.NewPayoutMethodController(payoutMethodService)

	// --------------------------------------------------
	// Quest Milestone flow
	// --------------------------------------------------

	questMilestoneController := controllers.NewQuestMilestoneController(cfg)
	milestoneFundingService := services.NewMilestoneFundingService(cfg, paymentProvider)
	milestoneFundingController := controllers.NewMilestoneFundingControllerFromService(milestoneFundingService)
	milestoneReleaseController := controllers.NewMilestoneReleaseController(cfg, paymentProvider)

	jobs.StartReconciliationJob(milestoneFundingService)

	// --------------------------------------------------
	// Wallet + Withdrawal + Refund
	// --------------------------------------------------

	walletController := controllers.NewWalletController(cfg)
	walletWithdrawalController := controllers.NewWalletWithdrawalController(cfg, paymentProvider)
	refundController := controllers.NewRefundController(cfg, paymentProvider)

	// --------------------------------------------------
	// Webhook infrastructure
	// --------------------------------------------------

	webhookEventRepo := repository.NewPaymentWebhookEventRepository()
	payoutMethodRepoForWebhook := repository.NewPayoutMethodRepository()
	webhookService := services.NewWebhookService(
		webhookEventRepo,
		payoutMethodRepoForWebhook,
		milestoneFundingService,
	)

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

	routes.RegisterAuthRoutes(api, authController)
	routes.RegisterProfileRoutes(api, profileController)
	routes.RegisterSettingsRoutes(api, settingsController)
	routes.RegisterOAuthRoutes(api, oauthController)

	routes.RegisterPartyRoutes(api)
	routes.RegisterVerificationRoutes(api)
	routes.RegisterTwoFactorRoutes(api)

	routes.RegisterQuestRoutes(api)
	routes.RegisterQuestApplicationRoutes(api)
	routes.RegisterTaskManagementRoutes(api)

	routes.RegisterQuestMilestoneRoutes(api, questMilestoneController)
	routes.RegisterMilestoneFundingRoutes(api, milestoneFundingController)
	routes.RegisterMilestoneReleaseRoutes(api, milestoneReleaseController)
	routes.RegisterWalletRoutes(api, walletController)
	routes.RegisterWalletWithdrawalRoutes(api, walletWithdrawalController)
	routes.RegisterRefundRoutes(api, refundController)

	routes.RegisterPaymentMethodRoutes(api, paymentMethodController)
	routes.RegisterPayoutMethodRoutes(api, payoutMethodController)

	routes.RegisterWebhookRoutes(api, webhookController)

	// -------- DISABLED (project-based, replaced by Quest flow) --------
	// routes.RegisterContractRoutes(api)
	// routes.RegisterMilestoneRoutes(api)
	// routes.RegisterProjectRoutes(api)
	// routes.RegisterProjectApplicationRoutes(api)
	// routes.RegisterSubmissionRoutes(api)

	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
