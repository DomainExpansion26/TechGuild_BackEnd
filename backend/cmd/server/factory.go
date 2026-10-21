package main

import (
	"log"
	"strings"

	"techguild-backend/src/config"
	"techguild-backend/src/services"
	razorpay "techguild-backend/src/services/razorpay"
)

func buildPaymentProvider(cfg *config.Config) services.PaymentProvider {
	providerName := strings.TrimSpace(strings.ToLower(cfg.PaymentProvider))
	if providerName == "" {
		providerName = "mock"
	}

	switch providerName {
	case "mock":
		log.Println("payment provider: mock")
		return services.NewMockPaymentProvider()

	case "razorpay":
		keyID := cfg.RazorpayKeyID
		keySecret := cfg.RazorpayKeySecret
		if keyID == "" {
			keyID = cfg.RazorpayXKeyID
			keySecret = cfg.RazorpayXKeySecret
		}
		if keyID == "" || keySecret == "" {
			log.Fatal("razorpay selected but keys missing")
		}

		rzpClient, err := razorpay.NewClient(keyID, keySecret)
		if err != nil {
			log.Fatal("razorpay client init failed: ", err)
		}

		if cfg.RazorpayXAccountNumber != "" {
			log.Println("payment provider: razorpay (with RazorpayX payouts)")
			return razorpay.NewAdapterWithPayouts(rzpClient, cfg.RazorpayXAccountNumber)
		}

		log.Println("payment provider: razorpay (payments only, no RazorpayX)")
		return razorpay.NewAdapter(rzpClient)

	default:
		log.Fatalf("unknown PAYMENT_PROVIDER: %s", providerName)
		return nil
	}
}
