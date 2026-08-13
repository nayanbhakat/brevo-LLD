package repositories

import (
	"errors"
	"sync"
	"time"

	"brevo/internal/models"
)

var (
	ErrCartNotFound     = errors.New("cart not found")
	ErrInvalidCartState = errors.New("cart is not in expected state")
)

// OrderRepository handles persistence of Carts and Orders.
type OrderRepository interface {
	SaveCart(cart *models.Cart) error
	GetCart(cartID string) (*models.Cart, error)
	UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error
	
	// GetExpiredReservations retrieves carts that are RESERVED but past their ExpiresAt time.
	GetExpiredReservations(now time.Time) ([]*models.Cart, error)
	
	SaveOrder(order *models.Order) error
}

type orderRepoImpl struct {
	mu     sync.RWMutex
	carts  map[string]*models.Cart
	orders map[string]*models.Order
}

func NewOrderRepository() OrderRepository {
	return &orderRepoImpl{
		carts:  make(map[string]*models.Cart),
		orders: make(map[string]*models.Order),
	}
}

func (r *orderRepoImpl) SaveCart(cart *models.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carts[cart.ID] = cart
	return nil
}

func (r *orderRepoImpl) GetCart(cartID string) (*models.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cart, exists := r.carts[cartID]
	if !exists {
		return nil, ErrCartNotFound
	}
	// Return a copy to prevent accidental outside mutation
	cartCopy := *cart
	return &cartCopy, nil
}

func (r *orderRepoImpl) UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cart, exists := r.carts[cartID]
	if !exists {
		return ErrCartNotFound
	}
	
	if cart.Status != expectedStatus {
		return ErrInvalidCartState
	}
	
	cart.Status = newStatus
	return nil
}

func (r *orderRepoImpl) GetExpiredReservations(now time.Time) ([]*models.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var expired []*models.Cart
	for _, cart := range r.carts {
		if cart.Status == models.Reserved && cart.ExpiresAt.Before(now) {
			// Copy to avoid external mutation issues
			cartCopy := *cart
			expired = append(expired, &cartCopy)
		}
	}
	return expired, nil
}

func (r *orderRepoImpl) SaveOrder(order *models.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}
