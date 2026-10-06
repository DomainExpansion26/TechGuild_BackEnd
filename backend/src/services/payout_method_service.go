package services

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrPayoutMethodNotFound  = errors.New("payout method not found")
	ErrPayoutMethodDuplicate = errors.New("payout method already exists")
	ErrPayoutValidation      = errors.New("payout method validation failed")
	ErrPayoutNotVerified     = errors.New("payout method must be verified before setting as primary")
)

type PayoutMethodService struct {
	cfg      *config.Config
	provider PaymentProvider
	repo     repository.PayoutMethodRepository
}

func NewPayoutMethodService(cfg *config.Config, provider PaymentProvider) *PayoutMethodService {
	return &PayoutMethodService{
		cfg:      cfg,
		provider: provider,
		repo:     repository.NewPayoutMethodRepository(),
	}
}

// AddPayoutMethod creates a payout destination by sending raw details to
// the provider. Only masked/non-sensitive metadata is persisted.
func (s *PayoutMethodService) AddPayoutMethod(userIDStr string, req dto.AddPayoutMethodRequest) (*dto.PayoutMethodResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	if err := validatePayoutMethodRequest(req); err != nil {
		return nil, err
	}

	// Build provider request with raw details
	fundReq := FundAccountRequest{
		Type:          req.Type,
		AccountHolder: derefString(req.AccountHolder),
		AccountNumber: derefString(req.AccountNumber),
		IFSC:          derefString(req.IFSC),
		VPA:           derefString(req.VPA),
		Last4:         derefString(req.Last4),
		Brand:         derefString(req.Brand),
	}

	result, err := s.provider.CreatePayoutFundAccount(fundReq)
	if err != nil {
		log.Printf("provider fund account creation failed for user=%s: %v", userIDStr, err)
		return nil, ErrPaymentProviderFailed
	}

	method := &models.PayoutMethod{
		UserID:           userID,
		Purpose:          models.PayoutMethodPurpose(req.Purpose),
		Type:             models.PayoutMethodType(req.Type),
		Provider:         req.Provider,
		ProviderMethodID: &result.ProviderFundAccountID,
		BankName:         req.BankName,
		AccountHolder:    req.AccountHolder,
		IFSC:             req.IFSC,
		VPA:              req.VPA,
		Last4:            req.Last4,
		Brand:            req.Brand,
		Status:           models.PayoutMethodStatus(result.Status),
	}
	if result.MaskedAccount != "" {
		method.MaskedAccount = &result.MaskedAccount
	}

	if err := s.repo.Create(method); err != nil {
		log.Printf("failed to store payout method for user=%s: %v", userIDStr, err)
		return nil, errors.New("failed to save payout method")
	}

	// Set primary if requested AND verified
	if req.SetPrimary && method.Status == models.PayoutMethodStatusVerified {
		if err := s.repo.SetPrimary(method.ID, userID); err != nil {
			log.Printf("failed to set primary payout method=%s user=%s: %v", method.ID, userIDStr, err)
		} else {
			method.IsPrimary = true
		}
	}

	return toPayoutMethodResponse(method), nil
}

func (s *PayoutMethodService) ListPayoutMethods(userIDStr string) (*dto.ListPayoutMethodsResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	methods, err := s.repo.ListByUser(userID)
	if err != nil {
		log.Printf("list payout methods failed for user=%s: %v", userIDStr, err)
		return nil, errors.New("failed to fetch payout methods")
	}

	resp := &dto.ListPayoutMethodsResponse{
		Methods: make([]dto.PayoutMethodResponse, 0, len(methods)),
	}
	for i := range methods {
		resp.Methods = append(resp.Methods, *toPayoutMethodResponse(&methods[i]))
	}
	return resp, nil
}

func (s *PayoutMethodService) SetPrimaryPayoutMethod(userIDStr, methodIDStr string) (*dto.PayoutMethodResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	methodID, err := uuid.Parse(methodIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid method id")
	}

	if err := s.repo.SetPrimary(methodID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPayoutMethodNotFound
		}
		if strings.Contains(err.Error(), "must be verified") {
			return nil, ErrPayoutNotVerified
		}
		log.Printf("set primary payout failed for method=%s user=%s: %v", methodID, userIDStr, err)
		return nil, errors.New("failed to set primary payout method")
	}

	method, err := s.repo.GetByIDAndUser(methodID, userID)
	if err != nil {
		return nil, ErrPayoutMethodNotFound
	}
	return toPayoutMethodResponse(method), nil
}

func (s *PayoutMethodService) DeletePayoutMethod(userIDStr, methodIDStr string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	methodID, err := uuid.Parse(methodIDStr)
	if err != nil {
		return fmt.Errorf("invalid method id")
	}

	method, err := s.repo.GetByIDAndUser(methodID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPayoutMethodNotFound
		}
		return err
	}

	// Best-effort provider cleanup
	if method.ProviderMethodID != nil && *method.ProviderMethodID != "" {
		if err := s.provider.DeletePayoutFundAccount(*method.ProviderMethodID); err != nil {
			log.Printf("provider fund account delete failed for method=%s: %v (continuing)", methodID, err)
		}
	}

	if err := s.repo.Delete(methodID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPayoutMethodNotFound
		}
		log.Printf("local payout delete failed for method=%s user=%s: %v", methodID, userIDStr, err)
		return errors.New("failed to delete payout method")
	}
	return nil
}

// -------- helpers --------

func validatePayoutMethodRequest(req dto.AddPayoutMethodRequest) error {
	if req.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrPayoutValidation)
	}
	switch req.Type {
	case "bank":
		if req.AccountHolder == nil || *req.AccountHolder == "" {
			return fmt.Errorf("%w: account_holder is required for bank", ErrPayoutValidation)
		}
		if req.AccountNumber == nil || *req.AccountNumber == "" {
			return fmt.Errorf("%w: account_number is required for bank", ErrPayoutValidation)
		}
		if req.IFSC == nil || *req.IFSC == "" {
			return fmt.Errorf("%w: ifsc is required for bank", ErrPayoutValidation)
		}
	case "upi":
		if req.VPA == nil || *req.VPA == "" {
			return fmt.Errorf("%w: vpa is required for upi", ErrPayoutValidation)
		}
	case "card":
		if req.Last4 == nil || *req.Last4 == "" {
			return fmt.Errorf("%w: last4 is required for card", ErrPayoutValidation)
		}
	case "wallet":
		// nothing extra
	default:
		return fmt.Errorf("%w: invalid type %s", ErrPayoutValidation, req.Type)
	}
	return nil
}

func toPayoutMethodResponse(m *models.PayoutMethod) *dto.PayoutMethodResponse {
	resp := &dto.PayoutMethodResponse{
		ID:        m.ID.String(),
		Type:      string(m.Type),
		Purpose:   string(m.Purpose),
		Status:    string(m.Status),
		Provider:  m.Provider,
		IsPrimary: m.IsPrimary,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.BankName != nil {
		resp.BankName = *m.BankName
	}
	if m.AccountHolder != nil {
		resp.AccountHolder = *m.AccountHolder
	}
	if m.MaskedAccount != nil {
		resp.MaskedAccount = *m.MaskedAccount
	}
	if m.IFSC != nil {
		resp.IFSC = *m.IFSC
	}
	if m.VPA != nil {
		resp.VPA = maskVPA(*m.VPA)
	}
	if m.Last4 != nil {
		resp.Last4 = *m.Last4
	}
	if m.Brand != nil {
		resp.Brand = *m.Brand
	}
	return resp
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
