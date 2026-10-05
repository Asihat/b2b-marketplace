// Package config reads the application configuration from the environment.
// Variable names intentionally match the former Laravel .env so an existing
// deployment can switch backends without touching its configuration.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppName string
	AppEnv  string
	AppURL  string
	Debug   bool

	Host string
	Port int

	DatabaseURL string

	BaseCurrency string
	Locale       string

	MarketplaceMode    string
	CompanyName        string
	CompanyDescription string

	PaymentGateway    string
	ManualBankAccount string

	// StoragePath holds uploaded files. Public files live in <StoragePath>/public
	// and are served under /storage/… (the Laravel "public" disk layout).
	StoragePath string

	BcryptCost int
	SeedOnBoot bool
	DBWaitSecs int
}

func Load() Config {
	c := Config{
		AppName:            env("APP_NAME", "B2B Marketplace"),
		AppEnv:             env("APP_ENV", "production"),
		AppURL:             strings.TrimRight(env("APP_URL", "http://localhost:8080"), "/"),
		Debug:              envBool("APP_DEBUG", false),
		Host:               env("HOST", "0.0.0.0"),
		Port:               envInt("PORT", 8000),
		BaseCurrency:       strings.ToUpper(env("APP_BASE_CURRENCY", "USD")),
		Locale:             env("APP_LOCALE", "en"),
		MarketplaceMode:    env("MARKETPLACE_MODE", "b2b"),
		CompanyName:        env("MARKETPLACE_COMPANY_NAME", "Marketplace"),
		CompanyDescription: env("MARKETPLACE_COMPANY_DESCRIPTION", "B2B and B2C marketplace"),
		PaymentGateway:     env("PAYMENT_GATEWAY", "fake"),
		ManualBankAccount:  env("PAYMENT_MANUAL_BANK_ACCOUNT", ""),
		StoragePath:        env("STORAGE_PATH", "storage"),
		BcryptCost:         envInt("BCRYPT_ROUNDS", 12),
		SeedOnBoot:         envBool("SEED_ON_BOOT", true),
		DBWaitSecs:         envInt("DB_WAIT_SECONDS", 60),
	}

	c.DatabaseURL = os.Getenv("DATABASE_URL")
	if c.DatabaseURL == "" {
		c.DatabaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			url.QueryEscape(env("DB_USERNAME", "b2b")),
			url.QueryEscape(env("DB_PASSWORD", "secret")),
			env("DB_HOST", "127.0.0.1"),
			env("DB_PORT", "5432"),
			env("DB_DATABASE", "b2b_marketplace"),
			env("DB_SSLMODE", "disable"),
		)
	}

	return c
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}
