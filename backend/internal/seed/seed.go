// Package seed installs the demo catalog (currencies, languages, categories,
// companies, users, ~90 products with images, tiers and analog links) and the
// back-dated demo orders. Every step is an upsert, so re-running is safe.
package seed

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

type seeder struct {
	app      *services.App
	q        db.Querier
	now      time.Time
	password string
}

// Run seeds the whole demo data set (the former DatabaseSeeder).
func Run(ctx context.Context, app *services.App) error {
	hash, err := auth.HashPassword("password", app.Cfg.BcryptCost)
	if err != nil {
		return err
	}
	s := &seeder{app: app, q: app.DB, now: time.Now().UTC(), password: hash}

	if err := s.currencies(ctx); err != nil {
		return fmt.Errorf("currencies: %w", err)
	}
	if err := s.languages(ctx); err != nil {
		return fmt.Errorf("languages: %w", err)
	}
	categories, err := s.categories(ctx)
	if err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	companyID, err := s.companyAndUsers(ctx)
	if err != nil {
		return fmt.Errorf("company and users: %w", err)
	}
	if err := s.anchorProducts(ctx, categories, companyID); err != nil {
		return fmt.Errorf("anchor products: %w", err)
	}
	if err := s.catalog(ctx, categories, companyID); err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	if err := s.secondSupplier(ctx, companyID); err != nil {
		return fmt.Errorf("second supplier: %w", err)
	}
	app.FlushCaches()
	if err := RunDemoOrders(ctx, app); err != nil {
		return fmt.Errorf("demo orders: %w", err)
	}
	return nil
}

func (s *seeder) currencies(ctx context.Context) error {
	rows := []struct {
		code, name, symbol string
		rate               float64
		base               bool
	}{
		{"USD", "US Dollar", "$", 1.0, true},
		{"EUR", "Euro", "€", 0.92, false},
		{"KZT", "Kazakhstani Tenge", "₸", 470.0, false},
		{"RUB", "Russian Ruble", "₽", 90.0, false},
	}
	for _, c := range rows {
		if _, err := s.q.Exec(ctx, `
			INSERT INTO currencies (code, name, symbol, exchange_rate, is_base, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, $6, $6)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, symbol = EXCLUDED.symbol,
				exchange_rate = EXCLUDED.exchange_rate, is_base = EXCLUDED.is_base, updated_at = EXCLUDED.updated_at`,
			c.code, c.name, c.symbol, c.rate, c.base, s.now); err != nil {
			return err
		}
	}
	// Rates are cached; without this, re-seeding would keep converting at old rates.
	s.app.Currency.Flush()
	return nil
}

func (s *seeder) languages(ctx context.Context) error {
	rows := []struct {
		code, name, native string
		def                bool
	}{
		{"en", "English", "English", true},
		{"ru", "Russian", "Русский", false},
		{"kk", "Kazakh", "Қазақша", false},
	}
	for _, l := range rows {
		if _, err := s.q.Exec(ctx, `
			INSERT INTO languages (code, name, native_name, is_default, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, true, $5, $5)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, native_name = EXCLUDED.native_name,
				is_default = EXCLUDED.is_default, updated_at = EXCLUDED.updated_at`,
			l.code, l.name, l.native, l.def, s.now); err != nil {
			return err
		}
	}
	s.app.Locale.Flush()
	return nil
}

func (s *seeder) categories(ctx context.Context) (map[string]int64, error) {
	defs := []struct {
		slug  string
		names map[string]string
	}{
		{"electronics", map[string]string{"en": "Electronics", "ru": "Электроника", "kk": "Электроника"}},
		{"industrial", map[string]string{"en": "Industrial Supplies", "ru": "Промышленные товары", "kk": "Өнеркәсіптік тауарлар"}},
		{"office", map[string]string{"en": "Office", "ru": "Офис", "kk": "Кеңсе"}},
		{"tools", map[string]string{"en": "Tools", "ru": "Инструменты", "kk": "Құралдар"}},
		{"packaging", map[string]string{"en": "Packaging", "ru": "Упаковка", "kk": "Қаптама"}},
		{"safety", map[string]string{"en": "Safety & PPE", "ru": "Средства защиты", "kk": "Қорғаныс құралдары"}},
	}
	out := map[string]int64{}
	for pos, d := range defs {
		var id int64
		if err := s.q.QueryRow(ctx, `
			INSERT INTO categories (slug, name, name_translations, position, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, true, $5, $5)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, name_translations = EXCLUDED.name_translations,
				position = EXCLUDED.position, updated_at = EXCLUDED.updated_at
			RETURNING id`, d.slug, d.names["en"], d.names, pos, s.now).Scan(&id); err != nil {
			return nil, err
		}
		out[d.slug] = id
	}
	return out, nil
}

