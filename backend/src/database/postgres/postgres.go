package postgres

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisDB *redis.Client

func ConnectDatabase() {

	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found")
	}
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" || port == "" || user == "" || dbname == "" {
		log.Fatal("Missing DB environment variables")
	}
	dsn := "host=" + host +
		" user=" + user +
		" password=" + password +
		" dbname=" + dbname +
		" port=" + port +
		" sslmode=" + sslmode
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Postgres connection failed:", err)
	}
	log.Println("PostgreSQL Connected")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	if redisHost == "" || redisPort == "" {
		log.Fatal("Missing Redis environment variables")
	}
	RedisDB = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	_, err = RedisDB.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}
	log.Println("Redis Connected")
}