package services_test

import (
	"testing"
	"time"

	"brevo/internal/models"
	"brevo/internal/services"
)

type MockPaymentRepository struct {
	ClaimIdempotencyKeyFn   func(key string) (*models.Order, error)
	SaveIdempotencyKeyFn    func(key string, order *models.Order) error
	ReleaseIdempotencyKeyFn func(key string) error
}

func (m *MockPaymentRepository) ClaimIdempotencyKey(key string) (*models.Order, error) {
	if m.ClaimIdempotencyKeyFn != nil {
		return m.ClaimIdempotencyKeyFn(key)
	}
	return nil, nil
}
func (m *MockPaymentRepository) SaveIdempotencyKey(key string, order *models.Order) error {
	if m.SaveIdempotencyKeyFn != nil {
		return m.SaveIdempotencyKeyFn(key, order)
	}
	return nil
}
func (m *MockPaymentRepository) ReleaseIdempotencyKey(key string) error {
	if m.ReleaseIdempotencyKeyFn != nil {
		return m.ReleaseIdempotencyKeyFn(key)
	}
	return nil
}

type MockOrderService struct {
	UpdateCartStatusFn func(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error
	CompleteOrderFn    func(cartID string, userID string) (*models.Order, error)
	GetCartFn          func(cartID string) (*models.Cart, error)
}

func (m *MockOrderService) AddToCart(userID string, itemID string, quantity int) (*models.Cart, error) {
	return nil, nil
}
func (m *MockOrderService) GetCart(cartID string) (*models.Cart, error) {
	if m.GetCartFn != nil {
		return m.GetCartFn(cartID)
	}
	return &models.Cart{ID: cartID, ItemID: "item1", Quantity: 1, UserID: "user1"}, nil
}
func (m *MockOrderService) UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
	if m.UpdateCartStatusFn != nil {
		return m.UpdateCartStatusFn(cartID, expectedStatus, newStatus)
	}
	return nil
}
func (m *MockOrderService) CompleteOrder(cartID string, userID string) (*models.Order, error) {
	if m.CompleteOrderFn != nil {
		return m.CompleteOrderFn(cartID, userID)
	}
	return &models.Order{ID: "order1", CartID: cartID, UserID: userID, Status: "PAID"}, nil
}
func (m *MockOrderService) StartTTLWorker(interval time.Duration) {}
func (m *MockOrderService) StopTTLWorker() {}

func TestPaymentService_MakePayment(t *testing.T) {
	t.Run("idempotency check returns existing order", func(t *testing.T) {
		existingOrder := &models.Order{ID: "existing"}
		payRepo := &MockPaymentRepository{
			ClaimIdempotencyKeyFn: func(key string) (*models.Order, error) {
				return existingOrder, nil
			},
		}
		
		svc := services.NewPaymentService(payRepo, &MockOrderService{}, &MockInventoryService{})
		
		order, err := svc.MakePayment("cart1", "key1", "user1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order.ID != "existing" {
			t.Fatalf("expected existing order, got %+v", order)
		}
	})

	t.Run("successful payment creates order", func(t *testing.T) {
		payRepo := &MockPaymentRepository{
			ClaimIdempotencyKeyFn: func(key string) (*models.Order, error) {
				return nil, nil // Not seen before
			},
		}
		orderSvc := &MockOrderService{
			UpdateCartStatusFn: func(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
				if newStatus != models.PaymentProcessing {
					t.Errorf("expected state to transition to PAYMENT_PROCESSING, got %s", newStatus)
				}
				return nil
			},
			CompleteOrderFn: func(cartID string, userID string) (*models.Order, error) {
				return &models.Order{ID: "new_order", CartID: cartID}, nil
			},
		}
		
		svc := services.NewPaymentService(payRepo, orderSvc, &MockInventoryService{})
		
		order, err := svc.MakePayment("cart1", "key1", "user1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if order.ID != "new_order" {
			t.Fatalf("expected new order, got %+v", order)
		}
	})

	t.Run("failed mock payment releases inventory", func(t *testing.T) {
		payRepo := &MockPaymentRepository{}
		
		var statusUpdatedToFailed bool
		orderSvc := &MockOrderService{
			UpdateCartStatusFn: func(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
				if newStatus == models.PaymentFailed {
					statusUpdatedToFailed = true
				}
				return nil
			},
		}

		var inventoryReleased bool
		invSvc := &MockInventoryService{
			ReleaseFn: func(itemID string, quantity int) error {
				inventoryReleased = true
				return nil
			},
		}

		svc := services.NewPaymentService(payRepo, orderSvc, invSvc)
		
		_, err := svc.MakePayment("cart1", "fail_mock", "user1")
		if err != services.ErrPaymentFailed {
			t.Fatalf("expected payment failed error, got %v", err)
		}
		
		if !statusUpdatedToFailed {
			t.Errorf("expected cart status to be updated to PAYMENT_FAILED")
		}
		if !inventoryReleased {
			t.Errorf("expected inventory to be released on payment failure")
		}
	})
}
