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
	ErrPaymentMethodNotFound      = errors.New("payment method not found")
	ErrPaymentMethodDuplicate     = errors.New("payment method already exists")
	ErrPaymentMethodForbidden     = errors.New("not allowed to access this payment method")
	ErrPaymentProviderFailed      = errors.New("payment provider operation failed")
	ErrPaymentProviderUnsupported = errors.New("operation not supported by provider")
)

type PaymentMethodService struct {
	cfg          *config.Config
	provider     PaymentProvider
	customerRepo repository.PaymentCustomerRepository
	methodRepo   repository.ClientPaymentMethodRepository
}

func NewPaymentMethodService(cfg *config.Config, provider PaymentProvider) *PaymentMethodService {
	return &PaymentMethodService{
		cfg:          cfg,
		provider:     provider,
		customerRepo: repository.NewPaymentCustomerRepository(),
		methodRepo:   repository.NewClientPaymentMethodRepository(),
	}
}

// AddClientPaymentMethod stores a client-provided tokenized payment method.
// The raw card/UPI data NEVER reaches this service — only a provider token.
func (s *PaymentMethodService) AddClientPaymentMethod(userIDStr string, req dto.AddClientPaymentMethodRequest) (*dto.ClientPaymentMethodResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	if err := validateClientPaymentMethodRequest(req); err != nil {
		return nil, err
	}

	// Step 1: get or create provider customer for this user
	customer, err := s.getOrCreateCustomer(userIDStr, req.Provider)
	if err != nil {
		return nil, err
	}

	// Step 2: attach token to customer (some providers don't need this)
	if err := s.provider.AttachPaymentMethod(customer.ProviderCustomerID, req.ProviderMethodID); err != nil {
		log.Printf("attach payment method failed for user=%s provider=%s: %v", userIDStr, req.Provider, err)
		return nil, ErrPaymentProviderFailed
	}

	// Step 3: guard against duplicate storage
	exists, err := s.methodRepo.ExistsByProviderMethodID(req.Provider, req.ProviderMethodID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrPaymentMethodDuplicate
	}

	method := &models.ClientPaymentMethod{
		UserID:            userID,
		PaymentCustomerID: customer.ID,
		Provider:          req.Provider,
		ProviderMethodID:  req.ProviderMethodID,
		Type:              models.ClientPaymentMethodType(req.Type),
		Last4:             req.Last4,
		Brand:             req.Brand,
		ExpMonth:          req.ExpMonth,
		ExpYear:           req.ExpYear,
		VPAMasked:         req.VPAMasked,
		Status:            models.ClientPaymentMethodStatusActive,
	}

	if err := s.methodRepo.Create(method); err != nil {
		log.Printf("failed to store payment method for user=%s: %v", userIDStr, err)
		return nil, errors.New("failed to save payment method")
	}

	// Step 4: set as primary if requested
	if req.SetPrimary {
		if err := s.methodRepo.SetPrimary(method.ID, userID); err != nil {
			// Non-fatal: method saved, just primary flag failed
			log.Printf("failed to set primary for method=%s user=%s: %v", method.ID, userIDStr, err)
		} else {
			method.IsPrimary = true
		}
	}

	return toClientPaymentMethodResponse(method), nil
}

func (s *PaymentMethodService) ListClientPaymentMethods(userIDStr string) (*dto.ListClientPaymentMethodsResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	methods, err := s.methodRepo.ListByUser(userID)
	if err != nil {
		log.Printf("list payment methods failed for user=%s: %v", userIDStr, err)
		return nil, errors.New("failed to fetch payment methods")
	}

	resp := &dto.ListClientPaymentMethodsResponse{
		Methods: make([]dto.ClientPaymentMethodResponse, 0, len(methods)),
	}
	for i := range methods {
		resp.Methods = append(resp.Methods, *toClientPaymentMethodResponse(&methods[i]))
	}
	return resp, nil
}

