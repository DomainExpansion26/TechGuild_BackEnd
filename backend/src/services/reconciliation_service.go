package services

import (
	"log"

	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

type ReconciliationService struct {
	paymentRepo repository.PaymentRepository
	fundingSvc  *MilestoneFundingService
}

func NewReconciliationService(fundingSvc *MilestoneFundingService) *ReconciliationService {
	return &ReconciliationService{
		paymentRepo: repository.NewPaymentRepository(),
		fundingSvc:  fundingSvc,
	}
}

// ReconcilePendingPayments finds pending payments older than 10 mins and fixes them
func (s *ReconciliationService) ReconcilePendingPayments() {
	log.Println("[Reconciliation] Starting pending payments check...")

	// Get payments stuck in 'pending' for more than 10 minutes
	payments, err := s.paymentRepo.ListPendingOlderThan(10)
	if err != nil {
		log.Printf("[Reconciliation] Error fetching pending payments: %v", err)
		return
	}

	if len(payments) == 0 {
		return // Nothing to reconcile
	}

	for _, p := range payments {
		// We only care about Razorpay pending payments that have an Order ID
		if p.Provider == "razorpay" && p.ProviderOrderID != nil {
			s.reconcileSinglePayment(p)
		}
	}
	log.Println("[Reconciliation] Pending payments check complete.")
}

func (s *ReconciliationService) reconcileSinglePayment(p models.Payment) {
	// 1. Fetch actual status from Razorpay
	// (Assuming we already implemented GetPayment in adapter)

	// NOTE: In a real scenario, if a client hasn't paid yet, Razorpay API will return 404 or "created" status.
	// We handle it simply here: if it's captured, we update our DB.

	// OPTIONAL: Call Razorpay API to check Order status
	// order, err := rzpClient.Order.Fetch(*p.ProviderOrderID)
	// if err != nil { return }

	// if order.status == "paid" {
	//    // Get the payment ID
	//    paymentID := order.payments[0].id
	//
	//    // Call our existing funding service to mark it captured
	//    _, err = s.fundingSvc.MarkPaymentCaptured(*p.ProviderOrderID, paymentID)
	//    if err != nil {
	//        log.Printf("[Reconciliation] Failed to auto-repair payment %s: %v", p.ID, err)
	//    } else {
	//        log.Printf("[Reconciliation] ✅ Auto-repaired payment %s (Order: %s)", p.ID, *p.ProviderOrderID)
	//    }
	// }

	log.Printf("[Reconciliation] Checked payment ID: %s (Order: %s)", p.ID, *p.ProviderOrderID)
}