type companySpec struct {
	slug, name, taxNumber, email, phone, country string
}

func (s *seeder) upsertCompany(ctx context.Context, c companySpec) (int64, error) {
	var id int64
	var email, phone *string
	if c.email != "" {
		email = &c.email
	}
	if c.phone != "" {
		phone = &c.phone
	}
	err := s.q.QueryRow(ctx, `
		INSERT INTO companies (name, slug, tax_number, email, phone, country, default_currency, default_locale, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'USD', 'en', true, $7, $7)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, tax_number = EXCLUDED.tax_number, email = EXCLUDED.email,
			phone = EXCLUDED.phone, country = EXCLUDED.country, default_currency = EXCLUDED.default_currency,
			default_locale = EXCLUDED.default_locale, is_verified = EXCLUDED.is_verified, updated_at = EXCLUDED.updated_at
		RETURNING id`, c.name, c.slug, c.taxNumber, email, phone, c.country, s.now).Scan(&id)
	return id, err
}

type userSpec struct {
	email, name, typ, role, currency string
	companyID                        *int64
}

func (s *seeder) upsertUser(ctx context.Context, u userSpec) error {
	if u.currency == "" {
		u.currency = "USD"
	}
	_, err := s.q.Exec(ctx, `
		INSERT INTO users (name, email, password, type, role, company_id, currency, locale, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'en', true, $8, $8)
		ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, password = EXCLUDED.password, type = EXCLUDED.type,
			role = EXCLUDED.role, company_id = EXCLUDED.company_id, currency = EXCLUDED.currency, updated_at = EXCLUDED.updated_at`,
		u.name, u.email, s.password, u.typ, u.role, u.companyID, u.currency, s.now)
	return err
}

func (s *seeder) companyAndUsers(ctx context.Context) (int64, error) {
	companyID, err := s.upsertCompany(ctx, companySpec{slug: "acme-trading", name: "Acme Trading LLC", taxNumber: "TX-1000-2000", country: "KZ"})
	if err != nil {
		return 0, err
	}
	users := []userSpec{
		{email: "admin@marketplace.test", name: "Marketplace Admin", typ: models.TypeB2B, role: models.RoleAdmin, companyID: &companyID},
		{email: "buyer@acme.test", name: "Acme Buyer", typ: models.TypeB2B, role: models.RoleCustomer, companyID: &companyID, currency: "USD"},
		{email: "customer@example.test", name: "Retail Customer", typ: models.TypeB2C, role: models.RoleCustomer, currency: "EUR"},
	}
	for _, u := range users {
		if err := s.upsertUser(ctx, u); err != nil {
			return 0, err
		}
	}
	return companyID, nil
}

type productSpec struct {
	categoryID  int64
	sku, name   string
	brand, unit string
	basePrice   float64
	stock       int
	minOrderQty int
	b2bOnly     bool
}

