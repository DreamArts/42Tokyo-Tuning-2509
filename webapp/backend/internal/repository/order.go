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

// 注文を作成し、生成された注文IDを返す
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

// 複数の注文IDのステータスを一括で更新
// 主に配送ロボットが注文を引き受けた際に一括更新をするために使用
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

// 配送中(shipped_status:shipping)の注文一覧を取得
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

// 注文履歴一覧を取得
func (r *OrderRepository) ListOrders(ctx context.Context, userID int, req model.ListRequest) ([]model.Order, int, error) {
	// 基本のJOINクエリ - N+1問題を解決
	baseQuery := `
        SELECT 
            o.order_id, 
            o.product_id, 
            p.name as product_name,
            o.shipped_status, 
            o.created_at, 
            o.arrived_at
        FROM orders o 
        JOIN products p ON o.product_id = p.product_id
        WHERE o.user_id = ?`

	// 検索条件をSQLで処理
	var conditions []string
	var args []interface{}
	args = append(args, userID)

	if req.Search != "" {
		if req.Type == "prefix" {
			conditions = append(conditions, "p.name LIKE ?")
			args = append(args, req.Search+"%")
		} else {
			conditions = append(conditions, "p.name LIKE ?")
			args = append(args, "%"+req.Search+"%")
		}
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// ソート条件をSQLで処理
	orderClause := " ORDER BY "
	switch req.SortField {
	case "product_name":
		orderClause += "p.name"
	case "created_at":
		orderClause += "o.created_at"
	case "shipped_status":
		orderClause += "o.shipped_status"
	case "arrived_at":
		orderClause += "o.arrived_at"
	case "order_id":
		fallthrough
	default:
		orderClause += "o.order_id"
	}

	if strings.ToUpper(req.SortOrder) == "DESC" {
		orderClause += " DESC"
	} else {
		orderClause += " ASC"
	}

	// 件数取得用のクエリ
	countQuery := `
        SELECT COUNT(*) 
        FROM orders o 
        JOIN products p ON o.product_id = p.product_id 
        WHERE o.user_id = ?`

	countArgs := []interface{}{userID}
	if req.Search != "" {
		if req.Type == "prefix" {
			countQuery += " AND p.name LIKE ?"
			countArgs = append(countArgs, req.Search+"%")
		} else {
			countQuery += " AND p.name LIKE ?"
			countArgs = append(countArgs, "%"+req.Search+"%")
		}
	}

	// ページネーション
	dataQuery := baseQuery + orderClause + " LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, req.Offset)

	// データ構造の定義
	type orderRow struct {
		OrderID       int          `db:"order_id"`
		ProductID     int          `db:"product_id"`
		ProductName   string       `db:"product_name"`
		ShippedStatus string       `db:"shipped_status"`
		CreatedAt     sql.NullTime `db:"created_at"`
		ArrivedAt     sql.NullTime `db:"arrived_at"`
	}

	var ordersRaw []orderRow
	var total int

	// データと件数を取得
	if err := r.db.SelectContext(ctx, &ordersRaw, dataQuery, args...); err != nil {
		return nil, 0, err
	}

	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}

	// model.Orderに変換
	var orders []model.Order
	for _, o := range ordersRaw {
		orders = append(orders, model.Order{
			OrderID:       int64(o.OrderID),
			ProductID:     o.ProductID,
			ProductName:   o.ProductName, // JOINで取得済み
			ShippedStatus: o.ShippedStatus,
			CreatedAt:     o.CreatedAt.Time,
			ArrivedAt:     o.ArrivedAt,
		})
	}

	return orders, total, nil
}
