package services

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

// RegisterInput is the validated registration payload.
type RegisterInput struct {
	Name             string
	Email            string
	Password         string
	Type             string
	Phone            *string
	Locale           string
	Currency         string
	CompanyName      string
	CompanyTaxNumber *string
	CompanyCountry   *string
}

// AccountService registers retail (B2C) and business (B2B) accounts; B2B
// registrations also create the associated company.
type AccountService struct {
	db         *pgxpool.Pool
	bcryptCost int
}

func NewAccountService(pool *pgxpool.Pool, bcryptCost int) *AccountService {
	return &AccountService{db: pool, bcryptCost: bcryptCost}
}

func (s *AccountService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	if in.Type == "" {
		in.Type = models.TypeB2C
	}
	if in.Locale == "" {
		in.Locale = "en"
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	hash, err := auth.HashPassword(in.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}

	var user *models.User
	err = db.Tx(ctx, s.db, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		var companyID *int64

		if in.Type == models.TypeB2B {
			var id int64
			err := tx.QueryRow(ctx, `
				INSERT INTO companies (name, slug, tax_number, country, default_currency, default_locale, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $7) RETURNING id`,
				in.CompanyName,
				strx.Slug(in.CompanyName)+"-"+strings.ToLower(strx.Random(5)),
				in.CompanyTaxNumber, in.CompanyCountry, in.Currency, in.Locale, now,
			).Scan(&id)
			if err != nil {
				return err
			}
			companyID = &id
		}

		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO users (company_id, name, email, password, type, role, phone, locale, currency, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $10) RETURNING id`,
			companyID, in.Name, in.Email, hash, in.Type, models.RoleCustomer, in.Phone, in.Locale, in.Currency, now,
		).Scan(&id)
		if err != nil {
			return err
		}

		user, err = FindUser(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}
