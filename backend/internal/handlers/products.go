package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

// ---- Resources (ProductResource / AnalogResource) ---------------------------

type priceOut struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Formatted string  `json:"formatted,omitempty"`
}

type imageOut struct {
	URL       string  `json:"url"`
	Alt       *string `json:"alt"`
	IsPrimary bool    `json:"is_primary"`
}

type relationOut struct {
	Type string  `json:"type"`
	Note *string `json:"note"`
}

type analogOut struct {
	ID       int64        `json:"id"`
	SKU      string       `json:"sku"`
	Slug     string       `json:"slug"`
	Name     string       `json:"name"`
	Brand    *string      `json:"brand"`
	Stock    int32        `json:"stock"`
	Image    *string      `json:"image"`
	Relation *relationOut `json:"relation,omitempty"`
	Price    priceOut     `json:"price"`
}

type productOut struct {
	ID           int64                       `json:"id"`
	SKU          string                      `json:"sku"`
	Slug         string                      `json:"slug"`
	Name         string                      `json:"name"`
	Description  *string                     `json:"description"`
	Brand        *string                     `json:"brand"`
	Unit         string                      `json:"unit"`
	Stock        int32                       `json:"stock"`
	MinOrderQty  int32                       `json:"min_order_qty"`
	IsB2BOnly    bool                        `json:"is_b2b_only"`
	CategoryID   *int64                      `json:"category_id"`
	CompanyID    *int64                      `json:"company_id"`
	Company      any                         `json:"company,omitempty"`
	Image        *string                     `json:"image"`
	Images       []imageOut                  `json:"images"`
	Price        priceOut                    `json:"price"`
	Analogs      any                         `json:"analogs,omitempty"`
	Translations []models.ProductTranslation `json:"translations"`
}

// catalogContext holds the per-request relations needed to render products.
type catalogContext struct {
	currency     string
	qty          int
	locale       string
	showCompany  bool
	images       map[int64][]models.ProductImage
	translations map[int64][]models.ProductTranslation
	prices       map[int64][]models.ProductPrice
	companies    map[int64]*models.CompanyBrief
}

func (h *Handlers) loadCatalogContext(r *http.Request, products []models.Product, withCompany bool) (*catalogContext, error) {
	ctx := r.Context()
	ids := services.IDs(products, func(p models.Product) *int64 { return &p.ID })

	cc := &catalogContext{
		currency:    currencyParam(r),
		qty:         qtyParam(r),
		locale:      Locale(r),
		showCompany: withCompany && h.app.Settings.ShowCompanyNames(ctx),
	}

	var err error
	if cc.images, err = services.LoadImages(ctx, h.app.DB, ids); err != nil {
		return nil, err
	}
	if cc.translations, err = services.LoadTranslations(ctx, h.app.DB, ids); err != nil {
		return nil, err
	}
	if cc.prices, err = services.LoadPrices(ctx, h.app.DB, ids); err != nil {
		return nil, err
	}
	if cc.showCompany {
		companyIDs := services.IDs(products, func(p models.Product) *int64 { return p.CompanyID })
		if cc.companies, err = services.LoadCompanyBriefs(ctx, h.app.DB, companyIDs); err != nil {
			return nil, err
		}
	}
	return cc, nil
}

func (h *Handlers) productResource(r *http.Request, cc *catalogContext, p *models.Product) productOut {
	ctx := r.Context()
	images := cc.images[p.ID]
	imgs := make([]imageOut, 0, len(images))
	for _, img := range images {
		imgs = append(imgs, imageOut{URL: img.URL, Alt: img.Alt, IsPrimary: img.IsPrimary})
	}
	translations := cc.translations[p.ID]
	if translations == nil {
		translations = []models.ProductTranslation{}
	}
	amount := h.app.Currency.PriceFor(ctx, p, cc.prices[p.ID], cc.currency, cc.qty)

	out := productOut{
		ID: p.ID, SKU: p.SKU, Slug: p.Slug,
		Name:        models.TranslatedName(p, translations, cc.locale),
		Description: p.Description, Brand: p.Brand, Unit: p.Unit, Stock: p.Stock,
		MinOrderQty: p.MinOrderQty, IsB2BOnly: p.IsB2BOnly, CategoryID: p.CategoryID, CompanyID: p.CompanyID,
		Image:  models.PrimaryImageURL(images),
		Images: imgs,
		Price: priceOut{
			Currency:  cc.currency,
			Amount:    amount,
			Formatted: h.app.Currency.Format(ctx, amount, cc.currency),
		},
		Translations: translations,
	}
	if cc.showCompany {
		var company *models.CompanyBrief
		if p.CompanyID != nil {
			company = cc.companies[*p.CompanyID]
		}
		out.Company = models.Rel(company)
	}
	return out
}

