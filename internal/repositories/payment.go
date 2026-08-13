package repositories

import (
	"errors"
	"sync"
	"brevo/internal/models"
)

var (
	ErrAlreadyProcessed = errors.New("idempotency key already processed or in progress")
)

// PaymentRepository handles payment specific storage (like idempotency keys).
type PaymentRepository interface {
	ClaimIdempotencyKey(key string) (*models.Order, error)
	SaveIdempotencyKey(key string, order *models.Order) error
	ReleaseIdempotencyKey(key string) error
}

type paymentRepoImpl struct {
	mu               sync.Mutex
	idempotencyCache map[string]*models.Order
}

func NewPaymentRepository() PaymentRepository {
	return &paymentRepoImpl{
		idempotencyCache: make(map[string]*models.Order),
	}
}

func (r *paymentRepoImpl) ClaimIdempotencyKey(key string) (*models.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if order, exists := r.idempotencyCache[key]; exists {
		if order == nil {
			return nil, ErrAlreadyProcessed // another thread is processing
		}
		// Return copy
		orderCopy := *order
		return &orderCopy, nil
	}
	
	// Claim it with nil marker to signify processing
	r.idempotencyCache[key] = nil
	return nil, nil
}

func (r *paymentRepoImpl) SaveIdempotencyKey(key string, order *models.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.idempotencyCache[key] = order
	return nil
}

func (r *paymentRepoImpl) ReleaseIdempotencyKey(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.idempotencyCache, key)
	return nil
}
