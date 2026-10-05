package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/cache"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
)

// CurrencyInfo is the cached, plain view of an active currency.
type CurrencyInfo struct {
	Code         string
	Name         string
	Symbol       string
	ExchangeRate float64
	IsBase       bool
}

// CurrencyService implements multi-currency pricing. Prices are stored in a
// base currency; other currencies are derived via exchange rates unless an
// explicit ProductPrice override exists (fixed prices and B2B volume tiers).
type CurrencyService struct {
	db       *pgxpool.Pool
	fallback string
	active   cache.Value[map[string]CurrencyInfo]
}

func NewCurrencyService(pool *pgxpool.Pool, fallbackBase string) *CurrencyService {
	return &CurrencyService{db: pool, fallback: fallbackBase}
}

// Active returns the active currencies keyed by code (cached for an hour).
func (s *CurrencyService) Active(ctx context.Context) (map[string]CurrencyInfo, error) {
	return s.active.Get(time.Hour, func() (map[string]CurrencyInfo, error) {
		rows, err := Collect[models.Currency](s.db.Query(ctx,
			"SELECT "+db.Columns(models.CurrencyColumns, "")+" FROM currencies WHERE is_active = true"))
		if err != nil {
			return nil, err
		}
		out := make(map[string]CurrencyInfo, len(rows))
		for _, c := range rows {
			out[c.Code] = CurrencyInfo{Code: c.Code, Name: c.Name, Symbol: c.Symbol, ExchangeRate: c.ExchangeRate.Float(), IsBase: c.IsBase}
		}
		return out, nil
	})
}

func (s *CurrencyService) Base(ctx context.Context) CurrencyInfo {
	active, _ := s.Active(ctx)
	codes := make([]string, 0, len(active))
	for code := range active {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if active[code].IsBase {
			return active[code]
		}
	}
	return CurrencyInfo{Code: s.fallback, Symbol: "$", ExchangeRate: 1, IsBase: true}
}

// Rate is how many units of currency one unit of the base currency buys.
func (s *CurrencyService) Rate(ctx context.Context, currency string) float64 {
	active, _ := s.Active(ctx)
	if c, ok := active[strings.ToUpper(currency)]; ok {
		return c.ExchangeRate
	}
	return 1
}

// Convert turns a base-currency amount into the target currency.
func (s *CurrencyService) Convert(ctx context.Context, amount float64, to string) float64 {
	return Round(amount*s.Rate(ctx, to), 2)
}

// ToBase folds an amount held in another currency back into the base currency.
func (s *CurrencyService) ToBase(ctx context.Context, amount float64, from string) float64 {
	rate := s.Rate(ctx, from)
	if rate > 0 {
		return Round(amount/rate, 2)
	}
	return Round(amount, 2)
}

// PriceFor resolves the unit price of a product for a quantity, given the
// product's price overrides. Order of preference: an override in the
// requested currency, then a base-currency tier converted at the rate, then
// the converted base price.
func (s *CurrencyService) PriceFor(ctx context.Context, product *models.Product, tiers []models.ProductPrice, currency string, qty int) float64 {
	currency = strings.ToUpper(currency)
	if qty < 1 {
		qty = 1
	}

	if override := bestTier(tiers, currency, qty); override != nil {
		return Round(override.Price.Float(), 2)
	}

	base := s.Base(ctx).Code
	if currency != base {
		if tier := bestTier(tiers, base, qty); tier != nil {
			return s.Convert(ctx, tier.Price.Float(), currency)
		}
	}

	return s.Convert(ctx, product.BasePrice.Float(), currency)
}

// bestTier picks the tier with the highest min_qty <= qty for a currency.
func bestTier(tiers []models.ProductPrice, currency string, qty int) *models.ProductPrice {
	var best *models.ProductPrice
	for i := range tiers {
		t := &tiers[i]
		if t.CurrencyCode != currency || int(t.MinQty) > qty {
			continue
		}
		if best == nil || t.MinQty > best.MinQty {
			best = t
		}
	}
	return best
}

// PriceForProduct is PriceFor with the tiers loaded on demand.
func (s *CurrencyService) PriceForProduct(ctx context.Context, q db.Querier, product *models.Product, currency string, qty int) (float64, error) {
	tiers, err := LoadPrices(ctx, q, []int64{product.ID})
	if err != nil {
		return 0, err
	}
	return s.PriceFor(ctx, product, tiers[product.ID], currency, qty), nil
}

// EffectivePriceSQL is PriceFor as a SQL expression over the "products"
// table, for sorting and filtering by the price a buyer actually pays. The
// bind values are registered on q.
func (s *CurrencyService) EffectivePriceSQL(ctx context.Context, q *db.Query, currency string, qty int) string {
	currency = strings.ToUpper(currency)
	rate := s.Rate(ctx, currency)
	base := s.Base(ctx).Code

	return fmt.Sprintf(`COALESCE(
    (SELECT pp.price FROM product_prices pp
      WHERE pp.product_id = products.id AND pp.currency_code = %s AND pp.min_qty <= %s
      ORDER BY pp.min_qty DESC LIMIT 1),
    (SELECT bp.price * CAST(%s AS DECIMAL(18, 8)) FROM product_prices bp
      WHERE bp.product_id = products.id AND bp.currency_code = %s AND bp.min_qty <= %s
      ORDER BY bp.min_qty DESC LIMIT 1),
    products.base_price * CAST(%s AS DECIMAL(18, 8))
)`, q.Arg(currency), q.Arg(qty), q.Arg(rate), q.Arg(base), q.Arg(qty), q.Arg(rate))
}

// Format renders an amount with the currency symbol, like "$1,234.50".
func (s *CurrencyService) Format(ctx context.Context, amount float64, currency string) string {
	active, _ := s.Active(ctx)
	symbol := currency + " "
	if c, ok := active[currency]; ok {
		symbol = c.Symbol
	}
	return symbol + NumberFormat(amount, 2)
}

func (s *CurrencyService) Flush() { s.active.Flush() }

// Round rounds half away from zero, like PHP's round().
func Round(v float64, places int) float64 {
	p := math.Pow10(places)
	return math.Round(v*p) / p
}

// NumberFormat mirrors PHP number_format($n, $decimals) with "," thousands
// separators and "." as the decimal point.
func NumberFormat(v float64, decimals int) string {
	s := strconv.FormatFloat(Round(v, decimals), 'f', decimals, 64)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if decimals > 0 {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}
