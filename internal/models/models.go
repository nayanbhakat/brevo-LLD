package models

import "time"

// Item represents a product in the system
type Item struct {
	ID         string
	SKU        string
	Name       string
	PriceCents int64
}

type CartStatus string

const (
	Reserved          CartStatus = "RESERVED"
	PaymentProcessing CartStatus = "PAYMENT_PROCESSING"
	Completed         CartStatus = "COMPLETED"
	Expired           CartStatus = "EXPIRED"
	PaymentFailed     CartStatus = "PAYMENT_FAILED"
)

// Cart represents a temporary reservation of inventory for a user
type Cart struct {
	ID        string
	UserID    string
	ItemID    string
	Quantity  int
	ExpiresAt time.Time
	Status    CartStatus
}

// Order represents a finalized, successful purchase
type Order struct {
	ID     string
	CartID string
	UserID string
	Status string // e.g., "PAID"
}
