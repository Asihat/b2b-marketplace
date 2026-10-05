package services

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/jsonx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
)

// DashboardRanges are the day windows the dashboard can be asked for.
var DashboardRanges = []int{7, 30, 90}

const (
	DashboardDefaultRange = 30
	dashboardTopLimit     = 5
)

type KPI struct {
	Current  float64  `json:"current"`
	Previous float64  `json:"previous"`
	Change   *float64 `json:"change"`
}

type DayPoint struct {
	Date    string  `json:"date"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type Segment struct {
	Type    string  `json:"type"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type TopProduct struct {
	ID      *int64  `json:"id"`
	SKU     string  `json:"sku"`
	Name    string  `json:"name"`
	Qty     int     `json:"qty"`
	Revenue float64 `json:"revenue"`
}

type TopCategory struct {
	ID      *int64  `json:"id"`
	Name    string  `json:"name"`
	Qty     int     `json:"qty"`
	Revenue float64 `json:"revenue"`
}

type GatewayTotal struct {
	Gateway string  `json:"gateway"`
	Count   int     `json:"count"`
	Amount  float64 `json:"amount"`
}

type RecentOrder struct {
	ID           int64       `db:"id" json:"id"`
	Number       string      `db:"number" json:"number"`
	UserID       int64       `db:"user_id" json:"user_id"`
	Type         string      `db:"type" json:"type"`
	Status       string      `db:"status" json:"status"`
	CurrencyCode string      `db:"currency_code" json:"currency_code"`
	GrandTotal   jsonx.Money `db:"grand_total" json:"grand_total"`
	CreatedAt    jsonx.Time  `db:"created_at" json:"created_at"`
	User         any         `db:"-" json:"user"`
}

type LowStockProduct struct {
	ID          int64  `db:"id" json:"id"`
	SKU         string `db:"sku" json:"sku"`
	Name        string `db:"name" json:"name"`
	Stock       int32  `db:"stock" json:"stock"`
	MinOrderQty int32  `db:"min_order_qty" json:"min_order_qty"`
}

type Dashboard struct {
	Range          map[string]any    `json:"range"`
	BaseCurrency   map[string]string `json:"base_currency"`
	KPIs           map[string]KPI    `json:"kpis"`
	Stats          map[string]any    `json:"stats"`
	Timeseries     []DayPoint        `json:"timeseries"`
	OrdersByStatus []StatusCount     `json:"orders_by_status"`
	Segments       []Segment         `json:"segments"`
	TopProducts    []TopProduct      `json:"top_products"`
	TopCategories  []TopCategory     `json:"top_categories"`
	Gateways       []GatewayTotal    `json:"gateways"`
	Stock          map[string]int    `json:"stock"`
	RecentOrders   []RecentOrder     `json:"recent_orders"`
	LowStock       []LowStockProduct `json:"low_stock"`
}

// DashboardService builds the admin dashboard read-model. Orders and
// payments are stored in the currency the buyer checked out with, so every
// money figure is grouped by currency in SQL and folded into the base
// currency here via CurrencyService.ToBase().
type DashboardService struct {
	db       *pgxpool.Pool
	currency *CurrencyService
	now      func() time.Time
}

func NewDashboardService(pool *pgxpool.Pool, currency *CurrencyService) *DashboardService {
	return &DashboardService{db: pool, currency: currency, now: func() time.Time { return time.Now().UTC() }}
}

func validRange(days int) bool {
	for _, d := range DashboardRanges {
		if d == days {
			return true
		}
	}
	return false
}

func (s *DashboardService) Overview(ctx context.Context, days int) (*Dashboard, error) {
	if !validRange(days) {
		days = DashboardDefaultRange
	}

	now := s.now().UTC()
	to := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	previousFrom := from.AddDate(0, 0, -days)
	previousTo := from.Add(-time.Second)

	current, err := s.windowMetrics(ctx, from, to)
	if err != nil {
		return nil, err
	}
	previous, err := s.windowMetrics(ctx, previousFrom, previousTo)
	if err != nil {
		return nil, err
	}
	base := s.currency.Base(ctx)

	d := &Dashboard{
		Range:        map[string]any{"days": days, "from": from.Format("2006-01-02"), "to": to.Format("2006-01-02")},
		BaseCurrency: map[string]string{"code": base.Code, "symbol": base.Symbol},
		KPIs: map[string]KPI{
			"revenue":   kpi(current.revenue, previous.revenue),
			"orders":    kpi(float64(current.orders), float64(previous.orders)),
			"aov":       kpi(current.aov, previous.aov),
			"new_users": kpi(float64(current.newUsers), float64(previous.newUsers)),
		},
	}

	if d.Stats, err = s.stats(ctx); err != nil {
		return nil, err
	}
	if d.Timeseries, err = s.timeseries(ctx, from, to); err != nil {
		return nil, err
	}
	if d.OrdersByStatus, err = s.ordersByStatus(ctx, from, to); err != nil {
		return nil, err
	}
	if d.Segments, err = s.segments(ctx, from, to); err != nil {
		return nil, err
	}
	if d.TopProducts, err = s.topProducts(ctx, from, to); err != nil {
		return nil, err
	}
	if d.TopCategories, err = s.topCategories(ctx, from, to); err != nil {
		return nil, err
	}
	if d.Gateways, err = s.gateways(ctx, from, to); err != nil {
		return nil, err
	}
	if d.Stock, err = s.stock(ctx); err != nil {
		return nil, err
	}
	if d.RecentOrders, err = s.recentOrders(ctx); err != nil {
		return nil, err
	}
	if d.LowStock, err = s.lowStock(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func kpi(current, previous float64) KPI {
	k := KPI{Current: current, Previous: previous}
	if previous > 0 {
		k.Change = Ptr(Round((current-previous)/previous*100, 1))
	}
	return k
}

type windowMetrics struct {
	revenue    float64
	orders     int
	paidOrders int
	aov        float64
	newUsers   int
}

func (s *DashboardService) windowMetrics(ctx context.Context, from, to time.Time) (windowMetrics, error) {
	var m windowMetrics

	rows, err := s.db.Query(ctx, `
		SELECT currency_code, COUNT(*), COALESCE(SUM(grand_total), 0)::float8
		FROM orders
		WHERE created_at BETWEEN $1 AND $2 AND status = ANY($3)
		GROUP BY currency_code`, from, to, models.RevenueStatuses)
	if err != nil {
		return m, err
	}
	for rows.Next() {
		var code string
		var count int
		var total float64
		if err := rows.Scan(&code, &count, &total); err != nil {
			rows.Close()
			return m, err
		}
		m.revenue += s.currency.ToBase(ctx, total, code)
		m.paidOrders += count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return m, err
	}
	m.revenue = Round(m.revenue, 2)

	if m.orders, err = Count(ctx, s.db, `SELECT COUNT(*) FROM orders WHERE created_at BETWEEN $1 AND $2`, from, to); err != nil {
		return m, err
	}
	if m.paidOrders > 0 {
		m.aov = Round(m.revenue/float64(m.paidOrders), 2)
	}
	if m.newUsers, err = Count(ctx, s.db, `SELECT COUNT(*) FROM users WHERE created_at BETWEEN $1 AND $2`, from, to); err != nil {
		return m, err
	}
	return m, nil
}

func (s *DashboardService) stats(ctx context.Context) (map[string]any, error) {
	counts := map[string]string{
		"users":           `SELECT COUNT(*) FROM users`,
		"b2b_users":       `SELECT COUNT(*) FROM users WHERE type = 'b2b'`,
		"companies":       `SELECT COUNT(*) FROM companies`,
		"products":        `SELECT COUNT(*) FROM products`,
		"active_products": `SELECT COUNT(*) FROM products WHERE is_active = true`,
		"orders":          `SELECT COUNT(*) FROM orders`,
		"pending_orders":  `SELECT COUNT(*) FROM orders WHERE status = 'pending'`,
	}
	out := map[string]any{}
	for key, sql := range counts {
		n, err := Count(ctx, s.db, sql)
		if err != nil {
			return nil, err
		}
		out[key] = n
	}

	rows, err := s.db.Query(ctx, `
		SELECT currency_code, SUM(grand_total)::text FROM orders
		WHERE status = ANY($1) GROUP BY currency_code ORDER BY currency_code`, models.RevenueStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byCurrency := map[string]string{}
	for rows.Next() {
		var code, total string
		if err := rows.Scan(&code, &total); err != nil {
			return nil, err
		}
		byCurrency[code] = total
	}
	out["revenue_by_currency"] = byCurrency
	return out, rows.Err()
}

func (s *DashboardService) timeseries(ctx context.Context, from, to time.Time) ([]DayPoint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT TO_CHAR(created_at, 'YYYY-MM-DD') AS day, currency_code, COUNT(*),
		       SUM(CASE WHEN status = ANY($3) THEN grand_total ELSE 0 END)::float8
		FROM orders
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY day, currency_code`, from, to, models.RevenueStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := map[string]int{}
	var points []DayPoint
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		index[key] = len(points)
		points = append(points, DayPoint{Date: key})
	}

	for rows.Next() {
		var day, code string
		var count int
		var revenue float64
		if err := rows.Scan(&day, &code, &count, &revenue); err != nil {
			return nil, err
		}
		i, ok := index[day]
		if !ok {
			continue
		}
		points[i].Orders += count
		points[i].Revenue += s.currency.ToBase(ctx, revenue, code)
	}
	for i := range points {
		points[i].Revenue = Round(points[i].Revenue, 2)
	}
	return points, rows.Err()
}

func (s *DashboardService) ordersByStatus(ctx context.Context, from, to time.Time) ([]StatusCount, error) {
	rows, err := s.db.Query(ctx, `SELECT status, COUNT(*) FROM orders WHERE created_at BETWEEN $1 AND $2 GROUP BY status`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	out := make([]StatusCount, 0, len(models.OrderStatuses))
	for _, st := range models.OrderStatuses {
		out = append(out, StatusCount{Status: st, Count: counts[st]})
	}
	return out, rows.Err()
}

func (s *DashboardService) segments(ctx context.Context, from, to time.Time) ([]Segment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT type, currency_code, COUNT(*),
		       SUM(CASE WHEN status = ANY($3) THEN grand_total ELSE 0 END)::float8
		FROM orders WHERE created_at BETWEEN $1 AND $2
		GROUP BY type, currency_code`, from, to, models.RevenueStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := map[string]*Segment{}
	for _, t := range models.AccountTypes {
		segments[t] = &Segment{Type: t}
	}
	for rows.Next() {
		var typ, code string
		var count int
		var revenue float64
		if err := rows.Scan(&typ, &code, &count, &revenue); err != nil {
			return nil, err
		}
		seg, ok := segments[typ]
		if !ok {
			continue
		}
		seg.Orders += count
		seg.Revenue += s.currency.ToBase(ctx, revenue, code)
	}
	out := make([]Segment, 0, len(models.AccountTypes))
	for _, t := range models.AccountTypes {
		seg := *segments[t]
		seg.Revenue = Round(seg.Revenue, 2)
		out = append(out, seg)
	}
	return out, rows.Err()
}

func (s *DashboardService) topProducts(ctx context.Context, from, to time.Time) ([]TopProduct, error) {
	rows, err := s.db.Query(ctx, `
		SELECT order_items.product_id, order_items.sku, order_items.name, orders.currency_code,
		       SUM(order_items.quantity), SUM(order_items.line_total)::float8
		FROM order_items
		JOIN orders ON orders.id = order_items.order_id
		WHERE orders.created_at BETWEEN $1 AND $2 AND orders.status = ANY($3)
		GROUP BY order_items.product_id, order_items.sku, order_items.name, orders.currency_code`,
		from, to, models.RevenueStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byKey := map[string]*TopProduct{}
	var order []string
	for rows.Next() {
		var productID *int64
		var sku, name, code string
		var qty int
		var revenue float64
		if err := rows.Scan(&productID, &sku, &name, &code, &qty, &revenue); err != nil {
			return nil, err
		}
		// Key on the SKU: it survives a deleted product (product_id goes null).
		p, ok := byKey[sku]
		if !ok {
			p = &TopProduct{ID: productID, SKU: sku, Name: name}
			byKey[sku] = p
			order = append(order, sku)
		}
		p.Qty += qty
		p.Revenue += s.currency.ToBase(ctx, revenue, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]TopProduct, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Revenue > out[j].Revenue })
	if len(out) > dashboardTopLimit {
		out = out[:dashboardTopLimit]
	}
	for i := range out {
		out[i].Revenue = Round(out[i].Revenue, 2)
	}
	return out, nil
}

func (s *DashboardService) topCategories(ctx context.Context, from, to time.Time) ([]TopCategory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT categories.id, categories.name, orders.currency_code,
		       SUM(order_items.quantity), SUM(order_items.line_total)::float8
		FROM order_items
		JOIN orders ON orders.id = order_items.order_id
		LEFT JOIN products ON products.id = order_items.product_id
		LEFT JOIN categories ON categories.id = products.category_id
		WHERE orders.created_at BETWEEN $1 AND $2 AND orders.status = ANY($3)
		GROUP BY categories.id, categories.name, orders.currency_code`,
		from, to, models.RevenueStatuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byKey := map[int64]*TopCategory{}
	var order []int64
	for rows.Next() {
		var id *int64
		var name *string
		var code string
		var qty int
		var revenue float64
		if err := rows.Scan(&id, &name, &code, &qty, &revenue); err != nil {
			return nil, err
		}
		key := int64(0)
		if id != nil {
			key = *id
		}
		c, ok := byKey[key]
		if !ok {
			label := "Uncategorized"
			if name != nil {
				label = *name
			}
			c = &TopCategory{ID: id, Name: label}
			byKey[key] = c
			order = append(order, key)
		}
		c.Qty += qty
		c.Revenue += s.currency.ToBase(ctx, revenue, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]TopCategory, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Revenue > out[j].Revenue })
	if len(out) > dashboardTopLimit {
		out = out[:dashboardTopLimit]
	}
	for i := range out {
		out[i].Revenue = Round(out[i].Revenue, 2)
	}
	return out, nil
}

func (s *DashboardService) gateways(ctx context.Context, from, to time.Time) ([]GatewayTotal, error) {
	rows, err := s.db.Query(ctx, `
		SELECT gateway, currency_code, COUNT(*), SUM(amount)::float8
		FROM payments
		WHERE status = 'completed' AND created_at BETWEEN $1 AND $2
		GROUP BY gateway, currency_code`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byGateway := map[string]*GatewayTotal{}
	var order []string
	for rows.Next() {
		var gateway, code string
		var count int
		var total float64
		if err := rows.Scan(&gateway, &code, &count, &total); err != nil {
			return nil, err
		}
		g, ok := byGateway[gateway]
		if !ok {
			g = &GatewayTotal{Gateway: gateway}
			byGateway[gateway] = g
			order = append(order, gateway)
		}
		g.Count += count
		g.Amount += s.currency.ToBase(ctx, total, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]GatewayTotal, 0, len(order))
	for _, k := range order {
		out = append(out, *byGateway[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	for i := range out {
		out[i].Amount = Round(out[i].Amount, 2)
	}
	return out, nil
}

func (s *DashboardService) stock(ctx context.Context) (map[string]int, error) {
	queries := map[string]string{
		"out_of_stock": `SELECT COUNT(*) FROM products WHERE is_active = true AND stock = 0`,
		"low_stock":    `SELECT COUNT(*) FROM products WHERE is_active = true AND stock > 0 AND stock <= min_order_qty`,
		"inactive":     `SELECT COUNT(*) FROM products WHERE is_active = false`,
	}
	out := map[string]int{}
	for key, sql := range queries {
		n, err := Count(ctx, s.db, sql)
		if err != nil {
			return nil, err
		}
		out[key] = n
	}
	return out, nil
}

func (s *DashboardService) recentOrders(ctx context.Context) ([]RecentOrder, error) {
	orders, err := Collect[RecentOrder](s.db.Query(ctx, `
		SELECT id, number, user_id, type, status, currency_code, grand_total, created_at
		FROM orders ORDER BY created_at DESC, id DESC LIMIT 8`))
	if err != nil {
		return nil, err
	}
	users, err := LoadUserRefs(ctx, s.db, IDs(orders, func(o RecentOrder) *int64 { return &o.UserID }))
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].User = models.Rel(users[orders[i].UserID])
	}
	if orders == nil {
		orders = []RecentOrder{}
	}
	return orders, nil
}

func (s *DashboardService) lowStock(ctx context.Context) ([]LowStockProduct, error) {
	rows, err := Collect[LowStockProduct](s.db.Query(ctx, `
		SELECT id, sku, name, stock, min_order_qty FROM products
		WHERE is_active = true ORDER BY stock, id LIMIT 8`))
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []LowStockProduct{}
	}
	return rows, nil
}
