package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

// ---- Categories -------------------------------------------------------------

func (h *Handlers) AdminCategoriesIndex(w http.ResponseWriter, r *http.Request) error {
	cats, err := services.Collect[models.Category](h.app.DB.Query(r.Context(),
		"SELECT "+db.Columns(models.CategoryColumns, "categories")+
			", (SELECT COUNT(*) FROM products p WHERE p.category_id = categories.id) AS products_count"+
			" FROM categories ORDER BY position, name, id"))
	if err != nil {
		return err
	}
	if cats == nil {
		cats = []models.Category{}
	}
	httpx.JSON(w, http.StatusOK, cats)
	return nil
}

type categoryInput struct {
	name, slug          *string
	parentID            *int64
	parentPresent       bool
	translations        map[string]any
	translationsPresent bool
	position            *int64
	isActive            *bool
}

func (h *Handlers) validateCategory(r *http.Request, v *validate.V, creating bool, ignoreID int64) (categoryInput, error) {
	ctx := r.Context()
	var in categoryInput
	if s, ok := v.String("name", validate.Str{Required: creating, Max: 255}); ok {
		in.name = &s
	}
	if s, present := v.OptString("slug", validate.Str{}); present && s != nil {
		taken, err := services.Exists(ctx, h.app.DB, "categories", "slug = $1 AND id <> $2", *s, ignoreID)
		if err != nil {
			return in, err
		}
		if taken {
			v.Fail("slug", "The slug has already been taken.")
		} else {
			in.slug = s
		}
	}
	if id, present := v.OptInt("parent_id", validate.Num{}); present {
		in.parentPresent = true
		if id != nil {
			exists, err := services.Exists(ctx, h.app.DB, "categories", "id = $1", *id)
			if err != nil {
				return in, err
			}
			if !exists {
				v.Fail("parent_id", "The selected parent id is invalid.")
				in.parentPresent = false
			}
		}
		in.parentID = id
	}
	if v.Has("name_translations") {
		in.translationsPresent = true
		if obj, ok := v.Object("name_translations", true); ok {
			in.translations = obj
		} else if v.Get("name_translations") != nil {
			in.translationsPresent = false
		}
	}
	if n, present := v.OptInt("position", validate.Num{}); present && n != nil {
		in.position = n
	}
	if b, ok := v.Bool("is_active", false); ok {
		in.isActive = &b
	}
	return in, nil
}

func (h *Handlers) AdminCategoriesStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateCategory(r, v, true, 0)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}

	slug := strx.Slug(*in.name)
	if in.slug != nil {
		slug = *in.slug
	}
	position, active := int64(0), true
	if in.position != nil {
		position = *in.position
	}
	if in.isActive != nil {
		active = *in.isActive
	}
	var translations any
	if in.translations != nil {
		translations = in.translations
	}

	now := time.Now().UTC()
	var id int64
	if err := h.app.DB.QueryRow(ctx, `
		INSERT INTO categories (parent_id, slug, name, name_translations, position, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7) RETURNING id`,
		in.parentID, slug, *in.name, translations, position, active, now).Scan(&id); err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Unprocessable("The slug has already been taken.")
		}
		return err
	}
	cat, err := services.FindCategory(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, cat)
	return nil
}

func (h *Handlers) AdminCategoriesUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "category")
	if err != nil {
		return err
	}
	if _, err := services.FindCategory(ctx, h.app.DB, id); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateCategory(r, v, false, id)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}

	set := &db.Set{}
	if in.name != nil {
		set.Add("name", *in.name)
	}
	if in.slug != nil {
		set.Add("slug", *in.slug)
	}
	if in.parentPresent {
		set.Add("parent_id", in.parentID)
	}
	if in.translationsPresent {
		var translations any
		if in.translations != nil {
			translations = in.translations
		}
		set.Add("name_translations", translations)
	}
	if in.position != nil {
		set.Add("position", *in.position)
	}
	if in.isActive != nil {
		set.Add("is_active", *in.isActive)
	}
	if !set.Empty() {
		set.Add("updated_at", time.Now().UTC())
		args := append(set.Args(), id)
		if _, err := h.app.DB.Exec(ctx, "UPDATE categories SET "+set.SQL(0)+" WHERE id = $"+strconv.Itoa(len(args)), args...); err != nil {
			if db.IsUniqueViolation(err) {
				return httpx.Unprocessable("The slug has already been taken.")
			}
			return err
		}
	}
	cat, err := services.FindCategory(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, cat)
	return nil
}

