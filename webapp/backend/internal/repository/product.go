package repository

import (
	"backend/internal/model"
	"context"
)

// DB へのアクセスをまとめて面倒を見る層。UseCase からはこのパッケージを経由して DB とやり取りする。
type ProductRepository struct {
	db DBTX
}

// NewProductRepository はリポジトリを初期化し、呼び出し側から渡された DB インターフェースを保持する。
func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

// 条件やページ番号を受け取り、商品一覧と件数を返す
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// 取り出した商品を一時的に入れておく箱。
	var products []model.Product

	// まずは表から読みたい列だけ決めた基本の取り出し。
	baseQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	`
	// 後でクエリに渡す値を順番に詰めるためのリスト。
	args := []interface{}{}

	// キーワードが来たら名前と説明文にその文字が含まれる行だけに絞る。
	if req.Search != "" {
		baseQuery += " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// 並び順と欲しい件数を指定し、何行飛ばして何件取るかを決める。

	baseQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + ", product_id ASC LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, req.Offset)

	// ページ総数を計算するための件数カウント文を用意する。
	countQuery := "SELECT COUNT(*) FROM products"

	countArgs := []interface{}{}
	// 件数を数えるときも同じ絞り込み条件を使う。
	if req.Search != "" {
		countQuery += " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}

	// クエリを実行して商品一覧を受け取る。
	if err := r.db.SelectContext(ctx, &products, baseQuery, args...); err != nil {
		return nil, 0, err
	}

	var total int
	// 同じ条件で全体件数を取得する。
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}

	// 商品一覧と総件数をそのまま返す。
	return products, total, nil
}
