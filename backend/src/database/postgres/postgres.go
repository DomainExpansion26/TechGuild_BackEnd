package postgres

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisDB *redis.Client

// redisTLSEnabled reports whether the Redis connection should use TLS.
// REDIS_SSL is optional: missing, empty, "false", "0", etc. mean plain redis://
// (required by Redis Cloud Free tier and local Redis).
func redisTLSEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REDIS_SSL"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func ConnectDatabase() {

	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found")
	}
	dsn := os.Getenv("DATABASE_URL")

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Disable prepared statement caching.
		//
		// pgx driver reuses prepared statements per connection. When schema
		// changes (e.g. DROP TABLE + AutoMigrate during development), cached
		// plans become stale and Postgres rejects them with:
		//   "cached plan must not change result type" (SQLSTATE 0A000)
		//   "prepared statement name is already in use" (SQLSTATE 08P01)
		//
		// PrepareStmt: false avoids both by not using prepared statement cache.
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatal("Postgres connection failed:", err)
	}
	log.Println("PostgreSQL Connected")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	if redisHost == "" || redisPort == "" {
		log.Fatal("Missing Redis environment variables")
	}

	redisOptions := &redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}

	// TLS is opt-in. Redis Cloud Free tier and local Redis expect a plain
	// connection, so only set a TLS config when REDIS_SSL is explicitly true.
	if redisTLSEnabled() {
		redisOptions.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		log.Println("Redis TLS enabled")
	} else {
		log.Println("Redis TLS disabled (plain redis://)")
	}

	RedisDB = redis.NewClient(redisOptions)
	_, err = RedisDB.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}
	log.Println("Redis Connected")
}
