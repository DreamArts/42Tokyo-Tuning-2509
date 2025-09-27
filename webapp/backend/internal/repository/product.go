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

// 商品一覧を全件取得し、アプリケーション側でページング処理を行う
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// DB から取り出したレコードを入れておくためのスライス。
	var products []model.Product

	// 検索やソートの条件を追記していく基本の SELECT 文。
	baseQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	`
	// 検索条件の引数を格納するスライス。後で SQL に渡す。
	args := []interface{}{}

	// キーワード検索が指定されていれば name / description の部分一致に変換する。
	if req.Search != "" {
		baseQuery += " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// 並び替えのフィールドと昇降順を呼び出し側から受け取り、安定化のため product_id 
	baseQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + ", product_id ASC LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, req.Offset)

	countQuery := "SELECT COUNT(*) FROM products"

	countArgs := []interface{}{}
	
	if req.Search != "" {
		countQuery += " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}

	if err := r.db.SelectContext(ctx, &products, baseQuery, args...); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// package repository

// import (
// 	"backend/internal/model"
// 	"context"
// )

// // DB へのアクセスをまとめて面倒を見る層。UseCase からはこのパッケージを経由して DB とやり取りする。
// type ProductRepository struct {
// 	db DBTX
// }

// // NewProductRepository はリポジトリを初期化し、呼び出し側から渡された DB インターフェースを保持する。
// func NewProductRepository(db DBTX) *ProductRepository {
// 	return &ProductRepository{db: db}
// }

// // 商品一覧を全件取得し、アプリケーション側でページング処理を行う
// func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
// 	// DB から取り出したレコードを入れておくためのスライス。
// 	var products []model.Product

// 	// 検索やソートの条件を追記していく基本の SELECT 文。
// 	baseQuery := `
// 		SELECT product_id, name, value, weight, image, description
// 		FROM products
// 	`
// 	// 検索条件の引数を格納するスライス。後で SQL に渡す。
// 	args := []interface{}{}

// 	// キーワード検索が指定されていれば name / description の部分一致に変換する。
// 	if req.Search != "" {
// 		baseQuery += " WHERE (name LIKE ? OR description LIKE ?)"
// 		searchPattern := "%" + req.Search + "%"
// 		args = append(args, searchPattern, searchPattern)
// 	}

// 	// 並び替えのフィールドと昇降順を呼び出し側から受け取り、安定化のため product_id も追加する。
// 	baseQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + " , product_id ASC"

// 	// 組み立てた SQL を実行して構造体スライスにマッピングする。
// 	err := r.db.SelectContext(ctx, &products, baseQuery, args...)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// ページングに使う総件数と範囲を計算。Offset が件数を超える場合も安全に処理する。
// 	total := len(products)
// 	start := req.Offset
// 	end := req.Offset + req.PageSize
// 	if start > total {
// 		start = total
// 	}
// 	if end > total {
// 		end = total
// 	}

// 	// 範囲内のデータだけを切り出して、呼び出し元に返す。
// 	pagedProducts := products[start:end]

// 	return pagedProducts, total, nil
// }
