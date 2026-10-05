package services

import (
	"context"
	"testing"

	"github.com/asihat/b2b-marketplace/backend/internal/jsonx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
)

func cachedCurrencyService(t *testing.T) *CurrencyService {
	t.Helper()
	s := NewCurrencyService(nil, "USD")
	_, err := s.active.Get(0, func() (map[string]CurrencyInfo, error) {
		return map[string]CurrencyInfo{
			"USD": {Code: "USD", Symbol: "$", ExchangeRate: 1, IsBase: true},
			"EUR": {Code: "EUR", Symbol: "€", ExchangeRate: 0.92},
			"KZT": {Code: "KZT", Symbol: "₸", ExchangeRate: 470},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPriceForPrefersOverridesThenBaseTiersThenBasePrice(t *testing.T) {
	ctx := context.Background()
	s := cachedCurrencyService(t)
	product := &models.Product{ID: 1, BasePrice: jsonx.Money(0.80)}
	tiers := []models.ProductPrice{
		{ProductID: 1, CurrencyCode: "USD", MinQty: 500, Price: jsonx.Money(0.68)},
		{ProductID: 1, CurrencyCode: "EUR", MinQty: 1000, Price: jsonx.Money(0.50)},
	}

	cases := []struct {
		currency string
		qty      int
		want     float64
	}{
		{"USD", 1, 0.80},     // base price
		{"USD", 500, 0.68},   // base-currency tier
		{"EUR", 1, 0.74},     // converted base price
		{"EUR", 500, 0.63},   // base tier converted (0.68 * 0.92)
		{"EUR", 1000, 0.50},  // explicit EUR override wins verbatim
		{"KZT", 500, 319.60}, // 0.68 * 470
		{"usd", 2, 0.80},     // case-insensitive
	}
	for _, c := range cases {
		if got := s.PriceFor(ctx, product, tiers, c.currency, c.qty); got != c.want {
			t.Errorf("PriceFor(%s, %d) = %v, want %v", c.currency, c.qty, got, c.want)
		}
	}
}

func TestToBaseAndFormat(t *testing.T) {
	ctx := context.Background()
	s := cachedCurrencyService(t)
	if got := s.ToBase(ctx, 92, "EUR"); got != 100 {
		t.Fatalf("ToBase = %v", got)
	}
	if got := s.Format(ctx, 1234.5, "USD"); got != "$1,234.50" {
		t.Fatalf("Format = %q", got)
	}
	if got := s.Format(ctx, 3, "XXX"); got != "XXX 3.00" {
		t.Fatalf("Format unknown = %q", got)
	}
	if got := NumberFormat(-1234567.891, 2); got != "-1,234,567.89" {
		t.Fatalf("NumberFormat = %q", got)
	}
	if got := Round(2.675, 2); got != 2.68 {
		t.Fatalf("Round half away from zero = %v", got)
	}
}
