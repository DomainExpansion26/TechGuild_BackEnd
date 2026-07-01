package config

import "github.com/spf13/viper"

func LoadConfig() (*Config, error) {

	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppName: viper.GetString("APP_NAME"),
		AppEnv:  viper.GetString("APP_ENV"),
		Port:    viper.GetString("PORT"),

		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBName:     viper.GetString("DB_NAME"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBSSLMode:  viper.GetString("DB_SSLMODE"),

		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetString("REDIS_PORT"),
		RedisUsername: viper.GetString("REDIS_USERNAME"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		JWTSecret:        viper.GetString("JWT_SECRET"),
		JWTRefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
	}

	return cfg, nil
}