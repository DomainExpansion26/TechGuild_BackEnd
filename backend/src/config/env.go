package config

import (
	"github.com/spf13/viper"
)

func LoadConfig() (*Config, error) {

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

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

		FrontendURL: viper.GetString("FRONTEND_URL"),
		ZudokuURL:   viper.GetString("ZUDOKU_URL"),

		SMTPHost:     viper.GetString("SMTP_HOST"),
		SMTPPort:     viper.GetString("SMTP_PORT"),
		SMTPEmail:    viper.GetString("SMTP_EMAIL"),
		SMTPPassword: viper.GetString("SMTP_PASSWORD"),

		PaymentProvider:       viper.GetString("PAYMENT_PROVIDER"),
		RazorpayKeyID:         viper.GetString("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     viper.GetString("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: viper.GetString("RAZORPAY_WEBHOOK_SECRET"),

		RazorpayXKeyID:         viper.GetString("RAZORPAYX_KEY_ID"),
		RazorpayXKeySecret:     viper.GetString("RAZORPAYX_KEY_SECRET"),
		RazorpayXAccountNumber: viper.GetString("RAZORPAYX_ACCOUNT_NUMBER"),

		ClientProcessingFeeBps:     viper.GetInt64("CLIENT_PROCESSING_FEE_BPS"),
		FreelancerProcessingFeeBps: viper.GetInt64("FREELANCER_PROCESSING_FEE_BPS"),
	}

	return cfg, nil
}