func (h *Handlers) AdminCategoriesDestroy(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "category")
	if err != nil {
		return err
	}
	if _, err := services.FindCategory(ctx, h.app.DB, id); err != nil {
		return err
	}
	if _, err := h.app.DB.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id); err != nil {
		return err
	}
	httpx.Message(w, http.StatusOK, "Category deleted.")
	return nil
}

// ---- Currencies -------------------------------------------------------------

func (h *Handlers) AdminCurrenciesIndex(w http.ResponseWriter, r *http.Request) error {
	rows, err := services.Collect[models.Currency](h.app.DB.Query(r.Context(),
		"SELECT "+db.Columns(models.CurrencyColumns, "")+" FROM currencies ORDER BY is_base DESC, code"))
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []models.Currency{}
	}
	httpx.JSON(w, http.StatusOK, rows)
	return nil
}

type currencyInput struct {
	code, name, symbol *string
	rate               *float64
	isBase, isActive   *bool
}

func (h *Handlers) validateCurrency(r *http.Request, v *validate.V, creating bool, ignoreID int64) (currencyInput, error) {
	var in currencyInput
	if s, ok := v.String("code", validate.Str{Required: creating, Size: 3}); ok {
		upper := strings.ToUpper(s)
		taken, err := services.Exists(r.Context(), h.app.DB, "currencies", "UPPER(code) = $1 AND id <> $2", upper, ignoreID)
		if err != nil {
			return in, err
		}
		if taken {
			v.Fail("code", "The code has already been taken.")
		} else {
			in.code = &upper
		}
	}
	if s, ok := v.String("name", validate.Str{Required: creating, Max: 255}); ok {
		in.name = &s
	}
	if s, ok := v.String("symbol", validate.Str{Required: creating, Max: 8}); ok {
		in.symbol = &s
	}
	if f, ok := v.Float("exchange_rate", validate.Num{Required: creating, Min: validate.F(0)}); ok {
		in.rate = &f
	}
	if b, ok := v.Bool("is_base", false); ok {
		in.isBase = &b
	}
	if b, ok := v.Bool("is_active", false); ok {
		in.isActive = &b
	}
	return in, nil
}

// normalizeBase ensures only one base currency exists, pinned to rate 1.0.
func (h *Handlers) normalizeBase(r *http.Request, in *currencyInput) error {
	if in.isBase != nil && *in.isBase {
		if _, err := h.app.DB.Exec(r.Context(), `UPDATE currencies SET is_base = false WHERE is_base = true`); err != nil {
			return err
		}
		in.rate = services.Ptr(1.0)
	}
	return nil
}

func (h *Handlers) AdminCurrenciesStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateCurrency(r, v, true, 0)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}
	if err := h.normalizeBase(r, &in); err != nil {
		return err
	}
	isBase, isActive := false, true
	if in.isBase != nil {
		isBase = *in.isBase
	}
	if in.isActive != nil {
		isActive = *in.isActive
	}

	now := time.Now().UTC()
	var id int64
	if err := h.app.DB.QueryRow(ctx, `
		INSERT INTO currencies (code, name, symbol, exchange_rate, is_base, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7) RETURNING id`,
		*in.code, *in.name, *in.symbol, *in.rate, isBase, isActive, now).Scan(&id); err != nil {
		return err
	}
	h.app.Currency.Flush()
	currency, err := services.FindCurrency(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, currency)
	return nil
}

