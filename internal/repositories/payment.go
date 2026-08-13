package repositories

import (
	"errors"
	"sync"
	"brevo/internal/models"
)

type IdempotencyStatus string

const (
	Processing IdempotencyStatus = "PROCESSING"
	Completed  IdempotencyStatus = "COMPLETED"
)

type IdempotencyRecord struct {
	Status IdempotencyStatus
	Order  *models.Order
}

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
	idempotencyCache map[string]*IdempotencyRecord
}

func NewPaymentRepository() PaymentRepository {
	return &paymentRepoImpl{
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

func (r *paymentRepoImpl) ClaimIdempotencyKey(key string) (*models.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if record, exists := r.idempotencyCache[key]; exists {
		if record.Status == Processing {
			return nil, ErrAlreadyProcessed // another thread is processing
		}
		// Return copy
		if record.Order != nil {
			orderCopy := *record.Order
			return &orderCopy, nil
		}
		return nil, nil
	}
	
	// Claim it with PROCESSING status
	r.idempotencyCache[key] = &IdempotencyRecord{
		Status: Processing,
	}
	return nil, nil
}

func (r *paymentRepoImpl) SaveIdempotencyKey(key string, order *models.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.idempotencyCache[key] = &IdempotencyRecord{
		Status: Completed,
		Order:  order,
	}
	return nil
}

func (r *paymentRepoImpl) ReleaseIdempotencyKey(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.idempotencyCache, key)
	return nil
}
