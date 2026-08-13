package services_test

import (
	"testing"
	"brevo/internal/models"
	"brevo/internal/services"
)

// MockInventoryRepository implements repositories.InventoryRepository
type MockInventoryRepository struct {
	ReserveStockFn func(itemID string, quantity int) error
	ReleaseStockFn func(itemID string, quantity int) error
}

func (m *MockInventoryRepository) GetItem(itemID string) (models.Item, error) {
	return models.Item{}, nil
}

func (m *MockInventoryRepository) ReserveStock(itemID string, quantity int) error {
	if m.ReserveStockFn != nil {
		return m.ReserveStockFn(itemID, quantity)
	}
	return nil
}

func (m *MockInventoryRepository) ReleaseStock(itemID string, quantity int) error {
	if m.ReleaseStockFn != nil {
		return m.ReleaseStockFn(itemID, quantity)
	}
	return nil
}

func (m *MockInventoryRepository) GetAvailableStock(itemID string) (int, error) {
	return 0, nil
}

func (m *MockInventoryRepository) ListItems() []models.Item {
	return nil
}

func TestInventoryService_Reserve(t *testing.T) {
	t.Run("zero quantity", func(t *testing.T) {
		repo := &MockInventoryRepository{}
		svc := services.NewInventoryService(repo)

		err := svc.Reserve("item1", 0)
		if err != services.ErrInvalidQuantity {
			t.Errorf("expected ErrInvalidQuantity for zero quantity, got %v", err)
		}
	})

	t.Run("positive quantity", func(t *testing.T) {
		repoCalled := false
		repo := &MockInventoryRepository{
			ReserveStockFn: func(itemID string, quantity int) error {
				repoCalled = true
				if itemID != "item1" || quantity != 1 {
					t.Errorf("unexpected arguments to repo")
				}
				return nil
			},
		}
		svc := services.NewInventoryService(repo)

		err := svc.Reserve("item1", 1)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !repoCalled {
			t.Errorf("expected repository to be called")
		}
	})
}
