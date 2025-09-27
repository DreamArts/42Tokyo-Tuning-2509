package repository

import (
	"backend/internal/model"
	"context"
	"strings"
)

type ProductRepository struct {
	db DBTX
}

func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

// FIXED: Get product list - SQL-level sorting & LIMIT, separate COUNT for efficiency
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// Build search conditions
	whereClause := ""
	args := []interface{}{}

	if req.Search != "" {
		whereClause = "WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Get total count first
	countQuery := `SELECT COUNT(*) FROM products ` + whereClause
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Main query - SQL-level sorting & LIMIT
	mainQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
		` + whereClause + `
		ORDER BY ` + req.SortField + ` ` + strings.ToUpper(req.SortOrder) + `, product_id ASC
		LIMIT ? OFFSET ?
	`
	
	// Add LIMIT and OFFSET to arguments
	args = append(args, req.PageSize, req.Offset)

	var products []model.Product
	err := r.db.SelectContext(ctx, &products, mainQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}