package server_test

// Integration tests against a real PostgreSQL database. They run only when
// TEST_DATABASE_URL is set, e.g.
//
//	TEST_DATABASE_URL=postgres://b2b:secret@127.0.0.1:5432/b2b_test?sslmode=disable go test ./...
//
// Every test starts from truncated tables.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/config"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/server"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

type env struct {
	t    *testing.T
	ctx  context.Context
	pool *pgxpool.Pool
	app  *services.App
	srv  *httptest.Server
}

func setup(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `TRUNCATE users, companies, personal_access_tokens, currencies, languages, categories, products,
		product_translations, product_prices, product_analogs, product_images, orders, order_items, payments, app_settings
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Load()
	cfg.DatabaseURL = url
	cfg.AppURL = "http://api.test"
	cfg.StoragePath = t.TempDir()
	cfg.BcryptCost = 4
	app := services.NewApp(cfg, pool)
	srv := httptest.NewServer(server.NewRouter(app))

	t.Cleanup(func() {
		srv.Close()
		pool.Close()
	})
	return &env{t: t, ctx: ctx, pool: pool, app: app, srv: srv}
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(e.ctx, sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *env) currencies() {
	e.exec(`INSERT INTO currencies (code, name, symbol, exchange_rate, is_base, is_active) VALUES
		('USD', 'US Dollar', '$', 1.0, true, true), ('EUR', 'Euro', '€', 0.5, false, true)`)
	e.app.FlushCaches()
}

func (e *env) user(email, role, typ string) int64 {
	e.t.Helper()
	hash, _ := auth.HashPassword("password", 4)
	var id int64
	err := e.pool.QueryRow(e.ctx, `INSERT INTO users (name, email, password, type, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now()) RETURNING id`, email, email, hash, typ, role).Scan(&id)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *env) token(userID int64) string {
	e.t.Helper()
	tok, err := auth.IssueToken(e.ctx, e.pool, userID, "test")
	if err != nil {
		e.t.Fatal(err)
	}
	return tok
}

func (e *env) order(buyer int64, currency string, total float64, status string, daysAgo int) int64 {
	e.t.Helper()
	placed := time.Now().UTC().AddDate(0, 0, -daysAgo)
	placed = time.Date(placed.Year(), placed.Month(), placed.Day(), 12, 0, 0, 0, time.UTC)
	var id int64
	err := e.pool.QueryRow(e.ctx, `INSERT INTO orders (number, user_id, type, status, currency_code, subtotal, grand_total, created_at, updated_at)
		VALUES ($1, $2, 'b2c', $3, $4, $5, $5, $6, $6) RETURNING id`,
		"ORD-"+fmt.Sprintf("%08d", time.Now().UnixNano()%1e8), buyer, status, currency, total, placed).Scan(&id)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

type resp struct {
	status int
	body   map[string]any
	raw    string
}

func (e *env) call(method, path, token string, body any, contentType string) resp {
	e.t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		buf, _ := json.Marshal(b)
		reader = bytes.NewReader(buf)
		if contentType == "" {
			contentType = "application/json"
		}
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, reader)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{status: res.StatusCode, raw: string(raw)}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

// path walks a dotted path through decoded JSON ("kpis.revenue.current").
func path(v any, p string) any {
	for _, key := range strings.Split(p, ".") {
		switch t := v.(type) {
		case map[string]any:
			v = t[key]
		case []any:
			var i int
			fmt.Sscanf(key, "%d", &i)
			if i >= len(t) {
				return nil
			}
			v = t[i]
		default:
			return nil
		}
	}
	return v
}

func money(t *testing.T, got any, want float64) {
	t.Helper()
	f, ok := got.(float64)
	if !ok || math.Abs(f-want) > 0.001 {
		t.Fatalf("money = %v, want %v", got, want)
	}
}

// ---- Dashboard (port of AdminDashboardTest) ---------------------------------

func TestDashboardRevenueIsConvertedIntoBaseCurrency(t *testing.T) {
	e := setup(t)
	e.currencies()
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	buyer := e.user("buyer@test", "customer", "b2c")

	// Current 30-day window: 100 EUR (= 200 USD at 0.5) + 50 USD = 250 USD.
	e.order(buyer, "EUR", 100, "completed", 2)
	e.order(buyer, "USD", 50, "paid", 1)
	// Pending never counts as revenue, but it is still an order.
	e.order(buyer, "USD", 999, "pending", 3)
	// Previous window (days 31–60 ago).
	e.order(buyer, "USD", 125, "completed", 40)
	// Outside both windows entirely.
	e.order(buyer, "USD", 5000, "completed", 200)

	r := e.call("GET", "/api/admin/dashboard?days=30", admin, nil, "")
	if r.status != 200 {
		t.Fatalf("status %d: %s", r.status, r.raw)
	}
	if path(r.body, "base_currency.code") != "USD" {
		t.Fatalf("base currency %v", path(r.body, "base_currency"))
	}
	money(t, path(r.body, "kpis.orders.current"), 3)
	money(t, path(r.body, "kpis.revenue.current"), 250)
	money(t, path(r.body, "kpis.revenue.previous"), 125)
	money(t, path(r.body, "kpis.revenue.change"), 100)
	// Average is over the two revenue-bearing orders only, not the pending one.
	money(t, path(r.body, "kpis.aov.current"), 125)
}

func TestDashboardTimeseriesCoversEveryDay(t *testing.T) {
	e := setup(t)
	e.currencies()
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	buyer := e.user("buyer@test", "customer", "b2c")
	e.order(buyer, "EUR", 10, "completed", 1)

	r := e.call("GET", "/api/admin/dashboard?days=7", admin, nil, "")
	series := path(r.body, "timeseries").([]any)
	if len(series) != 7 {
		t.Fatalf("expected 7 days, got %d", len(series))
	}
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	orders := 0.0
	for _, pt := range series {
		m := pt.(map[string]any)
		orders += m["orders"].(float64)
		if m["date"] == yesterday {
			money(t, m["revenue"], 20)
		}
	}
	if orders != 1 {
		t.Fatalf("expected 1 order in series, got %v", orders)
	}
}

func TestDashboardLeaderboardsReportBaseCurrencyTotals(t *testing.T) {
	e := setup(t)
	e.currencies()
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	buyer := e.user("buyer@test", "customer", "b2c")
	e.exec(`INSERT INTO categories (slug, name, position) VALUES ('tools', 'Tools', 0)`)
	e.exec(`INSERT INTO products (category_id, sku, slug, name, base_price, stock) VALUES (1, 'W-1', 'wrench', 'Wrench', 20, 10)`)
	orderID := e.order(buyer, "EUR", 40, "completed", 1)
	e.exec(`INSERT INTO order_items (order_id, product_id, name, sku, quantity, unit_price, line_total) VALUES ($1, 1, 'Wrench', 'W-1', 2, 20, 40)`, orderID)
	e.exec(`INSERT INTO payments (order_id, gateway, status, currency_code, amount, paid_at, created_at, updated_at) VALUES ($1, 'fake', 'completed', 'EUR', 40, now(), now(), now())`, orderID)

	r := e.call("GET", "/api/admin/dashboard?days=30", admin, nil, "")
	if path(r.body, "top_products.0.sku") != "W-1" || path(r.body, "top_products.0.qty") != 2.0 {
		t.Fatalf("top products %v", path(r.body, "top_products"))
	}
	if path(r.body, "top_categories.0.name") != "Tools" || path(r.body, "gateways.0.gateway") != "fake" {
		t.Fatalf("categories %v gateways %v", path(r.body, "top_categories"), path(r.body, "gateways"))
	}
	// 40 EUR at a 0.5 rate is 80 USD everywhere it is reported.
	money(t, path(r.body, "top_products.0.revenue"), 80)
	money(t, path(r.body, "top_categories.0.revenue"), 80)
	money(t, path(r.body, "gateways.0.amount"), 80)
}

func TestDashboardStatusBreakdownAndRangeAndGuards(t *testing.T) {
	e := setup(t)
	e.currencies()
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	buyerID := e.user("buyer@test", "customer", "b2c")
	e.order(buyerID, "USD", 10, "cancelled", 1)

	r := e.call("GET", "/api/admin/dashboard", admin, nil, "")
	breakdown := path(r.body, "orders_by_status").([]any)
	if len(breakdown) != 6 {
		t.Fatalf("expected every status, got %d", len(breakdown))
	}
	counts := map[string]float64{}
	for _, row := range breakdown {
		m := row.(map[string]any)
		counts[m["status"].(string)] = m["count"].(float64)
	}
	if counts["cancelled"] != 1 || counts["completed"] != 0 {
		t.Fatalf("counts %v", counts)
	}
	if path(r.body, "range.days") != 30.0 {
		t.Fatalf("default range %v", path(r.body, "range"))
	}

	if r := e.call("GET", "/api/admin/dashboard?days=365", admin, nil, ""); r.status != 422 || path(r.body, "errors.days") == nil {
		t.Fatalf("expected days validation error, got %d %s", r.status, r.raw)
	}
	if r := e.call("GET", "/api/admin/dashboard?days=7", admin, nil, ""); path(r.body, "range.days") != 7.0 {
		t.Fatalf("range 7 got %v", path(r.body, "range"))
	}
	if r := e.call("GET", "/api/admin/dashboard", e.token(buyerID), nil, ""); r.status != 403 {
		t.Fatalf("buyer should be forbidden, got %d", r.status)
	}
	if r := e.call("GET", "/api/admin/dashboard", "", nil, ""); r.status != 401 {
		t.Fatalf("guest should be unauthenticated, got %d", r.status)
	}
}

// ---- Settings (port of AdminSettingsIconTest) -------------------------------

const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func multipartFile(t *testing.T, field, name, mime string, content []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func TestAdminCanUploadReplaceAndRemoveTheIcon(t *testing.T) {
	e := setup(t)
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	png, _ := base64.StdEncoding.DecodeString(tinyPNG)

	body, ct := multipartFile(t, "icon", "icon.txt", "text/plain", []byte("not an image"))
	if r := e.call("POST", "/api/admin/settings/icon", admin, body, ct); r.status != 422 || path(r.body, "errors.icon") == nil {
		t.Fatalf("expected icon validation error, got %d %s", r.status, r.raw)
	}

	body, ct = multipartFile(t, "icon", "icon.png", "image/png", png)
	r := e.call("POST", "/api/admin/settings/icon", admin, body, ct)
	if r.status != 200 {
		t.Fatalf("upload failed: %d %s", r.status, r.raw)
	}
	var stored string
	if err := e.pool.QueryRow(e.ctx, `SELECT value FROM app_settings WHERE key = 'main_icon_path'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if r.body["icon_url"] != "http://api.test/storage/"+stored {
		t.Fatalf("icon_url %v for %s", r.body["icon_url"], stored)
	}
	if !e.app.Disk.Exists(stored) {
		t.Fatalf("file %s was not stored", stored)
	}
	if res := e.call("GET", "/storage/"+stored, "", nil, ""); res.status != 200 {
		t.Fatalf("stored icon not served: %d", res.status)
	}

	// Public settings expose it too.
	if pub := e.call("GET", "/api/settings", "", nil, ""); pub.body["icon_url"] != r.body["icon_url"] {
		t.Fatalf("public settings %v", pub.body)
	}

	// Replacing removes the old file.
	r2 := e.call("POST", "/api/admin/settings/icon", admin, body, ct)
	if r2.status != 200 || e.app.Disk.Exists(stored) {
		t.Fatalf("old icon should be gone after replace (%d)", r2.status)
	}

	r3 := e.call("DELETE", "/api/admin/settings/icon", admin, nil, "")
	if r3.status != 200 || r3.body["icon_url"] != nil {
		t.Fatalf("remove: %d %s", r3.status, r3.raw)
	}
	entries, _ := os.ReadDir(filepath.Join(e.app.Disk.Root, "settings", "icons"))
	if len(entries) != 0 {
		t.Fatalf("icon files left behind: %d", len(entries))
	}
	var n int
	e.pool.QueryRow(e.ctx, `SELECT COUNT(*) FROM app_settings WHERE key = 'main_icon_path'`).Scan(&n)
	if n != 0 {
		t.Fatal("main_icon_path setting still present")
	}
}

