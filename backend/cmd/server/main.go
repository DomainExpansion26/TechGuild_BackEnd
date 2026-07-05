package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"techguild-backend/src/config"
	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/routes"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = postgres.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = migration.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
routes.AuthRoutes(router)

	log.Printf("🚀 Server started on port %s", cfg.Port)

	router.Run(":" + cfg.Port)
}