func (s *PaymentMethodService) SetPrimaryClientPaymentMethod(userIDStr, methodIDStr string) (*dto.ClientPaymentMethodResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	methodID, err := uuid.Parse(methodIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid method id")
	}

	if err := s.methodRepo.SetPrimary(methodID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		log.Printf("set primary failed for method=%s user=%s: %v", methodID, userIDStr, err)
		return nil, errors.New("failed to set primary payment method")
	}

	method, err := s.methodRepo.GetByIDAndUser(methodID, userID)
	if err != nil {
		return nil, ErrPaymentMethodNotFound
	}
	return toClientPaymentMethodResponse(method), nil
}

func (s *PaymentMethodService) DeleteClientPaymentMethod(userIDStr, methodIDStr string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	methodID, err := uuid.Parse(methodIDStr)
	if err != nil {
		return fmt.Errorf("invalid method id")
	}

	method, err := s.methodRepo.GetByIDAndUser(methodID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPaymentMethodNotFound
		}
		return err
	}

	// Provider detach (best-effort — don't block local delete on provider failure)
	if err := s.provider.DetachPaymentMethod(method.ProviderMethodID); err != nil {
		log.Printf("provider detach failed for method=%s: %v (continuing with local delete)", methodID, err)
	}

	if err := s.methodRepo.Delete(methodID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPaymentMethodNotFound
		}
		log.Printf("local delete failed for method=%s user=%s: %v", methodID, userIDStr, err)
		return errors.New("failed to delete payment method")
	}
	return nil
}

// -------- helpers --------

func (s *PaymentMethodService) getOrCreateCustomer(userIDStr, provider string) (*models.PaymentCustomer, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	// Try existing first
	customer, err := s.customerRepo.GetByUserAndProvider(userID, provider)
	if err == nil {
		return customer, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create on provider
	// NOTE: email/phone should come from user repo; for now pass empty.
	// Wire this once you inject UserRepository into this service.
	providerCustomerID, err := s.provider.CreateCustomer(userIDStr, "", "")
	if err != nil {
		log.Printf("provider customer creation failed for user=%s: %v", userIDStr, err)
		return nil, ErrPaymentProviderFailed
	}

	newCustomer := &models.PaymentCustomer{
		UserID:             userID,
		Provider:           provider,
		ProviderCustomerID: providerCustomerID,
	}
	if err := s.customerRepo.Create(newCustomer); err != nil {
		return nil, errors.New("failed to save payment customer")
	}
	return newCustomer, nil
}

func validateClientPaymentMethodRequest(req dto.AddClientPaymentMethodRequest) error {
	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if req.ProviderMethodID == "" {
		return fmt.Errorf("provider_method_id is required")
	}
	switch req.Type {
	case "card":
		// card must have at least last4 or brand
		if req.Last4 == nil && req.Brand == nil {
			return fmt.Errorf("card requires last4 or brand")
		}
	case "upi":
		if req.VPAMasked == nil {
			return fmt.Errorf("upi requires vpa_masked")
		}
	case "netbanking":
		// nothing extra
	default:
		return fmt.Errorf("invalid type: %s", req.Type)
	}
	return nil
}

func toClientPaymentMethodResponse(m *models.ClientPaymentMethod) *dto.ClientPaymentMethodResponse {
	resp := &dto.ClientPaymentMethodResponse{
		ID:        m.ID.String(),
		Type:      string(m.Type),
		Status:    string(m.Status),
		Provider:  m.Provider,
		IsPrimary: m.IsPrimary,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.Last4 != nil {
		resp.Last4 = *m.Last4
	}
	if m.Brand != nil {
		resp.Brand = *m.Brand
	}
	if m.ExpMonth != nil {
		resp.ExpMonth = *m.ExpMonth
	}
	if m.ExpYear != nil {
		resp.ExpYear = *m.ExpYear
	}
	if m.VPAMasked != nil {
		resp.VPAMasked = maskVPA(*m.VPAMasked)
	}
	return resp
}

// maskVPA partially hides a UPI VPA, e.g. "john@okhdfc" -> "jo**@okhdfc"
func maskVPA(vpa string) string {
	parts := strings.SplitN(vpa, "@", 2)
	if len(parts) != 2 {
		return vpa
	}
	local := parts[0]
	if len(local) <= 2 {
		return vpa
	}
	return local[:2] + strings.Repeat("*", len(local)-2) + "@" + parts[1]
}
