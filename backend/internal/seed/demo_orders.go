package seed

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

const (
	demoDays       = 90
	demoOrderCount = 140
)

// Status -> weight. Roughly mirrors a healthy funnel with some churn.
var statusWeights = []weighted{
	{models.OrderCompleted, 34}, {models.OrderShipped, 15}, {models.OrderProcessing, 12},
	{models.OrderPaid, 14}, {models.OrderPending, 15}, {models.OrderCancelled, 10},
}

var gatewayWeights = []weighted{{"fake", 70}, {"manual", 30}}

type weighted struct {
	key    string
	weight int
}

func pick(r *rand.Rand, weights []weighted) string {
	total := 0
	for _, w := range weights {
		total += w.weight
	}
	roll := r.Intn(total) + 1
	for _, w := range weights {
		roll -= w.weight
		if roll <= 0 {
			return w.key
		}
	}
	return weights[0].key
}

// RunDemoOrders writes back-dated demo orders so the admin dashboard has
// trends, leaderboards and revenue on a fresh install. Orders are written
// directly (not through OrderService) because they need historical
// timestamps and must not decrement today's stock levels. It skips itself
// when orders already exist.
func RunDemoOrders(ctx context.Context, app *services.App) error {
	exists, err := services.Exists(ctx, app.DB, "orders", "true")
	if err != nil {
		return err
	}
	if exists {
		logf("orders already exist — skipping demo order seeding")
		return nil
	}

	buyers, err := services.Collect[models.User](app.DB.Query(ctx,
		"SELECT "+db.Columns(models.UserColumns, "")+" FROM users WHERE role <> 'admin' ORDER BY id"))
	if err != nil {
		return err
	}
	products, err := services.Collect[models.Product](app.DB.Query(ctx,
		"SELECT "+db.Columns(models.ProductColumns, "")+" FROM products WHERE is_active = true ORDER BY id"))
	if err != nil {
		return err
	}
	if len(buyers) == 0 || len(products) == 0 {
		logf("no buyers or products to build demo orders from")
		return nil
	}
	tiers, err := services.LoadPrices(ctx, app.DB, services.IDs(products, func(p models.Product) *int64 { return &p.ID }))
	if err != nil {
		return err
	}

	// Deterministic history: re-seeding produces the same demo numbers.
	r := rand.New(rand.NewSource(20260101))
	today := time.Now().UTC()
	base := app.Currency.Base(ctx).Code

	for i := 0; i < demoOrderCount; i++ {
		placedAt := randomOrderDate(r, today)
		buyer := &buyers[r.Intn(len(buyers))]
		if err := createDemoOrder(ctx, app, r, buyer, products, tiers, base, placedAt); err != nil {
			return err
		}
	}
	logf("%d demo orders seeded across the last %d days", demoOrderCount, demoDays)
	return nil
}

// randomOrderDate picks a moment in the window, biased towards recent days so
// period-over-period deltas show growth rather than noise.
func randomOrderDate(r *rand.Rand, today time.Time) time.Time {
	var daysAgo int
	for {
		daysAgo = r.Intn(demoDays)
		recency := 1 - float64(daysAgo)/demoDays
		if float64(r.Intn(101)) <= 35+65*recency {
			break
		}
	}
	d := today.AddDate(0, 0, -daysAgo)
	return time.Date(d.Year(), d.Month(), d.Day(), 8+r.Intn(13), r.Intn(60), r.Intn(60), 0, time.UTC)
}

func createDemoOrder(ctx context.Context, app *services.App, r *rand.Rand, buyer *models.User, products []models.Product, tiers map[int64][]models.ProductPrice, base string, placedAt time.Time) error {
	isB2B := buyer.IsB2B()
	currency := buyer.Currency
	if currency == "" {
		currency = base
	}
	status := pick(r, statusWeights)

	// B2C shoppers never see B2B-only stock, and buy in smaller baskets.
	catalog := products
	if !isB2B {
		catalog = nil
		for i := range products {
			if !products[i].IsB2BOnly {
				catalog = append(catalog, products[i])
			}
		}
	}
	maxLines := 3
	if isB2B {
		maxLines = 4
	}
	count := min(len(catalog), 1+r.Intn(maxLines))
	perm := r.Perm(len(catalog))[:count]

	orderType := models.TypeB2C
	if isB2B {
		orderType = models.TypeB2B
	}

	var orderID int64
	if err := app.DB.QueryRow(ctx, `
		INSERT INTO orders (number, user_id, company_id, type, status, currency_code, contact_name, contact_email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9) RETURNING id`,
		"ORD-"+strings.ToUpper(strx.Random(8)), buyer.ID, buyer.CompanyID, orderType, status, currency, buyer.Name, buyer.Email, placedAt,
	).Scan(&orderID); err != nil {
		return err
	}

	subtotal := 0.0
	for _, idx := range perm {
		p := &catalog[idx]
		moq := int(p.MinOrderQty)
		var qty int
		if isB2B {
			qty = max(moq, (1+r.Intn(12))*moq)
		} else {
			qty = max(moq, 1+r.Intn(3))
		}
		unitPrice := app.Currency.PriceFor(ctx, p, tiers[p.ID], currency, qty)
		lineTotal := services.Round(unitPrice*float64(qty), 4)
		subtotal += lineTotal

		if _, err := app.DB.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, name, sku, quantity, unit_price, line_total, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
			orderID, p.ID, p.Name, p.SKU, qty, unitPrice, lineTotal, placedAt); err != nil {
			return err
		}
	}

	if _, err := app.DB.Exec(ctx, `UPDATE orders SET subtotal = $1, tax_total = 0, grand_total = $1, updated_at = $2 WHERE id = $3`,
		subtotal, placedAt, orderID); err != nil {
		return err
	}

	if status != models.OrderPending && status != models.OrderCancelled {
		// Settled orders get a completed payment shortly after they were placed.
		paidAt := placedAt.Add(time.Duration(1+r.Intn(240)) * time.Minute)
		if _, err := app.DB.Exec(ctx, `
			INSERT INTO payments (order_id, gateway, status, currency_code, amount, reference, paid_at, created_at, updated_at)
			VALUES ($1, $2, 'completed', $3, $4, $5, $6, $6, $6)`,
			orderID, pick(r, gatewayWeights), currency, subtotal, "DEMO-"+strings.ToUpper(strx.Random(10)), paidAt); err != nil {
			return err
		}
	}
	return nil
}
