package services_test

import (
	"errors"
	"testing"
	"time"

	"brevo/internal/models"
	"brevo/internal/services"
)

type MockOrderRepository struct {
	SaveCartFn               func(cart *models.Cart) error
	GetCartFn                func(cartID string) (*models.Cart, error)
	UpdateCartStatusFn       func(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error
	GetExpiredReservationsFn func(now time.Time) ([]*models.Cart, error)
	SaveOrderFn              func(order *models.Order) error
}

func (m *MockOrderRepository) SaveCart(cart *models.Cart) error {
	if m.SaveCartFn != nil {
		return m.SaveCartFn(cart)
	}
	return nil
}
func (m *MockOrderRepository) GetCart(cartID string) (*models.Cart, error) {
	if m.GetCartFn != nil {
		return m.GetCartFn(cartID)
	}
	return nil, errors.New("not implemented")
}
func (m *MockOrderRepository) UpdateCartStatus(cartID string, expectedStatus models.CartStatus, newStatus models.CartStatus) error {
	if m.UpdateCartStatusFn != nil {
		return m.UpdateCartStatusFn(cartID, expectedStatus, newStatus)
	}
	return nil
}
func (m *MockOrderRepository) GetExpiredReservations(now time.Time) ([]*models.Cart, error) {
	if m.GetExpiredReservationsFn != nil {
		return m.GetExpiredReservationsFn(now)
	}
	return nil, nil
}
func (m *MockOrderRepository) SaveOrder(order *models.Order) error {
	if m.SaveOrderFn != nil {
		return m.SaveOrderFn(order)
	}
	return nil
}

// MockInventoryService implements services.InventoryService
type MockInventoryService struct {
	ReserveFn func(itemID string, quantity int) error
	ReleaseFn func(itemID string, quantity int) error
}

func (m *MockInventoryService) ListProducts() []models.Item { return nil }
func (m *MockInventoryService) GetItem(itemID string) (models.Item, error) { return models.Item{}, nil }
func (m *MockInventoryService) Reserve(itemID string, quantity int) error {
	if m.ReserveFn != nil {
		return m.ReserveFn(itemID, quantity)
	}
	return nil
}
func (m *MockInventoryService) Release(itemID string, quantity int) error {
	if m.ReleaseFn != nil {
		return m.ReleaseFn(itemID, quantity)
	}
	return nil
}
func (m *MockInventoryService) GetAvailableStock(itemID string) (int, error) { return 0, nil }

func TestOrderService_AddToCart(t *testing.T) {
	t.Run("successful add to cart", func(t *testing.T) {
		invSvc := &MockInventoryService{
			ReserveFn: func(itemID string, quantity int) error {
				return nil
			},
		}
		repo := &MockOrderRepository{
			SaveCartFn: func(cart *models.Cart) error {
				return nil
			},
		}
		
		svc := services.NewOrderService(repo, invSvc, 5*time.Minute)
		cart, err := svc.AddToCart("user1", "item1", 1)
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cart == nil {
			t.Fatalf("expected cart, got nil")
		}
		if cart.Status != models.Reserved {
			t.Errorf("expected status RESERVED, got %s", cart.Status)
		}
		if cart.UserID != "user1" || cart.ItemID != "item1" || cart.Quantity != 1 {
			t.Errorf("cart data mismatch: %+v", cart)
		}
	})

	t.Run("inventory reserve fails", func(t *testing.T) {
		expectedErr := errors.New("sold out")
		invSvc := &MockInventoryService{
			ReserveFn: func(itemID string, quantity int) error {
				return expectedErr
			},
		}
		repo := &MockOrderRepository{}
		
		svc := services.NewOrderService(repo, invSvc, 5*time.Minute)
		_, err := svc.AddToCart("user1", "item1", 1)
		
		if err != expectedErr {
			t.Fatalf("expected sold out error, got %v", err)
		}
	})
}
