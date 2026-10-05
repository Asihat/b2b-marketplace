package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

// attachAdminProductRelations loads category:id,name, company:id,name,
// images and (optionally) prices for a set of products.
func (h *Handlers) attachAdminProductRelations(r *http.Request, products []models.Product, withPrices bool) error {
	ctx := r.Context()
	ids := services.IDs(products, func(p models.Product) *int64 { return &p.ID })

	categories, err := services.LoadNamedRefs(ctx, h.app.DB, "categories", services.IDs(products, func(p models.Product) *int64 { return p.CategoryID }))
	if err != nil {
		return err
	}
	companies, err := services.LoadNamedRefs(ctx, h.app.DB, "companies", services.IDs(products, func(p models.Product) *int64 { return p.CompanyID }))
	if err != nil {
		return err
	}
	images, err := services.LoadImages(ctx, h.app.DB, ids)
	if err != nil {
		return err
	}
	var prices map[int64][]models.ProductPrice
	if withPrices {
		if prices, err = services.LoadPrices(ctx, h.app.DB, ids); err != nil {
			return err
		}
	}

	for i := range products {
		p := &products[i]
		var cat, comp *models.NamedRef
		if p.CategoryID != nil {
			cat = categories[*p.CategoryID]
		}
		if p.CompanyID != nil {
			comp = companies[*p.CompanyID]
		}
		p.Category = models.Rel(cat)
		p.Company = models.Rel(comp)
		p.Images = models.Slice(images[p.ID])
		if withPrices {
			p.Prices = models.Slice(prices[p.ID])
		}
	}
	return nil
}

func adminProductQuery(where string) string {
	return "SELECT " + db.Columns(models.ProductColumns, "products") +
		", (SELECT COUNT(*) FROM product_analogs pa WHERE pa.product_id = products.id) AS analogs_count FROM products" + where
}

func (h *Handlers) AdminProductsIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := &db.Query{}
	q.ILike(httpx.Query(r, "search"), "products.name", "products.sku", "products.brand")
	if id, ok := httpx.QueryInt64(r, "category_id"); ok && id > 0 {
		q.Wheref("products.category_id = %s", id)
	}
	if httpx.Filled(r, "is_active") {
		q.Wheref("products.is_active = %s", httpx.Truthy(httpx.Query(r, "is_active")))
	}
	if httpx.Filled(r, "is_b2b_only") {
		q.Wheref("products.is_b2b_only = %s", httpx.Truthy(httpx.Query(r, "is_b2b_only")))
	}
	if id, ok := httpx.QueryInt64(r, "exclude_id"); ok && id > 0 {
		q.Wheref("products.id <> %s", id)
	}
	where := q.SQL()

	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM products"+where, q.Args()...)
	if err != nil {
		return err
	}
	page, perPage := httpx.PageParams(r, 20, 100)
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	products, err := services.Collect[models.Product](h.app.DB.Query(ctx,
		adminProductQuery(where)+" ORDER BY products.created_at DESC, products.id DESC LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}
	if err := h.attachAdminProductRelations(r, products, false); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, httpx.NewPaginator(r, products, total, page, perPage))
	return nil
}

