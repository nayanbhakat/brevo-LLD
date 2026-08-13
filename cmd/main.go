package main

import (
	"fmt"
	"sync"
	"time"

	"brevo/internal/models"
	"brevo/internal/repositories"
	"brevo/internal/services"
)

func main() {
	// 1. Initialize data
	initialItems := []models.Item{
		{
			ID:         "item_flash_1",
			SKU:        "LTD-EDITION-001",
			Name:       "Limited Edition Sneakers",
			PriceCents: 19999, // $199.99
		},
	}
	initialStock := map[string]int{
		"item_flash_1": 500,
	}

	// 2. Initialize Repositories
	invRepo := repositories.NewInventoryRepository(initialItems, initialStock)
	orderRepo := repositories.NewOrderRepository()
	payRepo := repositories.NewPaymentRepository()

	// 3. Initialize Services
	invSvc := services.NewInventoryService(invRepo)
	orderSvc := services.NewOrderService(orderRepo, invSvc, 5*time.Minute)
	paySvc := services.NewPaymentService(payRepo, orderSvc, invSvc)

	// 4. Start Background Workers
	orderSvc.StartTTLWorker(1 * time.Second)
	defer orderSvc.StopTTLWorker()

	fmt.Println("Starting Flash Sale Simulation...")
	
	startStock, _ := invSvc.GetAvailableStock("item_flash_1")
	fmt.Printf("Initial Stock: %d\n", startStock)

	// 5. Simulate 50,000 concurrent users
	var wg sync.WaitGroup
	numUsers := 50000

	var successfulCarts int
	var mu sync.Mutex

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			uID := fmt.Sprintf("user_%d", userID)
			
			// Try to add to cart
			cart, err := orderSvc.AddToCart(uID, "item_flash_1", 1)
			if err != nil {
				// Most will fail with ErrSoldOut
				return
			}
			
			mu.Lock()
			successfulCarts++
			mu.Unlock()

			// Try to pay immediately
			idempKey := fmt.Sprintf("pay_key_%s", cart.ID)
			_, _ = paySvc.MakePayment(cart.ID, idempKey, uID)
		}(i)
	}

	// Wait for all requests to finish
	wg.Wait()

	fmt.Println("---------------------------------------------------")
	fmt.Println("Simulation Complete!")
	fmt.Printf("Total Concurrent Requests: %d\n", numUsers)
	fmt.Printf("Successful Cart Reservations: %d\n", successfulCarts)
	
	finalStock, _ := invSvc.GetAvailableStock("item_flash_1")
	fmt.Printf("Final Available Stock: %d\n", finalStock)

	if finalStock == 0 && successfulCarts == 500 {
		fmt.Println("SUCCESS: System successfully prevented overselling!")
	} else {
		fmt.Println("ERROR: System failed to maintain consistency.")
	}
}
