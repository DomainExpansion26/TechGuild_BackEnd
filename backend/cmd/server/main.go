package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"techguild-backend/src/config"
	"techguild-backend/src/database/migration"
	"techguild-backend/src/database/postgres"
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

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "TechGuild Backend Running",
		})
	})

	log.Printf("🚀 Server started on port %s", cfg.Port)

	router.Run(":" + cfg.Port)
}