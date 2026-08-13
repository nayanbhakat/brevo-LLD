package services

import (
	"errors"
	"brevo/internal/models"
	"brevo/internal/repositories"
)

var (
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
)

type InventoryService interface {
	ListProducts() []models.Item
	GetItem(itemID string) (models.Item, error)
	Reserve(itemID string, quantity int) error
	Release(itemID string, quantity int) error
	GetAvailableStock(itemID string) (int, error)
}

type inventoryServiceImpl struct {
	repo repositories.InventoryRepository
}

func NewInventoryService(repo repositories.InventoryRepository) InventoryService {
	return &inventoryServiceImpl{repo: repo}
}

func (s *inventoryServiceImpl) ListProducts() []models.Item {
	return s.repo.ListItems()
}

func (s *inventoryServiceImpl) GetItem(itemID string) (models.Item, error) {
	return s.repo.GetItem(itemID)
}

func (s *inventoryServiceImpl) Reserve(itemID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	return s.repo.ReserveStock(itemID, quantity)
}

func (s *inventoryServiceImpl) Release(itemID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	return s.repo.ReleaseStock(itemID, quantity)
}

func (s *inventoryServiceImpl) GetAvailableStock(itemID string) (int, error) {
	return s.repo.GetAvailableStock(itemID)
}