func (h *Handlers) AdminCurrenciesUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "currency")
	if err != nil {
		return err
	}
	if _, err := services.FindCurrency(ctx, h.app.DB, id); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateCurrency(r, v, false, id)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}
	if err := h.normalizeBase(r, &in); err != nil {
		return err
	}

	set := &db.Set{}
	if in.code != nil {
		set.Add("code", *in.code)
	}
	if in.name != nil {
		set.Add("name", *in.name)
	}
	if in.symbol != nil {
		set.Add("symbol", *in.symbol)
	}
	if in.rate != nil {
		set.Add("exchange_rate", *in.rate)
	}
	if in.isBase != nil {
		set.Add("is_base", *in.isBase)
	}
	if in.isActive != nil {
		set.Add("is_active", *in.isActive)
	}
	if !set.Empty() {
		set.Add("updated_at", time.Now().UTC())
		args := append(set.Args(), id)
		if _, err := h.app.DB.Exec(ctx, "UPDATE currencies SET "+set.SQL(0)+" WHERE id = $"+strconv.Itoa(len(args)), args...); err != nil {
			return err
		}
	}
	h.app.Currency.Flush()
	currency, err := services.FindCurrency(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, currency)
	return nil
}

func (h *Handlers) AdminCurrenciesDestroy(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "currency")
	if err != nil {
		return err
	}
	currency, err := services.FindCurrency(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	if currency.IsBase {
		return httpx.Unprocessable("Cannot delete the base currency.")
	}
	if _, err := h.app.DB.Exec(ctx, `DELETE FROM currencies WHERE id = $1`, id); err != nil {
		return err
	}
	h.app.Currency.Flush()
	httpx.Message(w, http.StatusOK, "Currency deleted.")
	return nil
}

// ---- Companies --------------------------------------------------------------

func companyQuery(where string) string {
	return "SELECT " + db.Columns(models.CompanyColumns, "companies") +
		", (SELECT COUNT(*) FROM users u WHERE u.company_id = companies.id) AS users_count" +
		", (SELECT COUNT(*) FROM products p WHERE p.company_id = companies.id) AS products_count" +
		", (SELECT COUNT(*) FROM orders o WHERE o.company_id = companies.id) AS orders_count" +
		" FROM companies" + where
}

func (h *Handlers) AdminCompaniesIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := &db.Query{}
	q.ILike(httpx.Query(r, "search"), "companies.name")
	where := q.SQL()

	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM companies"+where, q.Args()...)
	if err != nil {
		return err
	}
	page, perPage := httpx.PageParams(r, 20, 20)
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	companies, err := services.Collect[models.Company](h.app.DB.Query(ctx,
		companyQuery(where)+" ORDER BY companies.created_at DESC, companies.id DESC LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, httpx.NewPaginator(r, companies, total, page, perPage))
	return nil
}

func (h *Handlers) AdminCompaniesUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "company")
	if err != nil {
		return err
	}
	if _, err := services.FindCompany(ctx, h.app.DB, id); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	set := &db.Set{}

	if s, ok := v.String("name", validate.Str{Max: 255}); ok {
		set.Add("name", s)
	}
	nullable := []struct {
		key  string
		rule validate.Str
	}{
		{"tax_number", validate.Str{Max: 100}},
		{"email", validate.Str{Email: true}},
		{"phone", validate.Str{Max: 50}},
		{"country", validate.Str{Size: 2}},
		{"address", validate.Str{}},
	}
	for _, f := range nullable {
		if s, present := v.OptString(f.key, f.rule); present {
			set.Add(f.key, s)
		}
	}
	if s, present := v.OptString("default_currency", validate.Str{Size: 3}); present && s != nil {
		set.Add("default_currency", strings.ToUpper(*s))
	}
	if s, present := v.OptString("default_locale", validate.Str{Max: 5}); present && s != nil {
		set.Add("default_locale", *s)
	}
	if b, ok := v.Bool("is_verified", false); ok {
		set.Add("is_verified", b)
	}
	if err := v.Err(); err != nil {
		return err
	}

	if !set.Empty() {
		set.Add("updated_at", time.Now().UTC())
		args := append(set.Args(), id)
		if _, err := h.app.DB.Exec(ctx, "UPDATE companies SET "+set.SQL(0)+" WHERE id = $"+strconv.Itoa(len(args)), args...); err != nil {
			return err
		}
	}
	company, err := services.FindCompany(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, company)
	return nil
}
