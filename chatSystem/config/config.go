package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppName string
	AppEnv  string
	Port    string

	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBName      string
	DBUser      string
	DBPassword  string
	DBSSLMode   string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisUsername string
	RedisPassword string

	// JWT (Application Auth)
	JWTAccessSecret  string
	JWTRefreshSecret string

	// Centrifugo Config
	CentrifugoAPIURL             string
	CentrifugoAPIKey             string
	CentrifugoTokenHMACSecretKey string
	CentrifugoTokenTTL           time.Duration
	CentrifugoWSURL              string

	// Cloudflare R2 / S3 Storage
	R2AccountID       string
	R2AccessKeyID     string
	R2AccessKeySecret string
	R2BucketName      string
	R2PublicURL       string

	// File Upload Settings
	MaxFileSizeBytes int64

	// Path to the loaded .env file (if found)
	EnvFilePath string
}

// FindAndLoadEnv searches current and parent directories for .env and loads it
func FindAndLoadEnv() string {
	currDir, err := os.Getwd()
	var foundPath string
	if err == nil {
		dir := currDir
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, ".env")
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				if err := godotenv.Load(candidate); err == nil {
					foundPath = candidate
					viper.SetConfigFile(candidate)
					_ = viper.ReadInConfig()
					break
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	if foundPath == "" {
		// Fallback check standard relative paths
		candidates := []string{".env", "../.env", "../../.env", "../../../.env"}
		for _, c := range candidates {
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				if err := godotenv.Load(c); err == nil {
					foundPath = c
					viper.SetConfigFile(c)
					_ = viper.ReadInConfig()
					break
				}
			}
		}
	}

	viper.AutomaticEnv()
	return foundPath
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	loadedPath := FindAndLoadEnv()
	if loadedPath != "" {
		log.Printf("[ChatSystem Config] Loaded configuration from %s", loadedPath)
	}

	port := getEnv("PORT", "8080")
	if port == "" {
		port = "8080"
	}

	maxFileSizeStr := getEnv("MAX_FILE_SIZE_BYTES", "26214400") // 25 MB default
	maxFileSize, err := strconv.ParseInt(maxFileSizeStr, 10, 64)
	if err != nil {
		maxFileSize = 25 * 1024 * 1024
	}

	ttlSecondsStr := getEnv("CENTRIFUGO_TOKEN_TTL_SECONDS", "3600")
	ttlSeconds, err := strconv.Atoi(ttlSecondsStr)
	if err != nil {
		ttlSeconds = 3600
	}

	centrifugoAPIURL := getEnv("CENTRIFUGO_API_URL", "http://localhost:8000/api")
	centrifugoAPIKey := getEnv("CENTRIFUGO_API_KEY", "centrifugo-api-key-techguild")
	centrifugoSecret := getEnv("CENTRIFUGO_TOKEN_HMAC_SECRET_KEY", "centrifugo-token-secret-techguild")
	centrifugoWS := getEnv("CENTRIFUGO_WS_URL", "wss://chat.techguild.io/connection/websocket")

	// JWT Access Secret (support JWT_ACCESS_SECRET or JWT_SECRET)
	jwtAccess := getEnv("JWT_ACCESS_SECRET", "")
	if jwtAccess == "" {
		jwtAccess = getEnv("JWT_SECRET", "temporary-default-secret-for-test-environments")
	}

	jwtRefresh := getEnv("JWT_REFRESH_SECRET", "temporary-default-secret-for-test-environments")

	cfg := &Config{
		AppName:                      getEnv("APP_NAME", "techguild-chatsystem"),
		AppEnv:                       getEnv("APP_ENV", "development"),
		Port:                         port,
		DatabaseURL:                  getEnv("DATABASE_URL", ""),
		DBHost:                       getEnv("DB_HOST", "localhost"),
		DBPort:                       getEnv("DB_PORT", "5432"),
		DBName:                       getEnv("DB_NAME", "techguild"),
		DBUser:                       getEnv("DB_USER", "postgres"),
		DBPassword:                   getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:                    getEnv("DB_SSLMODE", "disable"),
		RedisHost:                    getEnv("REDIS_HOST", "localhost"),
		RedisPort:                    getEnv("REDIS_PORT", "6379"),
		RedisUsername:                getEnv("REDIS_USERNAME", ""),
		RedisPassword:                getEnv("REDIS_PASSWORD", ""),
		JWTAccessSecret:              jwtAccess,
		JWTRefreshSecret:             jwtRefresh,
		CentrifugoAPIURL:             centrifugoAPIURL,
		CentrifugoAPIKey:             centrifugoAPIKey,
		CentrifugoTokenHMACSecretKey: centrifugoSecret,
		CentrifugoTokenTTL:           time.Duration(ttlSeconds) * time.Second,
		CentrifugoWSURL:              centrifugoWS,
		R2AccountID:                  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:                getEnv("R2_ACCESS_KEY_ID", ""),
		R2AccessKeySecret:            getEnv("R2_ACCESS_KEY_SECRET", ""),
		R2BucketName:                 getEnv("R2_BUCKET_NAME", "techguild-chat-files"),
		R2PublicURL:                  getEnv("R2_PUBLIC_URL", "https://storage.techguild.io"),
		MaxFileSizeBytes:             maxFileSize,
		EnvFilePath:                  loadedPath,
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if val := viper.GetString(key); val != "" {
		return val
	}
	return defaultVal
}