// makeProduct creates/updates a product, attaches a volume tier and images.
func (s *seeder) makeProduct(ctx context.Context, companyID int64, companyName string, p productSpec) (int64, error) {
	if p.unit == "" {
		p.unit = "pcs"
	}
	if p.minOrderQty < 1 {
		p.minOrderQty = 1
	}
	slug := strx.Slug(p.name + " " + p.sku)
	description := p.name + " — supplied by " + companyName + ". Quality " + strings.ToLower(p.name) + " for B2B and retail buyers."

	id, err := s.upsertProduct(ctx, p.sku, slug, p.name, description, p.brand, p.unit, p.categoryID, companyID, p.basePrice, p.stock, p.minOrderQty, p.b2bOnly)
	if err != nil {
		return 0, err
	}

	// B2B volume tier: 15% off per unit at 500+.
	if p.b2bOnly || p.minOrderQty > 1 {
		if err := s.upsertTier(ctx, id, "USD", 500, services.Round(p.basePrice*0.85, 4)); err != nil {
			return 0, err
		}
	}
	if err := s.attachImages(ctx, id, slug, p.name); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *seeder) upsertProduct(ctx context.Context, sku, slug, name, description, brand, unit string, categoryID, companyID int64, price float64, stock, moq int, b2bOnly bool) (int64, error) {
	var id int64
	err := s.q.QueryRow(ctx, `
		INSERT INTO products (category_id, company_id, sku, slug, name, description, brand, unit, base_price, stock, min_order_qty, is_b2b_only, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, $13, $13)
		ON CONFLICT (sku) DO UPDATE SET category_id = EXCLUDED.category_id, company_id = EXCLUDED.company_id,
			slug = EXCLUDED.slug, name = EXCLUDED.name, description = EXCLUDED.description, brand = EXCLUDED.brand,
			unit = EXCLUDED.unit, base_price = EXCLUDED.base_price, stock = EXCLUDED.stock, min_order_qty = EXCLUDED.min_order_qty,
			is_b2b_only = EXCLUDED.is_b2b_only, is_active = true, updated_at = EXCLUDED.updated_at
		RETURNING id`, categoryID, companyID, sku, slug, name, description, brand, unit, price, stock, moq, b2bOnly, s.now).Scan(&id)
	return id, err
}

func (s *seeder) upsertTier(ctx context.Context, productID int64, currency string, minQty int, price float64) error {
	_, err := s.q.Exec(ctx, `
		INSERT INTO product_prices (product_id, currency_code, min_qty, price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (product_id, currency_code, min_qty) DO UPDATE SET price = EXCLUDED.price, updated_at = EXCLUDED.updated_at`,
		productID, currency, minQty, price, s.now)
	return err
}

func (s *seeder) upsertTranslation(ctx context.Context, productID int64, locale, name string, description *string) error {
	_, err := s.q.Exec(ctx, `
		INSERT INTO product_translations (product_id, locale, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (product_id, locale) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at`,
		productID, locale, name, description, s.now)
	return err
}

