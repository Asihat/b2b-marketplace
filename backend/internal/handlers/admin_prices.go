package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

func (h *Handlers) pricesOf(r *http.Request, productID int64) ([]models.ProductPrice, error) {
	rows, err := services.Collect[models.ProductPrice](h.app.DB.Query(r.Context(),
		"SELECT "+db.Columns(models.ProductPriceColumns, "")+" FROM product_prices WHERE product_id = $1 ORDER BY currency_code, min_qty", productID))
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []models.ProductPrice{}
	}
	return rows, nil
}

func (h *Handlers) AdminPricesIndex(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(r.Context(), h.app.DB, id); err != nil {
		return err
	}
	prices, err := h.pricesOf(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, prices)
	return nil
}

type priceRow struct {
	currency string
	minQty   int64
	price    float64
}

// validatePriceRow checks one tier: currency (upper-cased, must exist),
// min_qty >= 1 and price >= 0.
func (h *Handlers) validatePriceRow(r *http.Request, v *validate.V) (priceRow, bool) {
	var row priceRow
	ok := true
	if raw, isStr := v.Get("currency_code").(string); isStr {
		v.Data()["currency_code"] = strings.ToUpper(raw)
	}
	if code, valid := v.String("currency_code", validate.Str{Required: true, Size: 3}); valid {
		exists, err := services.Exists(r.Context(), h.app.DB, "currencies", "code = $1", code)
		if err != nil || !exists {
			v.Fail("currency_code", "The selected currency code is invalid.")
			ok = false
		}
		row.currency = code
	} else {
		ok = false
	}
	if n, valid := v.Int("min_qty", validate.Num{Required: true, Min: validate.F(1)}); valid {
		row.minQty = n
	} else {
		ok = false
	}
	if f, valid := v.Float("price", validate.Num{Required: true, Min: validate.F(0)}); valid {
		row.price = f
	} else {
		ok = false
	}
	return row, ok
}

func (h *Handlers) AdminPricesStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	productID, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(ctx, h.app.DB, productID); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	row, ok := h.validatePriceRow(r, v)
	if ok {
		dup, err := services.Exists(ctx, h.app.DB, "product_prices", "product_id = $1 AND currency_code = $2 AND min_qty = $3", productID, row.currency, row.minQty)
		if err != nil {
			return err
		}
		if dup {
			v.Fail("currency_code", "The currency code has already been taken.")
		}
	}
	if err := v.Err(); err != nil {
		return err
	}

	var id int64
	now := time.Now().UTC()
	if err := h.app.DB.QueryRow(ctx, `
		INSERT INTO product_prices (product_id, currency_code, min_qty, price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5) RETURNING id`, productID, row.currency, row.minQty, row.price, now).Scan(&id); err != nil {
		return err
	}
	price, err := services.FindProductPrice(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, price)
	return nil
}

func (h *Handlers) ownedPrice(r *http.Request) (*models.ProductPrice, error) {
	productID, err := pathID(r, "product")
	if err != nil {
		return nil, err
	}
	priceID, err := pathID(r, "price")
	if err != nil {
		return nil, err
	}
	if _, err := services.FindProduct(r.Context(), h.app.DB, productID); err != nil {
		return nil, err
	}
	price, err := services.FindProductPrice(r.Context(), h.app.DB, priceID)
	if err != nil {
		return nil, err
	}
	if price.ProductID != productID {
		return nil, httpx.NotFound()
	}
	return price, nil
}

func (h *Handlers) AdminPricesUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	price, err := h.ownedPrice(r)
	if err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	row, ok := h.validatePriceRow(r, v)
	if ok {
		dup, err := services.Exists(ctx, h.app.DB, "product_prices", "product_id = $1 AND currency_code = $2 AND min_qty = $3 AND id <> $4",
			price.ProductID, row.currency, row.minQty, price.ID)
		if err != nil {
			return err
		}
		if dup {
			v.Fail("currency_code", "The currency code has already been taken.")
		}
	}
	if err := v.Err(); err != nil {
		return err
	}

	if _, err := h.app.DB.Exec(ctx, `UPDATE product_prices SET currency_code = $1, min_qty = $2, price = $3, updated_at = $4 WHERE id = $5`,
		row.currency, row.minQty, row.price, time.Now().UTC(), price.ID); err != nil {
		return err
	}
	fresh, err := services.FindProductPrice(ctx, h.app.DB, price.ID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, fresh)
	return nil
}

func (h *Handlers) AdminPricesDestroy(w http.ResponseWriter, r *http.Request) error {
	price, err := h.ownedPrice(r)
	if err != nil {
		return err
	}
	if _, err := h.app.DB.Exec(r.Context(), `DELETE FROM product_prices WHERE id = $1`, price.ID); err != nil {
		return err
	}
	httpx.Message(w, http.StatusOK, "Price deleted.")
	return nil
}

// AdminPricesSync replaces the product's whole price grid. Rows are rewritten
// rather than matched by id so reordering tiers can never trip the
// (product, currency, min_qty) unique index mid-update.
func (h *Handlers) AdminPricesSync(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	productID, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(ctx, h.app.DB, productID); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)

	var rows []priceRow
	if list, ok := v.Array("prices", validate.Arr{Present: true, Max: 100}); ok {
		seen := map[string]bool{}
		for i, raw := range list {
			obj, _ := raw.(map[string]any)
			sub := v.Sub("prices."+strconv.Itoa(i), obj)
			row, rowOK := h.validatePriceRow(r, sub)
			if !rowOK {
				continue
			}
			key := row.currency + "|" + strconv.FormatInt(row.minQty, 10)
			if seen[key] {
				sub.Fail("min_qty", fmt.Sprintf("%s already has a tier starting at %d.", row.currency, row.minQty))
				continue
			}
			seen[key] = true
			rows = append(rows, row)
		}
	}
	if err := v.Err(); err != nil {
		return err
	}

	err = db.Tx(ctx, h.app.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM product_prices WHERE product_id = $1`, productID); err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, row := range rows {
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_prices (product_id, currency_code, min_qty, price, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $5)`, productID, row.currency, row.minQty, row.price, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	prices, err := h.pricesOf(r, productID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, prices)
	return nil
}
