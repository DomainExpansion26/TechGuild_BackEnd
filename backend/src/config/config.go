package config

type Config struct {
	AppName string
	AppEnv  string
	Port    string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	RedisHost     string
	RedisPort     string
	RedisUsername string
	RedisPassword string

	JWTSecret        string
	JWTRefreshSecret string

	FrontendURL string
	ZudokuURL   string
}
