package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/cache"
	"github.com/asihat/b2b-marketplace/backend/internal/config"
)

// PublicSettings is what GET /api/settings returns to the storefront.
type PublicSettings struct {
	Mode               string  `json:"mode"`
	ShowCompanyNames   bool    `json:"show_company_names"`
	IconURL            *string `json:"icon_url"`
	CompanyName        string  `json:"company_name"`
	CompanyDescription string  `json:"company_description"`
}

// SettingsService reads and writes the app_settings key/value table, with
// config/env values as defaults.
type SettingsService struct {
	db   *pgxpool.Pool
	cfg  config.Config
	disk *PublicDisk
	all  cache.Value[map[string]*string]
}

func NewSettingsService(pool *pgxpool.Pool, cfg config.Config, disk *PublicDisk) *SettingsService {
	return &SettingsService{db: pool, cfg: cfg, disk: disk}
}

func (s *SettingsService) values(ctx context.Context) map[string]*string {
	vals, err := s.all.Get(0, func() (map[string]*string, error) {
		rows, err := s.db.Query(ctx, `SELECT key, value FROM app_settings`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]*string{}
		for rows.Next() {
			var k string
			var v *string
			if err := rows.Scan(&k, &v); err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, rows.Err()
	})
	if err != nil {
		return map[string]*string{}
	}
	return vals
}

func (s *SettingsService) get(ctx context.Context, key string) (string, bool) {
	v, ok := s.values(ctx)[key]
	if !ok || v == nil {
		return "", false
	}
	return *v, true
}

func (s *SettingsService) Public(ctx context.Context) PublicSettings {
	mode := s.Mode(ctx)
	return PublicSettings{
		Mode:               mode,
		ShowCompanyNames:   mode == "b2b",
		IconURL:            s.IconURL(ctx),
		CompanyName:        s.CompanyName(ctx),
		CompanyDescription: s.CompanyDescription(ctx),
	}
}

func (s *SettingsService) Mode(ctx context.Context) string {
	mode, ok := s.get(ctx, "marketplace_mode")
	if !ok {
		mode = s.cfg.MarketplaceMode
	}
	if mode == "b2b" || mode == "b2c" {
		return mode
	}
	return "b2c"
}

func (s *SettingsService) ShowCompanyNames(ctx context.Context) bool { return s.Mode(ctx) == "b2b" }

func (s *SettingsService) CompanyName(ctx context.Context) string {
	if v, ok := s.get(ctx, "company_name"); ok {
		return v
	}
	return s.cfg.CompanyName
}

func (s *SettingsService) CompanyDescription(ctx context.Context) string {
	if v, ok := s.get(ctx, "company_description"); ok {
		return v
	}
	return s.cfg.CompanyDescription
}

func (s *SettingsService) IconURL(ctx context.Context) *string {
	path, ok := s.get(ctx, "main_icon_path")
	if !ok || path == "" {
		return nil
	}
	u := s.disk.URL(path)
	return &u
}

func (s *SettingsService) set(ctx context.Context, key, value string) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx, `
		INSERT INTO app_settings (key, value, created_at, updated_at) VALUES ($1, $2, $3, $3)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		key, value, now)
	return err
}

// Update persists any of mode / company_name / company_description.
func (s *SettingsService) Update(ctx context.Context, values map[string]*string) (PublicSettings, error) {
	keys := map[string]string{
		"mode":                "marketplace_mode",
		"company_name":        "company_name",
		"company_description": "company_description",
	}
	for input, key := range keys {
		v, present := values[input]
		if !present {
			continue
		}
		str := ""
		if v != nil {
			str = *v
		}
		if err := s.set(ctx, key, str); err != nil {
			return PublicSettings{}, err
		}
	}
	s.Flush()
	return s.Public(ctx), nil
}

// UpdateIcon stores the new icon path and removes the previous file.
func (s *SettingsService) UpdateIcon(ctx context.Context, path string) (PublicSettings, error) {
	old, _ := s.get(ctx, "main_icon_path")
	if err := s.set(ctx, "main_icon_path", path); err != nil {
		return PublicSettings{}, err
	}
	s.Flush()
	if old != "" && old != path && isManagedIcon(old) {
		_ = s.disk.Delete(old)
	}
	return s.Public(ctx), nil
}

func (s *SettingsService) RemoveIcon(ctx context.Context) (PublicSettings, error) {
	old, _ := s.get(ctx, "main_icon_path")
	if _, err := s.db.Exec(ctx, `DELETE FROM app_settings WHERE key = 'main_icon_path'`); err != nil {
		return PublicSettings{}, err
	}
	s.Flush()
	if old != "" && isManagedIcon(old) {
		_ = s.disk.Delete(old)
	}
	return s.Public(ctx), nil
}

func isManagedIcon(path string) bool {
	return len(path) > len("settings/icons/") && path[:len("settings/icons/")] == "settings/icons/"
}

func (s *SettingsService) Flush() { s.all.Flush() }
