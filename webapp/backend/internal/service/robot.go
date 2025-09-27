package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
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
			
			log.Printf("Robot %s: found %d orders with status 'shipping'", robotID, len(orders))
			
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
				log.Printf("Robot %s: knapsack selected %d orders (total weight: %d, total value: %d)", 
					robotID, len(orderIDs), plan.TotalWeight, plan.TotalValue)
				log.Printf("Updated status to 'delivering' for %d orders", len(orderIDs))
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

// Optimized knapsack using Dynamic Programming - O(n*capacity)
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

	// Use greedy for very large datasets to avoid memory issues
	if n > 1000 || robotCapacity > 10000 {
		return selectOrdersGreedy(orders, robotID, robotCapacity), nil
	}

	// Dynamic Programming solution - O(n * capacity)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, robotCapacity+1)
	}

	// Fill DP table
	for i := 1; i <= n; i++ {
		order := orders[i-1]
		for w := 0; w <= robotCapacity; w++ {
			// Check context cancellation periodically
			if i%100 == 0 && w == 0 {
				select {
				case <-ctx.Done():
					return model.DeliveryPlan{}, ctx.Err()
				default:
				}
			}

			// Don't include current item
			dp[i][w] = dp[i-1][w]
			
			// Include current item if it fits
			if order.Weight <= w {
				includeValue := dp[i-1][w-order.Weight] + order.Value
				if includeValue > dp[i][w] {
					dp[i][w] = includeValue
				}
			}
		}
	}

	// Backtrack to find selected orders
	selectedOrders := make([]model.Order, 0)
	totalWeight := 0
	w := robotCapacity
	
	for i := n; i > 0 && w > 0; i-- {
		// Check if this item was included
		if dp[i][w] != dp[i-1][w] {
			order := orders[i-1]
			selectedOrders = append(selectedOrders, order)
			w -= order.Weight
			totalWeight += order.Weight
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  dp[n][robotCapacity],
		Orders:      selectedOrders,
	}, nil
}

// Greedy approach for very large datasets - O(n log n)
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

	// Sort by value/weight ratio (greedy heuristic)
	for i := 0; i < len(ordersCopy)-1; i++ {
		for j := i + 1; j < len(ordersCopy); j++ {
			ratio1 := float64(ordersCopy[i].Value) / float64(ordersCopy[i].Weight)
			ratio2 := float64(ordersCopy[j].Value) / float64(ordersCopy[j].Weight)
			if ratio2 > ratio1 {
				ordersCopy[i], ordersCopy[j] = ordersCopy[j], ordersCopy[i]
			}
		}
	}

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