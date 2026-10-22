package services

import (
	"log"
	"os"

	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/razorpay/razorpay-go"
)

type ReconciliationService struct {
	paymentRepo repository.PaymentRepository
	fundingSvc  *MilestoneFundingService
	rzpClient   *razorpay.Client
}

func NewReconciliationService(fundingSvc *MilestoneFundingService) *ReconciliationService {
	// .env se Razorpay keys uthayenge
	keyID := os.Getenv("RAZORPAY_KEY_ID")
	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")

	var client *razorpay.Client
	if keyID != "" && keySecret != "" {
		client = razorpay.NewClient(keyID, keySecret)
	}

	return &ReconciliationService{
		paymentRepo: repository.NewPaymentRepository(),
		fundingSvc:  fundingSvc,
		rzpClient:   client,
	}
}

func (s *ReconciliationService) ReconcilePendingPayments() {
	log.Println("[Reconciliation] Starting pending payments check...")

	// 10 minute se purane pending/failed payments fetch karo
	payments, err := s.paymentRepo.ListPendingOlderThan(10)
	if err != nil {
		log.Printf("[Reconciliation] Error fetching pending payments: %v", err)
		return
	}

	if len(payments) == 0 {
		return
	}

	for _, p := range payments {
		// Sirf Razorpay payments ko process karo jinke paas Order ID hai
		if p.Provider == "razorpay" && p.ProviderOrderID != nil {
			s.reconcileSinglePayment(p)
		}
	}
	log.Println("[Reconciliation] Pending payments check complete.")
}

func (s *ReconciliationService) reconcileSinglePayment(p models.Payment) {
	if s.rzpClient == nil {
		log.Println("[Reconciliation] Skipping: Razorpay client not initialized (check .env keys)")
		return
	}

	log.Printf("[Reconciliation] Checking Razorpay Order: %s", *p.ProviderOrderID)

	// 1. Razorpay API se Order fetch karo
	orderData, err := s.rzpClient.Order.Fetch(*p.ProviderOrderID, nil, nil)
	if err != nil {
		log.Printf("[Reconciliation] Failed to fetch order %s: %v", *p.ProviderOrderID, err)
		return
	}

	// 2. Order ka status check karo
	status, ok := orderData["status"].(string)
	if !ok {
		return
	}

	if status == "paid" {
		// 3. Payments list nikal kar Payment ID extract karo
		paymentsList, ok := orderData["payments"].([]interface{})
		if ok && len(paymentsList) > 0 {
			paymentData := paymentsList[0].(map[string]interface{})
			paymentID := paymentData["id"].(string)

			// 4. Call Funding Service to mark it as captured
			_, err := s.fundingSvc.MarkPaymentCaptured(*p.ProviderOrderID, paymentID)
			if err != nil {
				log.Printf("[Reconciliation] Failed to auto-repair payment %s: %v", p.ID, err)
			} else {
				log.Printf("[Reconciliation] ✅ Auto-repaired payment %s (Order: %s)", p.ID, *p.ProviderOrderID)
			}
		} else {
			log.Printf("[Reconciliation] Order %s is paid but no payment ID found.", *p.ProviderOrderID)
		}
	} else {
		log.Printf("[Reconciliation] Order %s is not paid (Status: %s). Skipping.", *p.ProviderOrderID, status)
	}
}
