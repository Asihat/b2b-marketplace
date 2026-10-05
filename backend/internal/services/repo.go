package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
)

// Collect scans every row into T by column name.
func Collect[T any](rows pgx.Rows, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByNameLax[T])
}

// One scans the first row into T, returning pgx.ErrNoRows when there is none.
func One[T any](rows pgx.Rows, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	t, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Count runs a COUNT(*) query.
func Count(ctx context.Context, q db.Querier, sql string, args ...any) (int, error) {
	var n int
	err := q.QueryRow(ctx, sql, args...).Scan(&n)
	return n, err
}

// Exists reports whether a row matching the condition exists.
func Exists(ctx context.Context, q db.Querier, table, where string, args ...any) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE %s)", table, where), args...).Scan(&ok)
	return ok, err
}

// ---- Single-row finders (ErrNoRows -> 404 upstream) -------------------------

func FindUser(ctx context.Context, q db.Querier, id int64) (*models.User, error) {
	return One[models.User](q.Query(ctx, "SELECT "+db.Columns(models.UserColumns, "")+" FROM users WHERE id = $1", id))
}

func FindUserByEmail(ctx context.Context, q db.Querier, email string) (*models.User, error) {
	return One[models.User](q.Query(ctx, "SELECT "+db.Columns(models.UserColumns, "")+" FROM users WHERE email = $1", email))
}

func FindCompany(ctx context.Context, q db.Querier, id int64) (*models.Company, error) {
	return One[models.Company](q.Query(ctx, "SELECT "+db.Columns(models.CompanyColumns, "")+" FROM companies WHERE id = $1", id))
}

func FindProduct(ctx context.Context, q db.Querier, id int64) (*models.Product, error) {
	return One[models.Product](q.Query(ctx, "SELECT "+db.Columns(models.ProductColumns, "")+" FROM products WHERE id = $1", id))
}

func FindCategory(ctx context.Context, q db.Querier, id int64) (*models.Category, error) {
	return One[models.Category](q.Query(ctx, "SELECT "+db.Columns(models.CategoryColumns, "")+" FROM categories WHERE id = $1", id))
}

func FindCurrency(ctx context.Context, q db.Querier, id int64) (*models.Currency, error) {
	return One[models.Currency](q.Query(ctx, "SELECT "+db.Columns(models.CurrencyColumns, "")+" FROM currencies WHERE id = $1", id))
}

func FindOrder(ctx context.Context, q db.Querier, id int64) (*models.Order, error) {
	return One[models.Order](q.Query(ctx, "SELECT "+db.Columns(models.OrderColumns, "")+" FROM orders WHERE id = $1", id))
}

func FindPayment(ctx context.Context, q db.Querier, id int64) (*models.Payment, error) {
	return One[models.Payment](q.Query(ctx, "SELECT "+db.Columns(models.PaymentColumns, "")+" FROM payments WHERE id = $1", id))
}

func FindProductPrice(ctx context.Context, q db.Querier, id int64) (*models.ProductPrice, error) {
	return One[models.ProductPrice](q.Query(ctx, "SELECT "+db.Columns(models.ProductPriceColumns, "")+" FROM product_prices WHERE id = $1", id))
}

// ---- Relation loaders (one query per relation, grouped by parent id) ---------

func LoadImages(ctx context.Context, q db.Querier, productIDs []int64) (map[int64][]models.ProductImage, error) {
	out := map[int64][]models.ProductImage{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := Collect[models.ProductImage](q.Query(ctx,
		"SELECT "+db.Columns(models.ProductImageColumns, "")+" FROM product_images WHERE product_id = ANY($1) ORDER BY position, id", productIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ProductID] = append(out[r.ProductID], r)
	}
	return out, nil
}

func LoadTranslations(ctx context.Context, q db.Querier, productIDs []int64) (map[int64][]models.ProductTranslation, error) {
	out := map[int64][]models.ProductTranslation{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := Collect[models.ProductTranslation](q.Query(ctx,
		"SELECT "+db.Columns(models.ProductTranslationColumns, "")+" FROM product_translations WHERE product_id = ANY($1) ORDER BY id", productIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ProductID] = append(out[r.ProductID], r)
	}
	return out, nil
}

func LoadPrices(ctx context.Context, q db.Querier, productIDs []int64) (map[int64][]models.ProductPrice, error) {
	out := map[int64][]models.ProductPrice{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := Collect[models.ProductPrice](q.Query(ctx,
		"SELECT "+db.Columns(models.ProductPriceColumns, "")+" FROM product_prices WHERE product_id = ANY($1) ORDER BY currency_code, min_qty", productIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ProductID] = append(out[r.ProductID], r)
	}
	return out, nil
}

func LoadCompanyBriefs(ctx context.Context, q db.Querier, ids []int64) (map[int64]*models.CompanyBrief, error) {
	out := map[int64]*models.CompanyBrief{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := Collect[models.CompanyBrief](q.Query(ctx, "SELECT id, name, slug, is_verified FROM companies WHERE id = ANY($1)", ids))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

// LoadNamedRefs loads {id, name} projections from any table with those columns.
func LoadNamedRefs(ctx context.Context, q db.Querier, table string, ids []int64) (map[int64]*models.NamedRef, error) {
	out := map[int64]*models.NamedRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := Collect[models.NamedRef](q.Query(ctx, "SELECT id, name FROM "+table+" WHERE id = ANY($1)", ids))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func LoadUserRefs(ctx context.Context, q db.Querier, ids []int64) (map[int64]*models.UserRef, error) {
	out := map[int64]*models.UserRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := Collect[models.UserRef](q.Query(ctx, "SELECT id, name, email FROM users WHERE id = ANY($1)", ids))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func LoadOrderItems(ctx context.Context, q db.Querier, orderIDs []int64) (map[int64][]models.OrderItem, error) {
	out := map[int64][]models.OrderItem{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := Collect[models.OrderItem](q.Query(ctx,
		"SELECT "+db.Columns(models.OrderItemColumns, "")+" FROM order_items WHERE order_id = ANY($1) ORDER BY id", orderIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.OrderID] = append(out[r.OrderID], r)
	}
	return out, nil
}

func LoadPayments(ctx context.Context, q db.Querier, orderIDs []int64) (map[int64][]models.Payment, error) {
	out := map[int64][]models.Payment{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := Collect[models.Payment](q.Query(ctx,
		"SELECT "+db.Columns(models.PaymentColumns, "")+" FROM payments WHERE order_id = ANY($1) ORDER BY id", orderIDs))
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.OrderID] = append(out[r.OrderID], r)
	}
	return out, nil
}

// LatestPayments mirrors Order::payment() (hasOne latestOfMany).
func LatestPayments(ctx context.Context, q db.Querier, orderIDs []int64) (map[int64]*models.Payment, error) {
	all, err := LoadPayments(ctx, q, orderIDs)
	if err != nil {
		return nil, err
	}
	out := map[int64]*models.Payment{}
	for id, list := range all {
		last := list[len(list)-1]
		out[id] = &last
	}
	return out, nil
}

// IDs collects a column of ids from a slice, skipping nils and duplicates.
func IDs[T any](items []T, pick func(T) *int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(items))
	for _, it := range items {
		id := pick(it)
		if id == nil || seen[*id] {
			continue
		}
		seen[*id] = true
		out = append(out, *id)
	}
	return out
}

func Ptr[T any](v T) *T { return &v }

// Placeholders renders "$start, $start+1, ..." for n values.
func Placeholders(start, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", start+i)
	}
	return strings.Join(parts, ", ")
}
