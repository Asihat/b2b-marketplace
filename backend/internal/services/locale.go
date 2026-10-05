package services

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/cache"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
)

// LocaleService resolves the active locale from (in order) ?lang=, the
// authenticated user's preference, the Accept-Language header, or the default.
type LocaleService struct {
	db        *pgxpool.Pool
	def       string
	supported cache.Value[[]string]
}

func NewLocaleService(pool *pgxpool.Pool, def string) *LocaleService {
	return &LocaleService{db: pool, def: def}
}

func (s *LocaleService) Supported(ctx context.Context) []string {
	codes, err := s.supported.Get(time.Hour, func() ([]string, error) {
		rows, err := s.db.Query(ctx, `SELECT code FROM languages WHERE is_active = true ORDER BY id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		return out, rows.Err()
	})
	if err != nil {
		return nil
	}
	return codes
}

func (s *LocaleService) Flush() { s.supported.Flush() }

func (s *LocaleService) Resolve(r *http.Request, user *models.User) string {
	supported := s.Supported(r.Context())

	var locale string
	switch {
	case strings.TrimSpace(r.URL.Query().Get("lang")) != "":
		locale = strings.TrimSpace(r.URL.Query().Get("lang"))
	case user != nil && user.Locale != "":
		locale = user.Locale
	default:
		locale = preferredLanguage(r.Header.Get("Accept-Language"), supported)
		if locale == "" {
			locale = s.def
		}
	}

	for _, code := range supported {
		if code == locale {
			return locale
		}
	}
	return s.def
}

// preferredLanguage picks the best supported language from an
// Accept-Language header, falling back to the first supported one.
func preferredLanguage(header string, supported []string) string {
	if len(supported) == 0 {
		return ""
	}
	type lang struct {
		tag string
		q   float64
		pos int
	}
	var langs []lang
	for i, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag, params, _ := strings.Cut(part, ";")
		q := 1.0
		for _, p := range strings.Split(params, ";") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "q=") {
				if f, err := strconv.ParseFloat(p[2:], 64); err == nil {
					q = f
				}
			}
		}
		langs = append(langs, lang{tag: strings.ToLower(strings.TrimSpace(tag)), q: q, pos: i})
	}
	sort.SliceStable(langs, func(i, j int) bool { return langs[i].q > langs[j].q })

	isSupported := func(code string) bool {
		for _, s := range supported {
			if strings.EqualFold(s, code) {
				return true
			}
		}
		return false
	}
	for _, l := range langs {
		if l.q <= 0 {
			continue
		}
		candidates := []string{l.tag, strings.ReplaceAll(l.tag, "-", "_")}
		if primary, _, ok := strings.Cut(l.tag, "-"); ok {
			candidates = append(candidates, primary)
		}
		for _, c := range candidates {
			if isSupported(c) {
				for _, s := range supported {
					if strings.EqualFold(s, c) {
						return s
					}
				}
			}
		}
	}
	return supported[0]
}
