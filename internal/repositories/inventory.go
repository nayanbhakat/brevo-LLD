package repositories

import (
	"errors"
	"sync"

	"brevo/internal/models"
)

var (
	ErrSoldOut      = errors.New("item sold out")
	ErrItemNotFound = errors.New("item not found")
)

type InventoryRepository interface {
	GetItem(itemID string) (models.Item, error)
	ReserveStock(itemID string, quantity int) error
	ReleaseStock(itemID string, quantity int) error
	GetAvailableStock(itemID string) (int, error)
	ListItems() []models.Item
}

type inventoryRepoImpl struct {
	mu            sync.Mutex
	items         map[string]models.Item
	availableStock map[string]int
}

func NewInventoryRepository(initialItems []models.Item, initialStock map[string]int) InventoryRepository {
	itemsMap := make(map[string]models.Item)
	stockMap := make(map[string]int)

	for _, item := range initialItems {
		itemsMap[item.ID] = item
	}
	for itemID, qty := range initialStock {
		stockMap[itemID] = qty
	}

	return &inventoryRepoImpl{
		items:         itemsMap,
		availableStock: stockMap,
	}
}

func (r *inventoryRepoImpl) GetItem(itemID string) (models.Item, error) {
	// Simple map read, could use RWMutex if items were dynamically added
	item, ok := r.items[itemID]
	if !ok {
		return models.Item{}, ErrItemNotFound
	}
	return item, nil
}

func (r *inventoryRepoImpl) ListItems() []models.Item {
	var result []models.Item
	for _, item := range r.items {
		result = append(result, item)
	}
	return result
}

// ReserveStock atomically checks and deducts stock.
func (r *inventoryRepoImpl) ReserveStock(itemID string, quantity int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stock, exists := r.availableStock[itemID]
	if !exists {
		return ErrItemNotFound
	}

	if stock < quantity {
		return ErrSoldOut
	}

	r.availableStock[itemID] = stock - quantity
	return nil
}

func (r *inventoryRepoImpl) ReleaseStock(itemID string, quantity int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stock, exists := r.availableStock[itemID]
	if !exists {
		return ErrItemNotFound
	}

	r.availableStock[itemID] = stock + quantity
	return nil
}

func (r *inventoryRepoImpl) GetAvailableStock(itemID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stock, exists := r.availableStock[itemID]
	if !exists {
		return 0, ErrItemNotFound
	}
	return stock, nil
}