func (h *Handlers) adminProductDetail(r *http.Request, id int64) (*models.Product, error) {
	product, err := services.One[models.Product](h.app.DB.Query(r.Context(), adminProductQuery(" WHERE products.id = $1"), id))
	if err != nil {
		return nil, err
	}
	list := []models.Product{*product}
	if err := h.attachAdminProductRelations(r, list, true); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (h *Handlers) AdminProductsShow(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	product, err := h.adminProductDetail(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, product)
	return nil
}

// productInput is the validated ProductRequest payload. Pointer fields are
// nil when the key was omitted; the *Present flags distinguish "omitted"
// from "sent as null" for nullable columns.
type productInput struct {
	sku, slug, name       *string
	description, brand    *string
	descriptionPresent    bool
	brandPresent          bool
	unit                  *string
	categoryID, companyID *int64
	categoryPresent       bool
	companyPresent        bool
	basePrice             *float64
	stock, minOrderQty    *int64
	isB2BOnly, isActive   *bool
	images                []string
	imagesPresent         bool
}

func (h *Handlers) validateProduct(r *http.Request, v *validate.V, creating bool, ignoreID int64) (productInput, error) {
	ctx := r.Context()
	var in productInput

	if s, ok := v.String("sku", validate.Str{Required: creating}); ok {
		taken, err := services.Exists(ctx, h.app.DB, "products", "sku = $1 AND id <> $2", s, ignoreID)
		if err != nil {
			return in, err
		}
		if taken {
			v.Fail("sku", "The sku has already been taken.")
		} else {
			in.sku = &s
		}
	}
	if s, present := v.OptString("slug", validate.Str{}); present && s != nil {
		taken, err := services.Exists(ctx, h.app.DB, "products", "slug = $1 AND id <> $2", *s, ignoreID)
		if err != nil {
			return in, err
		}
		if taken {
			v.Fail("slug", "The slug has already been taken.")
		} else {
			in.slug = s
		}
	}
	if s, ok := v.String("name", validate.Str{Required: creating, Max: 255}); ok {
		in.name = &s
	}
	in.description, in.descriptionPresent = v.OptString("description", validate.Str{})
	in.brand, in.brandPresent = v.OptString("brand", validate.Str{Max: 255})
	if s, present := v.OptString("unit", validate.Str{Max: 50}); present && s != nil {
		in.unit = s
	}
	if id, present := v.OptInt("category_id", validate.Num{}); present {
		in.categoryPresent = true
		if id != nil {
			exists, err := services.Exists(ctx, h.app.DB, "categories", "id = $1", *id)
			if err != nil {
				return in, err
			}
			if !exists {
				v.Fail("category_id", "The selected category id is invalid.")
				in.categoryPresent = false
			}
		}
		in.categoryID = id
	}
	if id, present := v.OptInt("company_id", validate.Num{}); present {
		in.companyPresent = true
		if id != nil {
			exists, err := services.Exists(ctx, h.app.DB, "companies", "id = $1", *id)
			if err != nil {
				return in, err
			}
			if !exists {
				v.Fail("company_id", "The selected company id is invalid.")
				in.companyPresent = false
			}
		}
		in.companyID = id
	}
	if f, ok := v.Float("base_price", validate.Num{Required: creating, Min: validate.F(0)}); ok {
		in.basePrice = &f
	}
	if n, present := v.OptInt("stock", validate.Num{Min: validate.F(0)}); present && n != nil {
		in.stock = n
	}
	if n, present := v.OptInt("min_order_qty", validate.Num{Min: validate.F(1)}); present && n != nil {
		in.minOrderQty = n
	}
	if b, ok := v.Bool("is_b2b_only", false); ok {
		in.isB2BOnly = &b
	}
	if b, ok := v.Bool("is_active", false); ok {
		in.isActive = &b
	}
	if list, ok := v.Array("images", validate.Arr{Nullable: true}); ok {
		in.imagesPresent = true
		for i, raw := range list {
			if raw == nil {
				continue
			}
			s, isStr := raw.(string)
			if !isStr || len(s) > 2048 {
				v.Fail("images."+strconv.Itoa(i), "The images."+strconv.Itoa(i)+" field must be a string with a maximum of 2048 characters.")
				continue
			}
			in.images = append(in.images, s)
		}
	}
	return in, nil
}

// syncImages replaces a product's images from a list of URLs (first = primary).
func (h *Handlers) syncImages(r *http.Request, tx db.Querier, product *models.Product, urls []string) error {
	ctx := r.Context()
	if _, err := tx.Exec(ctx, `DELETE FROM product_images WHERE product_id = $1`, product.ID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, u := range urls {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_images (product_id, url, alt, position, is_primary, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)`, product.ID, u, product.Name, i+1, i == 0, now); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) AdminProductsStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateProduct(r, v, true, 0)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}

	slug := strx.Slug(*in.sku)
	if in.slug != nil {
		slug = *in.slug
	}
	unit, stock, moq := "pcs", int64(0), int64(1)
	if in.unit != nil {
		unit = *in.unit
	}
	if in.stock != nil {
		stock = *in.stock
	}
	if in.minOrderQty != nil {
		moq = *in.minOrderQty
	}
	b2bOnly, active := false, true
	if in.isB2BOnly != nil {
		b2bOnly = *in.isB2BOnly
	}
	if in.isActive != nil {
		active = *in.isActive
	}

	var id int64
	err = db.Tx(ctx, h.app.DB, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		if err := tx.QueryRow(ctx, `
			INSERT INTO products (category_id, company_id, sku, slug, name, description, brand, unit, base_price, stock, min_order_qty, is_b2b_only, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14) RETURNING id`,
			in.categoryID, in.companyID, *in.sku, slug, *in.name, in.description, in.brand, unit, *in.basePrice, stock, moq, b2bOnly, active, now,
		).Scan(&id); err != nil {
			return err
		}
		if in.imagesPresent {
			return h.syncImages(r, tx, &models.Product{ID: id, Name: *in.name}, in.images)
		}
		return nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Unprocessable("The slug has already been taken.")
		}
		return err
	}

	product, err := h.adminProductDetail(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, product)
	return nil
}

func (h *Handlers) AdminProductsUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	existing, err := services.FindProduct(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateProduct(r, v, false, id)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}

	set := &db.Set{}
	if in.sku != nil {
		set.Add("sku", *in.sku)
	}
	if in.slug != nil {
		set.Add("slug", *in.slug)
	}
	if in.name != nil {
		set.Add("name", *in.name)
		existing.Name = *in.name
	}
	if in.descriptionPresent {
		set.Add("description", in.description)
	}
	if in.brandPresent {
		set.Add("brand", in.brand)
	}
	if in.unit != nil {
		set.Add("unit", *in.unit)
	}
	if in.categoryPresent {
		set.Add("category_id", in.categoryID)
	}
	if in.companyPresent {
		set.Add("company_id", in.companyID)
	}
	if in.basePrice != nil {
		set.Add("base_price", *in.basePrice)
	}
	if in.stock != nil {
		set.Add("stock", *in.stock)
	}
	if in.minOrderQty != nil {
		set.Add("min_order_qty", *in.minOrderQty)
	}
	if in.isB2BOnly != nil {
		set.Add("is_b2b_only", *in.isB2BOnly)
	}
	if in.isActive != nil {
		set.Add("is_active", *in.isActive)
	}

	err = db.Tx(ctx, h.app.DB, func(tx pgx.Tx) error {
		if !set.Empty() {
			set.Add("updated_at", time.Now().UTC())
			args := append(set.Args(), id)
			if _, err := tx.Exec(ctx, "UPDATE products SET "+set.SQL(0)+" WHERE id = $"+strconv.Itoa(len(args)), args...); err != nil {
				return err
			}
		}
		if in.imagesPresent {
			return h.syncImages(r, tx, existing, in.images)
		}
		return nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Unprocessable("The slug has already been taken.")
		}
		return err
	}

	product, err := h.adminProductDetail(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, product)
	return nil
}

func (h *Handlers) AdminProductsDestroy(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(ctx, h.app.DB, id); err != nil {
		return err
	}
	if _, err := h.app.DB.Exec(ctx, `DELETE FROM products WHERE id = $1`, id); err != nil {
		return err
	}
	httpx.Message(w, http.StatusOK, "Product deleted.")
	return nil
}