func (h *Handlers) analogResource(r *http.Request, cc *catalogContext, p *models.Product, link *models.AnalogLink) analogOut {
	images := cc.images[p.ID]
	out := analogOut{
		ID: p.ID, SKU: p.SKU, Slug: p.Slug,
		Name:  models.TranslatedName(p, cc.translations[p.ID], cc.locale),
		Brand: p.Brand, Stock: p.Stock,
		Image: models.PrimaryImageURL(images),
		Price: priceOut{
			Currency: cc.currency,
			Amount:   h.app.Currency.PriceFor(r.Context(), p, cc.prices[p.ID], cc.currency, 1),
		},
	}
	if link != nil {
		out.Relation = &relationOut{Type: link.Type, Note: link.Note}
	}
	return out
}

// ---- Endpoints ----------------------------------------------------------------

func visibleProductsWhere(q *db.Query, user *models.User) {
	q.Where("products.is_active = true")
	if user == nil || !user.IsB2B() {
		q.Where("products.is_b2b_only = false")
	}
}

// ProductsIndex is the paginated, filterable catalog. B2B-only items are
// hidden from B2C buyers.
func (h *Handlers) ProductsIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := User(r)
	currency := currencyParam(r)
	qty := qtyParam(r)

	q := &db.Query{}
	visibleProductsWhere(q, user)
	q.ILike(httpx.Query(r, "search"), "products.name", "products.sku", "products.brand")
	if id, ok := httpx.QueryInt64(r, "category_id"); ok && id > 0 {
		q.Wheref("products.category_id = %s", id)
	}
	if brand := strings.TrimSpace(httpx.Query(r, "brand")); brand != "" {
		q.Wheref("products.brand = %s", brand)
	}
	if id, ok := httpx.QueryInt64(r, "company_id"); ok && id > 0 {
		q.Wheref("products.company_id = %s", id)
	}
	if v, ok := httpx.QueryFloat(r, "min_price"); ok {
		q.Where(h.app.Currency.EffectivePriceSQL(ctx, q, currency, qty) + " >= " + q.Arg(max(0, v)))
	}
	if v, ok := httpx.QueryFloat(r, "max_price"); ok {
		q.Where(h.app.Currency.EffectivePriceSQL(ctx, q, currency, qty) + " <= " + q.Arg(max(0, v)))
	}
	if httpx.Truthy(httpx.Query(r, "in_stock")) {
		q.Where("products.stock > 0")
	}

	where := q.SQL()
	countArgs := append([]any{}, q.Args()...)
	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM products"+where, countArgs...)
	if err != nil {
		return err
	}

	direction := "ASC"
	if strings.EqualFold(httpx.Query(r, "direction"), "desc") {
		direction = "DESC"
	}
	var orderBy string
	switch httpx.Query(r, "sort") {
	case "popular":
		orderBy = "(SELECT COALESCE(SUM(oi.quantity), 0) FROM order_items oi WHERE oi.product_id = products.id) DESC, products.name ASC"
	case "base_price":
		orderBy = h.app.Currency.EffectivePriceSQL(ctx, q, currency, qty) + " " + direction + ", products.name ASC"
	case "stock":
		orderBy = "products.stock " + direction + ", products.id ASC"
	case "created_at":
		orderBy = "products.created_at " + direction + ", products.id " + direction
	default:
		orderBy = "products.name " + direction + ", products.id ASC"
	}

	page, perPage := httpx.PageParams(r, 20, 100)
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	products, err := services.Collect[models.Product](h.app.DB.Query(ctx,
		"SELECT "+db.Columns(models.ProductColumns, "products")+" FROM products"+where+
			" ORDER BY "+orderBy+" LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}

	cc, err := h.loadCatalogContext(r, products, true)
	if err != nil {
		return err
	}
	out := make([]productOut, 0, len(products))
	for i := range products {
		out = append(out, h.productResource(r, cc, &products[i]))
	}

	httpx.JSON(w, http.StatusOK, httpx.NewCollection(r, out, total, page, perPage))
	return nil
}

var digits = regexp.MustCompile(`^[0-9]+$`)

func (h *Handlers) ProductShow(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	idOrSlug := strings.TrimSpace(chi.URLParam(r, "idOrSlug"))
	if idOrSlug == "" {
		return httpx.NotFound()
	}

	q := &db.Query{}
	visibleProductsWhere(q, User(r))
	if digits.MatchString(idOrSlug) {
		id, _ := strconv.ParseInt(idOrSlug, 10, 64)
		q.Where("(products.slug = " + q.Arg(idOrSlug) + " OR products.id = " + q.Arg(id) + ")")
	} else {
		q.Wheref("products.slug = %s", idOrSlug)
	}

	product, err := services.One[models.Product](h.app.DB.Query(ctx,
		"SELECT "+db.Columns(models.ProductColumns, "products")+" FROM products"+q.SQL()+" LIMIT 1", q.Args()...))
	if err != nil {
		return err
	}

	// All linked analogs (regardless of visibility), like analogs.images eager loading.
	analogs, links, err := h.linkedAnalogs(r, product.ID, nil)
	if err != nil {
		return err
	}

	all := append([]models.Product{*product}, analogs...)
	cc, err := h.loadCatalogContext(r, all, true)
	if err != nil {
		return err
	}

	out := h.productResource(r, cc, product)
	analogList := make([]analogOut, 0, len(analogs))
	for i := range analogs {
		analogList = append(analogList, h.analogResource(r, cc, &analogs[i], links[analogs[i].ID]))
	}
	out.Analogs = analogList

	httpx.JSON(w, http.StatusOK, map[string]any{"data": out})
	return nil
}

// linkedAnalogs loads the analog products of a product plus their pivot rows.
// extraWhere narrows the analog rows (e.g. visibility for the public endpoint).
func (h *Handlers) linkedAnalogs(r *http.Request, productID int64, extraWhere func(*db.Query)) ([]models.Product, map[int64]*models.AnalogLink, error) {
	q := &db.Query{}
	q.Wheref("pa.product_id = %s", productID)
	if extraWhere != nil {
		extraWhere(q)
	}
	rows, err := h.app.DB.Query(r.Context(),
		"SELECT "+db.Columns(models.ProductColumns, "products")+", pa.type AS pivot_type, pa.note AS pivot_note"+
			" FROM products JOIN product_analogs pa ON pa.analog_id = products.id"+q.SQL()+" ORDER BY products.id", q.Args()...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	type row struct {
		models.Product
		PivotType string  `db:"pivot_type"`
		PivotNote *string `db:"pivot_note"`
	}
	scanned, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[row])
	if err != nil {
		return nil, nil, err
	}

	products := make([]models.Product, 0, len(scanned))
	links := make(map[int64]*models.AnalogLink, len(scanned))
	for _, s := range scanned {
		products = append(products, s.Product)
		links[s.ID] = &models.AnalogLink{ProductID: productID, AnalogID: s.ID, Type: s.PivotType, Note: s.PivotNote}
	}
	return products, links, nil
}

// ProductAnalogs lists interchangeable cross-references for a product.
func (h *Handlers) ProductAnalogs(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "product")
	if err != nil {
		return err
	}
	if _, err := services.FindProduct(r.Context(), h.app.DB, id); err != nil {
		return err
	}

	user := User(r)
	analogs, links, err := h.linkedAnalogs(r, id, func(q *db.Query) { visibleProductsWhere(q, user) })
	if err != nil {
		return err
	}
	cc, err := h.loadCatalogContext(r, analogs, false)
	if err != nil {
		return err
	}
	out := make([]analogOut, 0, len(analogs))
	for i := range analogs {
		out = append(out, h.analogResource(r, cc, &analogs[i], links[analogs[i].ID]))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": out})
	return nil
}
