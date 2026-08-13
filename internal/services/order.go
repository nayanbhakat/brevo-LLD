package services

import (
	"errors"
	"time"

	"brevo/internal/models"
	"brevo/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrCartNotReserved = errors.New("cart is not in RESERVED state")
)

type OrderService interface {
	AddToCart(userID string, itemID string, quantity int) (*models.Cart, error)
	GetCart(cartID string) (*models.Cart, error)
	UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error
	CompleteOrder(cartID string, userID string) (*models.Order, error)
	
	StartTTLWorker(interval time.Duration)
	StopTTLWorker()
}

type orderServiceImpl struct {
	repo         repositories.OrderRepository
	inventorySvc InventoryService
	ttlDuration  time.Duration
	stopWorker   chan struct{}
}

func NewOrderService(repo repositories.OrderRepository, inventorySvc InventoryService, ttlDuration time.Duration) OrderService {
	return &orderServiceImpl{
		repo:         repo,
		inventorySvc: inventorySvc,
		ttlDuration:  ttlDuration,
		stopWorker:   make(chan struct{}),
	}
}

func (s *orderServiceImpl) AddToCart(userID string, itemID string, quantity int) (*models.Cart, error) {
	// 1. Reserve inventory
	err := s.inventorySvc.Reserve(itemID, quantity)
	if err != nil {
		return nil, err
	}

	// 2. Create Cart
	cartID := uuid.New().String()
	cart := &models.Cart{
		ID:        cartID,
		UserID:    userID,
		ItemID:    itemID,
		Quantity:  quantity,
		ExpiresAt: time.Now().Add(s.ttlDuration),
		Status:    models.Reserved,
	}

	// 3. Save cart to repo
	if err := s.repo.SaveCart(cart); err != nil {
		// Rollback inventory on DB failure
		_ = s.inventorySvc.Release(itemID, quantity)
		return nil, err
	}

	return cart, nil
}

func (s *orderServiceImpl) GetCart(cartID string) (*models.Cart, error) {
	return s.repo.GetCart(cartID)
}

func (s *orderServiceImpl) UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
	return s.repo.UpdateCartStatus(cartID, expectedStatus, newStatus)
}

func (s *orderServiceImpl) CompleteOrder(cartID string, userID string) (*models.Order, error) {
	// Mark cart as completed
	err := s.repo.UpdateCartStatus(cartID, models.PaymentProcessing, models.Completed)
	if err != nil {
		return nil, err
	}

	order := &models.Order{
		ID:     uuid.New().String(),
		CartID: cartID,
		UserID: userID,
		Status: "PAID",
	}

	err = s.repo.SaveOrder(order)
	return order, err
}

func (s *orderServiceImpl) StartTTLWorker(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopWorker:
				return
			case <-ticker.C:
				s.expireCarts()
			}
		}
	}()
}

func (s *orderServiceImpl) StopTTLWorker() {
	close(s.stopWorker)
}

func (s *orderServiceImpl) expireCarts() {
	now := time.Now()
	expiredCarts, err := s.repo.GetExpiredReservations(now)
	if err != nil {
		return
	}

	for _, cart := range expiredCarts {
		// Attempt to update status to EXPIRED.
		// Use expected status RESERVED to prevent race condition if payment just picked it up.
		err := s.UpdateCartStatus(cart.ID, models.Reserved, models.Expired)
		if err == nil {
			// Successfully expired, release inventory
			_ = s.inventorySvc.Release(cart.ItemID, cart.Quantity)
		}
	}
}
