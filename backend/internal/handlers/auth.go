package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

// withCompany attaches the full company relation (User::load('company')).
func (h *Handlers) withCompany(r *http.Request, user *models.User) error {
	if user.CompanyID == nil {
		user.Company = models.Rel[models.Company](nil)
		return nil
	}
	company, err := services.FindCompany(r.Context(), h.app.DB, *user.CompanyID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	user.Company = models.Rel(company)
	return nil
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) error {
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)

	name, _ := v.String("name", validate.Str{Required: true, Max: 255})
	email, emailOK := v.String("email", validate.Str{Required: true, Email: true})
	password, _ := v.String("password", validate.Str{Required: true, Min: 8})
	typ, _ := v.String("type", validate.Str{Nullable: true, In: models.AccountTypes})
	phone, _ := v.OptString("phone", validate.Str{Max: 50})
	locale, _ := v.String("locale", validate.Str{Nullable: true, Max: 5})
	currency, _ := v.String("currency", validate.Str{Nullable: true, Size: 3})
	companyName, _ := v.String("company_name", validate.Str{Required: typ == models.TypeB2B, Nullable: true, Max: 255})
	taxNumber, _ := v.OptString("company_tax_number", validate.Str{Max: 100})
	country, _ := v.OptString("company_country", validate.Str{Size: 2})

	if emailOK {
		exists, err := services.Exists(r.Context(), h.app.DB, "users", "email = $1", email)
		if err != nil {
			return err
		}
		if exists {
			v.Fail("email", "The email has already been taken.")
		}
	}
	if err := v.Err(); err != nil {
		return err
	}

	user, err := h.app.Account.Register(r.Context(), services.RegisterInput{
		Name: name, Email: email, Password: password, Type: typ, Phone: phone,
		Locale: locale, Currency: strings.ToUpper(currency), CompanyName: companyName,
		CompanyTaxNumber: taxNumber, CompanyCountry: country,
	})
	if err != nil {
		return err
	}
	if err := h.withCompany(r, user); err != nil {
		return err
	}
	token, err := auth.IssueToken(r.Context(), h.app.DB, user.ID, "api")
	if err != nil {
		return err
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"user": user, "token": token})
	return nil
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) error {
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	email, _ := v.String("email", validate.Str{Required: true, Email: true})
	password, _ := v.String("password", validate.Str{Required: true})
	if err := v.Err(); err != nil {
		return err
	}

	user, err := services.FindUserByEmail(r.Context(), h.app.DB, email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if user == nil || !auth.CheckPassword(user.Password, password) {
		v.Fail("email", "The provided credentials are incorrect.")
		return v.Err()
	}
	if !user.IsActive {
		v.Fail("email", "This account is disabled.")
		return v.Err()
	}

	if err := h.withCompany(r, user); err != nil {
		return err
	}
	token, err := auth.IssueToken(r.Context(), h.app.DB, user.ID, "api")
	if err != nil {
		return err
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"user": user, "token": token})
	return nil
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) error {
	user := User(r)
	if err := h.withCompany(r, user); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, user)
	return nil
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) error {
	if id := tokenID(r); id != 0 {
		if err := auth.DeleteToken(r.Context(), h.app.DB, id); err != nil {
			return err
		}
	}
	httpx.Message(w, http.StatusOK, "Logged out.")
	return nil
}
