package handlers

import (
	"net/http"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/jsonx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, h.app.Settings.Public(r.Context()))
	return nil
}

type categoryOut struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
}

func (h *Handlers) Categories(w http.ResponseWriter, r *http.Request) error {
	cats, err := services.Collect[models.Category](h.app.DB.Query(r.Context(),
		"SELECT "+db.Columns(models.CategoryColumns, "")+" FROM categories WHERE is_active = true ORDER BY position, id"))
	if err != nil {
		return err
	}
	locale := Locale(r)
	out := make([]categoryOut, 0, len(cats))
	for i := range cats {
		c := &cats[i]
		out = append(out, categoryOut{ID: c.ID, ParentID: c.ParentID, Slug: c.Slug, Name: c.TranslatedName(locale)})
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

type currencyOut struct {
	Code         string     `db:"code" json:"code"`
	Name         string     `db:"name" json:"name"`
	Symbol       string     `db:"symbol" json:"symbol"`
	ExchangeRate jsonx.Rate `db:"exchange_rate" json:"exchange_rate"`
	IsBase       bool       `db:"is_base" json:"is_base"`
}

func (h *Handlers) Currencies(w http.ResponseWriter, r *http.Request) error {
	rows, err := services.Collect[currencyOut](h.app.DB.Query(r.Context(),
		`SELECT code, name, symbol, exchange_rate, is_base FROM currencies WHERE is_active = true ORDER BY id`))
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []currencyOut{}
	}
	httpx.JSON(w, http.StatusOK, rows)
	return nil
}

type languageOut struct {
	Code       string `db:"code" json:"code"`
	Name       string `db:"name" json:"name"`
	NativeName string `db:"native_name" json:"native_name"`
	IsDefault  bool   `db:"is_default" json:"is_default"`
}

func (h *Handlers) Languages(w http.ResponseWriter, r *http.Request) error {
	rows, err := services.Collect[languageOut](h.app.DB.Query(r.Context(),
		`SELECT code, name, native_name, is_default FROM languages WHERE is_active = true ORDER BY id`))
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []languageOut{}
	}
	httpx.JSON(w, http.StatusOK, rows)
	return nil
}

func (h *Handlers) Gateways(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"default":   h.app.Payments.Default(),
		"available": h.app.Payments.Available(),
	})
	return nil
}
