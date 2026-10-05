package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

type adminAnalogOut struct {
	ID       int64   `json:"id"`
	SKU      string  `json:"sku"`
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Brand    *string `json:"brand"`
	Stock    int32   `json:"stock"`
	IsActive bool    `json:"is_active"`
	Image    *string `json:"image"`
	Type     string  `json:"type"`
	Note     *string `json:"note"`
}

// analogsOf returns the product's full analog list, ordered by name.
func (h *Handlers) analogsOf(r *http.Request, productID int64) ([]adminAnalogOut, error) {
	analogs, links, err := h.linkedAnalogs(r, productID, nil)
	if err != nil {
		return nil, err
	}
	images, err := services.LoadImages(r.Context(), h.app.DB, services.IDs(analogs, func(p models.Product) *int64 { return &p.ID }))
	if err != nil {
		return nil, err
	}
	out := make([]adminAnalogOut, 0, len(analogs))
	for i := range analogs {
		p := &analogs[i]
		link := links[p.ID]
		out = append(out, adminAnalogOut{
			ID: p.ID, SKU: p.SKU, Slug: p.Slug, Name: p.Name, Brand: p.Brand, Stock: p.Stock, IsActive: p.IsActive,
			Image: models.PrimaryImageURL(images[p.ID]), Type: link.Type, Note: link.Note,
		})
	}
	// ORDER BY name, matching the former implementation.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Name < out[j-1].Name; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

// LinkAnalogs creates or updates the pair in both directions (exported for the seeder).
func LinkAnalogs(r *http.Request, q db.Querier, productID, analogID int64, typ string, note *string) error {
	ctx := r.Context()
	now := time.Now().UTC()
	for _, pair := range [][2]int64{{productID, analogID}, {analogID, productID}} {
		if _, err := q.Exec(ctx, `
			INSERT INTO product_analogs (product_id, analog_id, type, note, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
			ON CONFLICT (product_id, analog_id) DO UPDATE SET type = EXCLUDED.type, note = EXCLUDED.note, updated_at = EXCLUDED.updated_at`,
			pair[0], pair[1], typ, note, now); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) AdminAnalogsIndex(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(r.Context(), h.app.DB, id); err != nil {
		return err
	}
	list, err := h.analogsOf(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list)
	return nil
}

func (h *Handlers) AdminAnalogsStore(w http.ResponseWriter, r *http.Request) error {
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
	analogID, idOK := v.Int("analog_id", validate.Num{Required: true})
	if idOK {
		if analogID == productID {
			v.Fail("analog_id", "A product cannot be an analog of itself.")
		} else if _, err := services.FindProduct(ctx, h.app.DB, analogID); err != nil {
			if err == pgx.ErrNoRows {
				v.Fail("analog_id", "The selected analog id is invalid.")
			} else {
				return err
			}
		}
	}
	typ, _ := v.String("type", validate.Str{Required: true, In: models.AnalogTypes})
	note, _ := v.OptString("note", validate.Str{Max: 500})
	if err := v.Err(); err != nil {
		return err
	}

	if err := db.Tx(ctx, h.app.DB, func(tx pgx.Tx) error { return LinkAnalogs(r, tx, productID, analogID, typ, note) }); err != nil {
		return err
	}
	list, err := h.analogsOf(r, productID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, list)
	return nil
}

func (h *Handlers) linkedPair(r *http.Request) (productID, analogID int64, err error) {
	if productID, err = pathID(r, "product"); err != nil {
		return
	}
	if analogID, err = pathID(r, "analog"); err != nil {
		return
	}
	if _, err = services.FindProduct(r.Context(), h.app.DB, productID); err != nil {
		return
	}
	if _, err = services.FindProduct(r.Context(), h.app.DB, analogID); err != nil {
		return
	}
	linked, err := services.Exists(r.Context(), h.app.DB, "product_analogs", "product_id = $1 AND analog_id = $2", productID, analogID)
	if err != nil {
		return
	}
	if !linked {
		err = httpx.NotFound()
	}
	return
}

func (h *Handlers) AdminAnalogsUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	productID, analogID, err := h.linkedPair(r)
	if err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	if v.Filled("analog_id") {
		v.Fail("analog_id", "The analog id field is prohibited.")
	}
	typ, _ := v.String("type", validate.Str{Required: true, In: models.AnalogTypes})
	note, _ := v.OptString("note", validate.Str{Max: 500})
	if err := v.Err(); err != nil {
		return err
	}

	if err := db.Tx(ctx, h.app.DB, func(tx pgx.Tx) error { return LinkAnalogs(r, tx, productID, analogID, typ, note) }); err != nil {
		return err
	}
	list, err := h.analogsOf(r, productID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list)
	return nil
}

func (h *Handlers) AdminAnalogsDestroy(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	productID, analogID, err := h.linkedPair(r)
	if err != nil {
		return err
	}
	if _, err := h.app.DB.Exec(ctx, `
		DELETE FROM product_analogs WHERE (product_id = $1 AND analog_id = $2) OR (product_id = $2 AND analog_id = $1)`,
		productID, analogID); err != nil {
		return err
	}
	list, err := h.analogsOf(r, productID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list)
	return nil
}