func TestAdminCanUpdateCompanyDetails(t *testing.T) {
	e := setup(t)
	admin := e.token(e.user("admin@test", "admin", "b2b"))

	r := e.call("PUT", "/api/admin/settings", admin, map[string]any{
		"company_name":        "Acme Marketplace",
		"company_description": "Wholesale supplies for growing businesses.",
	}, "")
	if r.status != 200 || r.body["company_name"] != "Acme Marketplace" || r.body["company_description"] != "Wholesale supplies for growing businesses." {
		t.Fatalf("update: %d %s", r.status, r.raw)
	}
	var v string
	e.pool.QueryRow(e.ctx, `SELECT value FROM app_settings WHERE key = 'company_name'`).Scan(&v)
	if v != "Acme Marketplace" {
		t.Fatalf("stored value %q", v)
	}
	if r := e.call("PUT", "/api/admin/settings", admin, map[string]any{"mode": "retail"}, ""); r.status != 422 {
		t.Fatalf("invalid mode accepted: %d", r.status)
	}
}

// ---- Storefront order flow --------------------------------------------------

func TestOrderFlowWithTiersStockAndPolicy(t *testing.T) {
	e := setup(t)
	e.currencies()
	e.exec(`INSERT INTO companies (name, slug) VALUES ('Acme', 'acme')`)
	e.exec(`INSERT INTO products (sku, slug, name, base_price, stock, min_order_qty, is_b2b_only) VALUES
		('BRG', 'bearing', 'Bearing', 0.80, 1000, 100, true), ('CBL', 'cable', 'Cable', 4.50, 10, 1, false)`)
	e.exec(`INSERT INTO product_prices (product_id, currency_code, min_qty, price) VALUES (1, 'USD', 500, 0.68)`)
	hash, _ := auth.HashPassword("password", 4)
	e.exec(`INSERT INTO users (name, email, password, type, role, company_id, currency) VALUES ('B2B', 'b2b@test', $1, 'b2b', 'customer', 1, 'USD')`, hash)
	e.exec(`INSERT INTO users (name, email, password, type, role, currency) VALUES ('B2C', 'b2c@test', $1, 'b2c', 'customer', 'EUR')`, hash)

	login := e.call("POST", "/api/auth/login", "", map[string]any{"email": "b2b@test", "password": "password"}, "")
	if login.status != 200 {
		t.Fatalf("login %d %s", login.status, login.raw)
	}
	b2b := login.body["token"].(string)
	b2c := e.token(2)

	// Guests and B2C buyers never see B2B-only stock.
	if r := e.call("GET", "/api/products", "", nil, ""); path(r.body, "meta.total") != 1.0 {
		t.Fatalf("guest catalog %v", path(r.body, "meta"))
	}
	if r := e.call("GET", "/api/products?qty=500", b2b, nil, ""); path(r.body, "meta.total") != 2.0 || path(r.body, "data.0.price.amount") != 0.68 {
		t.Fatalf("b2b catalog %s", r.raw)
	}

	// MOQ and B2B-only rules are 422 business errors.
	if r := e.call("POST", "/api/orders", b2b, map[string]any{"items": []any{map[string]any{"product_id": 1, "quantity": 5}}, "shipping_address": "x"}, ""); r.status != 422 || !strings.Contains(r.raw, "Minimum order quantity") {
		t.Fatalf("moq: %d %s", r.status, r.raw)
	}
	if r := e.call("POST", "/api/orders", b2c, map[string]any{"items": []any{map[string]any{"product_id": 1, "quantity": 500}}, "shipping_address": "x"}, ""); r.status != 422 || !strings.Contains(r.raw, "business accounts only") {
		t.Fatalf("b2b-only: %d %s", r.status, r.raw)
	}
	if r := e.call("POST", "/api/orders", b2b, map[string]any{"items": []any{map[string]any{"product_id": 2, "quantity": 11}}, "shipping_address": "x"}, ""); r.status != 422 || !strings.Contains(r.raw, "Insufficient stock") {
		t.Fatalf("stock: %d %s", r.status, r.raw)
	}

	// A valid order snapshots tier prices in the chosen currency and reserves stock.
	r := e.call("POST", "/api/orders", b2b, map[string]any{
		"items":            []any{map[string]any{"product_id": 1, "quantity": 500}, map[string]any{"product_id": 2, "quantity": 2}},
		"currency_code":    "EUR",
		"shipping_address": "Main st 1",
		"tax_rate":         0.1,
	}, "")
	if r.status != 201 {
		t.Fatalf("place: %d %s", r.status, r.raw)
	}
	if r.body["subtotal"] != "174.5000" || r.body["tax_total"] != "17.4500" || r.body["grand_total"] != "191.9500" || r.body["type"] != "b2b" {
		t.Fatalf("totals %s", r.raw)
	}
	if path(r.body, "items.0.unit_price") != "0.3400" || path(r.body, "items.1.unit_price") != "2.2500" {
		t.Fatalf("items %v", r.body["items"])
	}
	var stock int
	e.pool.QueryRow(e.ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock)
	if stock != 500 {
		t.Fatalf("stock not reserved: %d", stock)
	}
	orderID := int(r.body["id"].(float64))

	// Only the buyer (or an admin / company colleague) may view or pay.
	if r := e.call("GET", fmt.Sprintf("/api/orders/%d", orderID), b2c, nil, ""); r.status != 403 {
		t.Fatalf("policy: %d", r.status)
	}
	pay := e.call("POST", fmt.Sprintf("/api/orders/%d/pay", orderID), b2b, map[string]any{"gateway": "fake"}, "")
	if pay.status != 201 || path(pay.body, "order.status") != "paid" || path(pay.body, "payment.status") != "completed" {
		t.Fatalf("pay: %d %s", pay.status, pay.raw)
	}
	if r := e.call("POST", fmt.Sprintf("/api/orders/%d/pay", orderID), b2b, map[string]any{"gateway": "fake"}, ""); r.status != 422 {
		t.Fatalf("double pay: %d %s", r.status, r.raw)
	}
	list := e.call("GET", "/api/orders", b2b, nil, "")
	if path(list.body, "total") != 1.0 || path(list.body, "data.0.payment.gateway") != "fake" {
		t.Fatalf("orders index %s", list.raw)
	}
}