// attachImages adds deterministic placeholder images served by the app
// itself (GET /img/{seed}) when the product has none yet.
func (s *seeder) attachImages(ctx context.Context, productID int64, slug, name string) error {
	exists, err := services.Exists(ctx, s.q, "product_images", "product_id = $1", productID)
	if err != nil || exists {
		return err
	}
	for i := 1; i <= 3; i++ {
		url := httpx.AbsoluteURL(s.app.Cfg.AppURL, fmt.Sprintf("/img/%s-%d", slug, i))
		if _, err := s.q.Exec(ctx, `
			INSERT INTO product_images (product_id, url, alt, position, is_primary, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)`, productID, url, name, i, i == 1, s.now); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) linkAnalogs(ctx context.Context, a, b int64, typ string, note string) error {
	var n *string
	if note != "" {
		n = &note
	}
	for _, pair := range [][2]int64{{a, b}, {b, a}} {
		if _, err := s.q.Exec(ctx, `
			INSERT INTO product_analogs (product_id, analog_id, type, note, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
			ON CONFLICT (product_id, analog_id) DO NOTHING`, pair[0], pair[1], typ, n, s.now); err != nil {
			return err
		}
	}
	return nil
}

const acmeName = "Acme Trading LLC"

// anchorProducts are hand-crafted items with translations and explicit
// analog links, used by the i18n / analog walkthroughs.
func (s *seeder) anchorProducts(ctx context.Context, categories map[string]int64, companyID int64) error {
	type anchor struct {
		productSpec
		category string
		tr       map[string]string
	}
	anchors := []anchor{
		{productSpec{sku: "CBL-USB-C-1M", name: "USB-C Cable 1m", brand: "Volt", basePrice: 4.50, stock: 5000}, "electronics",
			map[string]string{"ru": "Кабель USB-C 1м", "kk": "USB-C кабелі 1м"}},
		{productSpec{sku: "CBL-USB-C-1M-PRO", name: "USB-C Cable 1m (Braided)", brand: "Volt", basePrice: 6.90, stock: 3000}, "electronics",
			map[string]string{"ru": "Кабель USB-C 1м (оплётка)", "kk": "USB-C кабелі 1м (өрімді)"}},
		{productSpec{sku: "IND-BEARING-608", name: "Ball Bearing 608ZZ", brand: "RollTech", basePrice: 0.80, stock: 100000, minOrderQty: 100, b2bOnly: true}, "industrial",
			map[string]string{"ru": "Подшипник 608ZZ", "kk": "Мойынтірек 608ZZ"}},
		{productSpec{sku: "IND-BEARING-608-2RS", name: "Ball Bearing 608-2RS", brand: "RollTech", basePrice: 0.95, stock: 80000, minOrderQty: 100, b2bOnly: true}, "industrial",
			map[string]string{"ru": "Подшипник 608-2RS", "kk": "Мойынтірек 608-2RS"}},
	}

	made := map[string]int64{}
	for _, a := range anchors {
		spec := a.productSpec
		spec.categoryID = categories[a.category]
		id, err := s.makeProduct(ctx, companyID, acmeName, spec)
		if err != nil {
			return err
		}
		for locale, name := range a.tr {
			if err := s.upsertTranslation(ctx, id, locale, name, &name); err != nil {
				return err
			}
		}
		made[a.sku] = id
	}

	if err := s.linkAnalogs(ctx, made["CBL-USB-C-1M"], made["CBL-USB-C-1M-PRO"], models.AnalogUpgrade, "Braided variant"); err != nil {
		return err
	}
	return s.linkAnalogs(ctx, made["IND-BEARING-608"], made["IND-BEARING-608-2RS"], models.AnalogEquivalent, "Sealed version")
}

type variant struct {
	name  string
	price float64
	stock int
}

type family struct {
	category, brand, unit string
	b2bOnly               bool
	moq                   int
	base                  string
	variants              []variant
}

var families = []family{
	{"electronics", "Volt", "pcs", false, 1, "HDMI 2.1 Cable", []variant{{"1m", 7.5, 4000}, {"2m", 9.9, 3500}, {"3m", 12.5, 2000}}},
	{"electronics", "Volt", "pcs", false, 1, "USB-C Power Adapter", []variant{{"30W", 14, 1500}, {"65W", 19, 1200}, {"100W", 29, 800}}},
	{"electronics", "Klyk", "pcs", false, 1, "Wireless Mouse", []variant{{"Basic", 8.9, 2200}, {"Ergo", 15.5, 1400}, {"Pro", 24.9, 900}}},
	{"electronics", "Klyk", "pcs", false, 1, "Mechanical Keyboard", []variant{{"TKL Red", 39, 600}, {"Full Brown", 45, 500}, {"Wireless Blue", 59, 350}}},
	{"electronics", "Datacore", "pcs", false, 1, "NVMe SSD", []variant{{"512GB", 42, 700}, {"1TB", 69, 500}, {"2TB", 119, 250}}},
	{"electronics", "Voltbank", "pcs", false, 1, "Power Bank", []variant{{"10000mAh", 18, 1300}, {"20000mAh", 27, 900}}},
	{"electronics", "Klyk", "pcs", false, 1, "USB Hub", []variant{{"4-Port", 11, 1800}, {"7-Port", 17, 1100}}},
	{"electronics", "SeeCam", "pcs", false, 1, "Webcam", []variant{{"720p", 16, 800}, {"1080p", 26, 650}, {"4K", 49, 300}}},

	{"industrial", "RollTech", "pcs", true, 100, "Hex Bolt M8", []variant{{"20mm", 0.12, 200000}, {"30mm", 0.15, 180000}, {"40mm", 0.18, 150000}}},
	{"industrial", "RollTech", "pcs", true, 100, "Ball Bearing 6203", []variant{{"ZZ", 1.1, 60000}, {"2RS", 1.3, 55000}}},
	{"industrial", "FlowLine", "m", true, 10, "Hydraulic Hose 1/2\"", []variant{{"SAE 1SN", 4.2, 9000}, {"SAE 2SN", 5.6, 7000}}},
	{"industrial", "BeltPro", "pcs", true, 5, "V-Belt A-Section", []variant{{"A40", 3.1, 12000}, {"A50", 3.6, 10000}, {"A60", 4.0, 9000}}},
	{"industrial", "SealMaster", "set", true, 5, "Gasket Set", []variant{{"Small", 6.5, 4000}, {"Medium", 9.0, 3000}, {"Large", 12.0, 2000}}},
	{"industrial", "ArcWeld", "kg", true, 5, "Welding Rod E6013", []variant{{"2.5mm", 2.2, 20000}, {"3.2mm", 2.5, 18000}}},
	{"industrial", "PipeCo", "m", true, 6, "Steel Pipe DN50", []variant{{"1.5mm wall", 7.8, 8000}, {"2.0mm wall", 9.4, 6000}}},

	{"office", "PaperCo", "ream", false, 1, "A4 Paper 80g", []variant{{"White", 5.2, 20000}, {"Recycled", 5.8, 9000}, {"Premium", 6.9, 7000}}},
	{"office", "InkFlow", "pcs", false, 1, "Ballpoint Pen", []variant{{"Blue", 0.4, 50000}, {"Black", 0.4, 50000}, {"Red", 0.4, 30000}}},
	{"office", "Clipix", "pcs", false, 1, "Stapler", []variant{{"Mini", 3.2, 4000}, {"Desktop", 5.5, 3000}, {"Heavy Duty", 11.0, 1200}}},
	{"office", "StikIt", "pack", false, 1, "Sticky Notes", []variant{{"Yellow 3x3", 1.2, 15000}, {"Neon 3x3", 1.6, 9000}}},
	{"office", "InkFlow", "pcs", false, 1, "Whiteboard Marker", []variant{{"Black", 0.9, 12000}, {"Assorted 4-pack", 3.2, 6000}}},
	{"office", "Filewise", "pcs", false, 1, "Lever Arch File", []variant{{"A4 Blue", 1.8, 9000}, {"A4 Black", 1.8, 9000}}},

	{"tools", "DriveMaster", "pcs", false, 1, "Cordless Drill", []variant{{"12V", 38, 700}, {"18V", 59, 500}, {"18V Brushless", 89, 250}}},
	{"tools", "GripPro", "set", false, 1, "Screwdriver Set", []variant{{"6-piece", 7.5, 2500}, {"12-piece", 12.9, 1500}}},
	{"tools", "GripPro", "pcs", false, 1, "Claw Hammer", []variant{{"16oz", 6.9, 3000}, {"20oz", 8.5, 2200}}},
	{"tools", "MeasureX", "pcs", false, 1, "Measuring Tape", []variant{{"5m", 3.5, 5000}, {"8m", 5.0, 3500}}},
	{"tools", "GripPro", "set", false, 1, "Combination Wrench Set", []variant{{"8-19mm", 18, 1200}, {"8-22mm", 24, 900}}},
	{"tools", "CutEdge", "pcs", false, 1, "Angle Grinder", []variant{{"115mm", 32, 800}, {"125mm", 39, 600}}},

	{"packaging", "BoxIt", "pcs", false, 1, "Cardboard Box", []variant{{"S 20x15x10", 0.45, 40000}, {"M 30x20x15", 0.7, 30000}, {"L 40x30x20", 1.1, 20000}}},
	{"packaging", "WrapCo", "roll", false, 1, "Bubble Wrap Roll", []variant{{"50cm x 50m", 6.5, 5000}, {"100cm x 50m", 11.0, 3000}}},
	{"packaging", "TapeWorks", "pcs", false, 1, "Packing Tape", []variant{{"Clear 48mm", 0.9, 30000}, {"Brown 48mm", 0.9, 25000}}},
	{"packaging", "WrapCo", "roll", false, 1, "Stretch Film", []variant{{"Standard", 4.8, 8000}, {"Heavy", 6.2, 6000}}},
	{"packaging", "BoxIt", "pack", false, 1, "Padded Mailer", []variant{{"A5", 2.4, 12000}, {"A4", 3.6, 9000}}},

	{"safety", "SafeGuard", "pcs", false, 1, "Safety Goggles", []variant{{"Clear", 2.2, 9000}, {"Anti-Fog", 3.4, 6000}}},
	{"safety", "HandPro", "pair", false, 1, "Work Gloves", []variant{{"Nitrile", 1.5, 20000}, {"Cut-Resistant", 3.8, 8000}, {"Leather", 5.2, 5000}}},
	{"safety", "SafeGuard", "pcs", false, 1, "Hard Hat", []variant{{"White", 6.5, 4000}, {"Yellow", 6.5, 4000}}},
	{"safety", "HiVis", "pcs", false, 1, "Hi-Vis Vest", []variant{{"Yellow M", 3.2, 7000}, {"Orange L", 3.2, 7000}}},
	{"safety", "BreatheWell", "pcs", false, 1, "Respirator Mask", []variant{{"FFP2", 0.8, 50000}, {"FFP3", 1.2, 30000}}},
}

var nonAlnum = regexp.MustCompile(`[^A-Z0-9]`)

// familyKey builds the SKU prefix: upper-cased ASCII letters/digits, max 10.
func familyKey(base string) string {
	key := nonAlnum.ReplaceAllString(strings.ToUpper(strx.ASCII(base)), "")
	if len(key) > 10 {
		key = key[:10]
	}
	return key
}

// catalog generates the broad catalog: each family yields several variants
// that are linked to each other as analogs.
func (s *seeder) catalog(ctx context.Context, categories map[string]int64, companyID int64) error {
	for _, f := range families {
		key := familyKey(f.base)
		var ids []int64
		for i, v := range f.variants {
			id, err := s.makeProduct(ctx, companyID, acmeName, productSpec{
				categoryID: categories[f.category], sku: fmt.Sprintf("%s-%d", key, i+1), name: f.base + " " + v.name,
				brand: f.brand, unit: f.unit, basePrice: v.price, stock: v.stock, minOrderQty: f.moq, b2bOnly: f.b2bOnly,
			})
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		for _, a := range ids {
			for _, b := range ids {
				if a != b {
					if err := s.linkAnalogs(ctx, a, b, models.AnalogEquivalent, "Variant of "+f.base); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// secondSupplier mirrors Acme's catalog under a second verified supplier at a
// slightly lower price, cross-linked as substitutes for B2B comparison.
func (s *seeder) secondSupplier(ctx context.Context, sourceCompanyID int64) error {
	supplierName := "Northwind Supply Co."
	supplierID, err := s.upsertCompany(ctx, companySpec{
		slug: "northwind-supply", name: supplierName, taxNumber: "NW-3000-4000",
		email: "sales@northwind.test", phone: "+7 700 300 4000", country: "KZ",
	})
	if err != nil {
		return err
	}
	if err := s.upsertUser(ctx, userSpec{email: "supplier@northwind.test", name: "Northwind Seller", typ: models.TypeB2B, role: models.RoleCustomer, companyID: &supplierID, currency: "USD"}); err != nil {
		return err
	}

	sources, err := services.Collect[models.Product](s.q.Query(ctx,
		"SELECT "+db.Columns(models.ProductColumns, "")+" FROM products WHERE company_id = $1 ORDER BY id", sourceCompanyID))
	if err != nil {
		return err
	}
	ids := services.IDs(sources, func(p models.Product) *int64 { return &p.ID })
	translations, err := services.LoadTranslations(ctx, s.q, ids)
	if err != nil {
		return err
	}
	prices, err := services.LoadPrices(ctx, s.q, ids)
	if err != nil {
		return err
	}

	for i := range sources {
		src := &sources[i]
		sku := "NW-" + src.SKU
		slug := strx.Slug(src.Name + " " + sku)
		price := services.Round(src.BasePrice.Float()*0.97, 4)
		stock := max(1, int(float64(src.Stock)*0.6))
		description := src.Name + " — supplied by " + supplierName + ". Alternative supplier offer for B2B comparison."
		brand := ""
		if src.Brand != nil {
			brand = *src.Brand
		}
		var categoryID int64
		if src.CategoryID != nil {
			categoryID = *src.CategoryID
		}

		id, err := s.upsertProduct(ctx, sku, slug, src.Name, description, brand, src.Unit, categoryID, supplierID, price, stock, int(src.MinOrderQty), src.IsB2BOnly)
		if err != nil {
			return err
		}
		for _, t := range translations[src.ID] {
			if err := s.upsertTranslation(ctx, id, t.Locale, t.Name, t.Description); err != nil {
				return err
			}
		}
		for _, tier := range prices[src.ID] {
			if err := s.upsertTier(ctx, id, tier.CurrencyCode, int(tier.MinQty), services.Round(tier.Price.Float()*0.97, 4)); err != nil {
				return err
			}
		}
		if err := s.attachImages(ctx, id, slug, src.Name); err != nil {
			return err
		}
		if err := s.linkAnalogs(ctx, src.ID, id, models.AnalogSubstitute, "Same item from another supplier"); err != nil {
			return err
		}
	}
	return nil
}

func logf(format string, args ...any) { slog.Info(fmt.Sprintf(format, args...)) }
