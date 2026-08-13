package services

import (
	"errors"
	"time"

	"brevo/internal/models"
	"brevo/internal/repositories"
)

var (
	ErrPaymentFailed = errors.New("payment failed mock")
	ErrUnauthorized  = errors.New("unauthorized to process payment for this cart")
)

type PaymentService interface {
	MakePayment(cartID string, idempotencyKey string, userID string) (*models.Order, error)
}

type paymentServiceImpl struct {
	repo         repositories.PaymentRepository
	orderSvc     OrderService
	inventorySvc InventoryService
}

func NewPaymentService(repo repositories.PaymentRepository, orderSvc OrderService, inventorySvc InventoryService) PaymentService {
	return &paymentServiceImpl{
		repo:         repo,
		orderSvc:     orderSvc,
		inventorySvc: inventorySvc,
	}
}

func (s *paymentServiceImpl) MakePayment(cartID string, idempotencyKey string, userID string) (*models.Order, error) {
	// 1. Atomic Idempotency Claim
	order, err := s.repo.ClaimIdempotencyKey(idempotencyKey)
	if err != nil {
		return nil, err // ErrAlreadyProcessed
	}
	if order != nil {
		return order, nil // Already processed successfully
	}

	// 2. Fetch cart and Authorize
	cart, err := s.orderSvc.GetCart(cartID)
	if err != nil {
		return nil, err
	}
	if cart.UserID != userID {
		return nil, ErrUnauthorized
	}

	// 3. Lock the cart from TTL expiry by moving it to PROCESSING
	err = s.orderSvc.UpdateCartStatus(cartID, models.Reserved, models.PaymentProcessing)
	if err != nil {
		return nil, err
	}

	// 4. Mock external payment gateway
	time.Sleep(10 * time.Millisecond) // Simulated latency
	paymentSuccess := true
	if idempotencyKey == "fail_mock" {
		paymentSuccess = false
	}

	// 5. Handle Failure
	if !paymentSuccess {
		_ = s.orderSvc.UpdateCartStatus(cartID, models.PaymentProcessing, models.PaymentFailed)
		
		// Release inventory
		_ = s.inventorySvc.Release(cart.ItemID, cart.Quantity)
		
		return nil, ErrPaymentFailed
	}

	// 6. Handle Success
	order, err = s.orderSvc.CompleteOrder(cartID, userID)
	if err != nil {
		return nil, err
	}

	// 7. Save final Order to Idempotency Cache
	_ = s.repo.SaveIdempotencyKey(idempotencyKey, order)

	return order, nil
}
