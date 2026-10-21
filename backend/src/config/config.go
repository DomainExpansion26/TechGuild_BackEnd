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

	SMTPHost     string
	SMTPPort     string
	SMTPEmail    string
	SMTPPassword string

	// Provider selection
	PaymentProvider string // "mock" | "razorpay"

	// Razorpay credentials
	// Razorpay credentials
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string

	// RazorpayX (payouts)
	RazorpayXKeyID         string
	RazorpayXKeySecret     string
	RazorpayXAccountNumber string
	// NEW — Fee configuration
	ClientProcessingFeeBps     int64 // 500 = 5.00%
	FreelancerProcessingFeeBps int64 // 500 = 5.00%
}
