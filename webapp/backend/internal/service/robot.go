package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
	"sort"
)

type RobotService struct {
	store *repository.Store
}

func NewRobotService(store *repository.Store) *RobotService {
	return &RobotService{store: store}
}

func (s *RobotService) GenerateDeliveryPlan(ctx context.Context, robotID string, capacity int) (*model.DeliveryPlan, error) {
	var plan model.DeliveryPlan

	err := utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.ExecTx(ctx, func(txStore *repository.Store) error {
			orders, err := txStore.OrderRepo.GetShippingOrders(ctx)
			if err != nil {
				return err
			}
			
			// Reduced logging for better performance
			if len(orders) > 50 {
				log.Printf("Robot %s: found %d orders with status 'shipping'", robotID, len(orders))
			}
			
			plan, err = selectOrdersForDeliveryOptimized(ctx, orders, robotID, capacity)
			if err != nil {
				return err
			}
			if len(plan.Orders) > 0 {
				orderIDs := make([]int64, len(plan.Orders))
				for i, order := range plan.Orders {
					orderIDs[i] = order.OrderID
				}

				if err := txStore.OrderRepo.UpdateStatuses(ctx, orderIDs, "delivering"); err != nil {
					return err
				}
				// Only log for significant batches
				if len(orderIDs) > 10 {
					log.Printf("Robot %s: selected %d orders (weight: %d, value: %d)", 
						robotID, len(orderIDs), plan.TotalWeight, plan.TotalValue)
				}
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *RobotService) UpdateOrderStatus(ctx context.Context, orderID int64, newStatus string) error {
	return utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.OrderRepo.UpdateStatuses(ctx, []int64{orderID}, newStatus)
	})
}

// Highly optimized knapsack - uses greedy for large datasets, DP for smaller ones
func selectOrdersForDeliveryOptimized(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
	n := len(orders)
	if n == 0 {
		return model.DeliveryPlan{
			RobotID:     robotID,
			TotalWeight: 0,
			TotalValue:  0,
			Orders:      []model.Order{},
		}, nil
	}

	// Use greedy for large datasets or high capacity to avoid memory/time issues
	if n > 500 || robotCapacity > 5000 {
		return selectOrdersGreedy(orders, robotID, robotCapacity), nil
	}

	// For smaller datasets, use optimized DP with early termination
	return selectOrdersDP(ctx, orders, robotID, robotCapacity)
}

// Optimized DP implementation with context checking and memory optimization
func selectOrdersDP(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
	n := len(orders)
	
	// Use 1D DP array for memory optimization
	dp := make([]int, robotCapacity+1)
	keep := make([][]bool, n+1)
	for i := range keep {
		keep[i] = make([]bool, robotCapacity+1)
	}

	// Fill DP table with optimizations
	for i := 1; i <= n; i++ {
		order := orders[i-1]
		
		// Check context cancellation every 50 iterations
		if i%50 == 0 {
			select {
			case <-ctx.Done():
				return model.DeliveryPlan{}, ctx.Err()
			default:
			}
		}

		// Process in reverse order to avoid overwriting
		for w := robotCapacity; w >= order.Weight; w-- {
			includeValue := dp[w-order.Weight] + order.Value
			if includeValue > dp[w] {
				dp[w] = includeValue
				keep[i][w] = true
			}
		}
	}

	// Backtrack to find selected orders
	selectedOrders := make([]model.Order, 0)
	totalWeight := 0
	w := robotCapacity
	
	for i := n; i > 0 && w > 0; i-- {
		if keep[i][w] {
			order := orders[i-1]
			selectedOrders = append(selectedOrders, order)
			w -= order.Weight
			totalWeight += order.Weight
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  dp[robotCapacity],
		Orders:      selectedOrders,
	}, nil
}

// Enhanced greedy approach with better sorting
func selectOrdersGreedy(orders []model.Order, robotID string, robotCapacity int) model.DeliveryPlan {
	if len(orders) == 0 {
		return model.DeliveryPlan{
			RobotID:     robotID,
			TotalWeight: 0,
			TotalValue:  0,
			Orders:      []model.Order{},
		}
	}

	// Create a copy to avoid modifying original slice
	ordersCopy := make([]model.Order, len(orders))
	copy(ordersCopy, orders)

	// Sort by value/weight ratio using Go's built-in sort (much faster)
	sort.Slice(ordersCopy, func(i, j int) bool {
		ratio1 := float64(ordersCopy[i].Value) / float64(ordersCopy[i].Weight)
		ratio2 := float64(ordersCopy[j].Value) / float64(ordersCopy[j].Weight)
		return ratio2 < ratio1 // Descending order
	})

	selectedOrders := make([]model.Order, 0)
	totalWeight := 0
	totalValue := 0

	for _, order := range ordersCopy {
		if totalWeight+order.Weight <= robotCapacity {
			selectedOrders = append(selectedOrders, order)
			totalWeight += order.Weight
			totalValue += order.Value
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  totalValue,
		Orders:      selectedOrders,
	}
}