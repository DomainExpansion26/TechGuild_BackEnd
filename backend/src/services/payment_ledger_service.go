package services

import (
	"errors"
	"time"

	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrLedgerDuplicate = errors.New("ledger entry with same idempotency key already exists")

type PaymentLedgerService struct {
	txRepo repository.PaymentTransactionRepository
}

func NewPaymentLedgerService() *PaymentLedgerService {
	return &PaymentLedgerService{
		txRepo: repository.NewPaymentTransactionRepository(),
	}
}

// LedgerEntry is the input shape for appending one immutable ledger row.
//
// PaymentID and QuestID are POINTERS (nullable) because wallet withdrawals
// don't reference a specific payment or quest.
type LedgerEntry struct {
	PaymentID             *uuid.UUID
	QuestID               *uuid.UUID
	QuestMilestoneID      *uuid.UUID
	TransactionType       models.TransactionType
	Amount                int64 // signed
	Currency              string
	Provider              string
	ProviderTransactionID *string
	Reference             string
	IdempotencyKey        string
}

// Append writes a ledger entry inside an existing transaction.
func (s *PaymentLedgerService) Append(tx *gorm.DB, entry LedgerEntry) error {
	if entry.IdempotencyKey == "" {
		return errors.New("ledger entry requires idempotency key")
	}
	if entry.Amount == 0 {
		return errors.New("ledger entry amount must be non-zero")
	}

	// Check duplicate inside the same tx
	txRepo := repository.NewPaymentTransactionRepositoryTx(tx)
	exists, err := txRepo.ExistsByIdempotencyKey(entry.IdempotencyKey)
	if err != nil {
		return err
	}
	if exists {
		return ErrLedgerDuplicate
	}

	record := &models.PaymentTransaction{
		PaymentID:             entry.PaymentID,
		QuestID:               entry.QuestID,
		QuestMilestoneID:      entry.QuestMilestoneID,
		TransactionType:       entry.TransactionType,
		Amount:                entry.Amount,
		Currency:              entry.Currency,
		Provider:              entry.Provider,
		ProviderTransactionID: entry.ProviderTransactionID,
		Reference:             entry.Reference,
		IdempotencyKey:        entry.IdempotencyKey,
		CreatedAt:             time.Now(),
	}
	return txRepo.Create(record)
}
