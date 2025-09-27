package repository

import (
	"backend/internal/model"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type OrderRepository struct {
	db DBTX
}

func NewOrderRepository(db DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create order and return generated order ID
func (r *OrderRepository) Create(ctx context.Context, order *model.Order) (string, error) {
	query := `INSERT INTO orders (user_id, product_id, shipped_status, created_at) VALUES (?, ?, 'shipping', NOW())`
	result, err := r.db.ExecContext(ctx, query, order.UserID, order.ProductID)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// Batch update status for multiple order IDs
// Used when delivery robot takes multiple orders at once
func (r *OrderRepository) UpdateStatuses(ctx context.Context, orderIDs []int64, newStatus string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In("UPDATE orders SET shipped_status = ? WHERE order_id IN (?)", newStatus, orderIDs)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// Get list of orders with shipping status
func (r *OrderRepository) GetShippingOrders(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	query := `
        SELECT
            o.order_id,
            p.weight,
            p.value
        FROM orders o
        JOIN products p ON o.product_id = p.product_id
        WHERE o.shipped_status = 'shipping'
    `
	err := r.db.SelectContext(ctx, &orders, query)
	return orders, err
}

// FIXED: Get order history list - Solved N+1 problem, fetch product names with JOIN in single query
func (r *OrderRepository) ListOrders(ctx context.Context, userID int, req model.ListRequest) ([]model.Order, int, error) {
	// Build search conditions
	whereClause := "WHERE o.user_id = ?"
	args := []interface{}{userID}
	
	if req.Search != "" {
		if req.Type == "prefix" {
			whereClause += " AND p.name LIKE ?"
			args = append(args, req.Search+"%")
		} else {
			whereClause += " AND p.name LIKE ?"
			args = append(args, "%"+req.Search+"%")
		}
	}

	// Build ORDER BY clause - Sort at SQL level
	orderClause := "ORDER BY "
	switch req.SortField {
	case "product_name":
		orderClause += "p.name " + strings.ToUpper(req.SortOrder)
	case "created_at":
		orderClause += "o.created_at " + strings.ToUpper(req.SortOrder)
	case "shipped_status":
		orderClause += "o.shipped_status " + strings.ToUpper(req.SortOrder)
	case "arrived_at":
		orderClause += "o.arrived_at " + strings.ToUpper(req.SortOrder)
	case "order_id":
		fallthrough
	default:
		orderClause += "o.order_id " + strings.ToUpper(req.SortOrder)
	}
	
	// Get total count
	countQuery := `
		SELECT COUNT(*) 
		FROM orders o
		JOIN products p ON o.product_id = p.product_id
		` + whereClause
	
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Main query - Get product names with JOIN in single query, sort and paginate at SQL level
	query := `
		SELECT 
			o.order_id,
			o.product_id,
			p.name as product_name,
			o.shipped_status,
			o.created_at,
			o.arrived_at
		FROM orders o
		JOIN products p ON o.product_id = p.product_id
		` + whereClause + `
		` + orderClause + `
		LIMIT ? OFFSET ?
	`
	
	args = append(args, req.PageSize, req.Offset)
	
	type orderRow struct {
		OrderID       int          `db:"order_id"`
		ProductID     int          `db:"product_id"`
		ProductName   string       `db:"product_name"`
		ShippedStatus string       `db:"shipped_status"`
		CreatedAt     sql.NullTime `db:"created_at"`
		ArrivedAt     sql.NullTime `db:"arrived_at"`
	}
	
	var ordersRaw []orderRow
	if err := r.db.SelectContext(ctx, &ordersRaw, query, args...); err != nil {
		return nil, 0, err
	}

	var orders []model.Order
	for _, o := range ordersRaw {
		orders = append(orders, model.Order{
			OrderID:       int64(o.OrderID),
			ProductID:     o.ProductID,
			ProductName:   o.ProductName,
			ShippedStatus: o.ShippedStatus,
			CreatedAt:     o.CreatedAt.Time,
			ArrivedAt:     o.ArrivedAt,
		})
	}

	return orders, total, nil
}