// Package services contains the business logic and persistence queries.
package services

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/config"
	"github.com/asihat/b2b-marketplace/backend/internal/payments"
)

// App wires every service together; handlers receive one *App.
type App struct {
	Cfg       config.Config
	DB        *pgxpool.Pool
	Disk      *PublicDisk
	Currency  *CurrencyService
	Settings  *SettingsService
	Locale    *LocaleService
	Payments  *payments.Manager
	Orders    *OrderService
	Account   *AccountService
	Dashboard *DashboardService
}

func NewApp(cfg config.Config, pool *pgxpool.Pool) *App {
	disk := NewPublicDisk(cfg.StoragePath, cfg.AppURL)
	currency := NewCurrencyService(pool, cfg.BaseCurrency)
	pm := payments.NewManager(cfg.PaymentGateway, payments.Fake{}, payments.Manual{BankAccount: cfg.ManualBankAccount})

	return &App{
		Cfg:       cfg,
		DB:        pool,
		Disk:      disk,
		Currency:  currency,
		Settings:  NewSettingsService(pool, cfg, disk),
		Locale:    NewLocaleService(pool, cfg.Locale),
		Payments:  pm,
		Orders:    NewOrderService(pool, currency, pm),
		Account:   NewAccountService(pool, cfg.BcryptCost),
		Dashboard: NewDashboardService(pool, currency),
	}
}

// FlushCaches drops every in-memory cache (after seeding, for instance).
func (a *App) FlushCaches() {
	a.Currency.Flush()
	a.Settings.Flush()
	a.Locale.Flush()
}
