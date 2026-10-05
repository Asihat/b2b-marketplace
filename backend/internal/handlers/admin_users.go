package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

func (h *Handlers) attachCompanyRefs(r *http.Request, users []models.User) error {
	refs, err := services.LoadNamedRefs(r.Context(), h.app.DB, "companies", services.IDs(users, func(u models.User) *int64 { return u.CompanyID }))
	if err != nil {
		return err
	}
	for i := range users {
		var ref *models.NamedRef
		if users[i].CompanyID != nil {
			ref = refs[*users[i].CompanyID]
		}
		users[i].Company = models.Rel(ref)
	}
	return nil
}

func (h *Handlers) AdminUsersIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := &db.Query{}
	q.ILike(httpx.Query(r, "search"), "name", "email")
	if t := httpx.Query(r, "type"); t != "" {
		q.Wheref("type = %s", t)
	}
	if role := httpx.Query(r, "role"); role != "" {
		q.Wheref("role = %s", role)
	}
	where := q.SQL()

	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM users"+where, q.Args()...)
	if err != nil {
		return err
	}
	page, perPage := httpx.PageParams(r, 20, 20)
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	users, err := services.Collect[models.User](h.app.DB.Query(ctx,
		"SELECT "+db.Columns(models.UserColumns, "")+" FROM users"+where+" ORDER BY created_at DESC, id DESC LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}
	if err := h.attachCompanyRefs(r, users); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, httpx.NewPaginator(r, users, total, page, perPage))
	return nil
}

// userInput validates the UserRequest rules; creating toggles required-ness.
type userInput struct {
	name, email, typ, role *string
	password               *string
	companyID              *int64
	companyPresent         bool
	currency, locale       *string
	currencyPresent        bool
	localePresent          bool
	isActive               *bool
}

func (h *Handlers) validateUser(r *http.Request, v *validate.V, creating bool, ignoreID int64) (userInput, error) {
	var in userInput
	ctx := r.Context()

	if s, ok := v.String("name", validate.Str{Required: creating, Max: 255}); ok {
		in.name = &s
	}
	if s, ok := v.String("email", validate.Str{Required: creating, Email: true}); ok {
		taken, err := services.Exists(ctx, h.app.DB, "users", "email = $1 AND id <> $2", s, ignoreID)
		if err != nil {
			return in, err
		}
		if taken {
			v.Fail("email", "The email has already been taken.")
		} else {
			in.email = &s
		}
	}
	if s, ok := v.String("password", validate.Str{Required: creating, Nullable: !creating, Min: 8}); ok {
		in.password = &s
	}
	if s, ok := v.String("type", validate.Str{Required: creating, In: models.AccountTypes}); ok {
		in.typ = &s
	}
	if s, ok := v.String("role", validate.Str{Required: creating, In: models.UserRoles}); ok {
		in.role = &s
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
	if s, present := v.OptString("currency", validate.Str{Size: 3}); present {
		in.currencyPresent = true
		if s != nil {
			upper := strings.ToUpper(*s)
			s = &upper
		}
		in.currency = s
	}
	if s, present := v.OptString("locale", validate.Str{Max: 5}); present {
		in.localePresent = true
		in.locale = s
	}
	if b, ok := v.Bool("is_active", false); ok {
		in.isActive = &b
	}
	return in, nil
}

func (h *Handlers) AdminUsersStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateUser(r, v, true, 0)
	if err != nil {
		return err
	}
	if err := v.Err(); err != nil {
		return err
	}

	hash, err := auth.HashPassword(*in.password, h.app.Cfg.BcryptCost)
	if err != nil {
		return err
	}
	currency, locale, active := "USD", "en", true
	if in.currency != nil {
		currency = *in.currency
	}
	if in.locale != nil {
		locale = *in.locale
	}
	if in.isActive != nil {
		active = *in.isActive
	}

	now := time.Now().UTC()
	var id int64
	if err := h.app.DB.QueryRow(ctx, `
		INSERT INTO users (company_id, name, email, password, type, role, locale, currency, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10) RETURNING id`,
		in.companyID, *in.name, *in.email, hash, *in.typ, *in.role, locale, currency, active, now).Scan(&id); err != nil {
		return err
	}

	user, err := services.FindUser(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	users := []models.User{*user}
	if err := h.attachCompanyRefs(r, users); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, users[0])
	return nil
}

func (h *Handlers) AdminUsersUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "user")
	if err != nil {
		return err
	}
	if _, err := services.FindUser(ctx, h.app.DB, id); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	in, err := h.validateUser(r, v, false, id)
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
	if in.email != nil {
		set.Add("email", *in.email)
	}
	if in.password != nil {
		hash, err := auth.HashPassword(*in.password, h.app.Cfg.BcryptCost)
		if err != nil {
			return err
		}
		set.Add("password", hash)
	}
	if in.typ != nil {
		set.Add("type", *in.typ)
	}
	if in.role != nil {
		set.Add("role", *in.role)
	}
	if in.companyPresent {
		set.Add("company_id", in.companyID)
	}
	if in.currencyPresent && in.currency != nil {
		set.Add("currency", *in.currency)
	}
	if in.localePresent && in.locale != nil {
		set.Add("locale", *in.locale)
	}
	if in.isActive != nil {
		set.Add("is_active", *in.isActive)
	}
	if !set.Empty() {
		set.Add("updated_at", time.Now().UTC())
		args := append(set.Args(), id)
		if _, err := h.app.DB.Exec(ctx, "UPDATE users SET "+set.SQL(0)+" WHERE id = $"+itoa(len(args)), args...); err != nil {
			return err
		}
	}

	user, err := services.FindUser(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	users := []models.User{*user}
	if err := h.attachCompanyRefs(r, users); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, users[0])
	return nil
}

func (h *Handlers) AdminUsersDestroy(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "user")
	if err != nil {
		return err
	}
	if _, err := services.FindUser(ctx, h.app.DB, id); err != nil {
		return err
	}
	if id == User(r).ID {
		return httpx.Unprocessable("You cannot delete your own account.")
	}
	if _, err := h.app.DB.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return err
	}
	httpx.Message(w, http.StatusOK, "User deleted.")
	return nil
}