func TestAnalogLinksAreSymmetric(t *testing.T) {
	e := setup(t)
	e.currencies()
	admin := e.token(e.user("admin@test", "admin", "b2b"))
	e.exec(`INSERT INTO products (sku, slug, name, base_price, stock) VALUES ('A', 'a', 'A', 1, 1), ('B', 'b', 'B', 2, 1)`)

	r := e.call("POST", "/api/admin/products/1/analogs", admin, map[string]any{"analog_id": 2, "type": "upgrade", "note": "n"}, "")
	if r.status != 201 || !strings.Contains(r.raw, `"sku":"B"`) {
		t.Fatalf("link: %d %s", r.status, r.raw)
	}
	if r := e.call("GET", "/api/products/2/analogs", "", nil, ""); path(r.body, "data.0.sku") != "A" || path(r.body, "data.0.relation.type") != "upgrade" {
		t.Fatalf("reverse link missing: %s", r.raw)
	}
	if r := e.call("DELETE", "/api/admin/products/2/analogs/1", admin, nil, ""); r.status != 200 || r.raw != "[]\n" {
		t.Fatalf("unlink: %d %s", r.status, r.raw)
	}
	if r := e.call("GET", "/api/admin/products/1/analogs", admin, nil, ""); r.raw != "[]\n" {
		t.Fatalf("link should be gone on both sides: %s", r.raw)
	}
}
