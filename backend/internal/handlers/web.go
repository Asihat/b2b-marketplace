package handlers

import (
	"fmt"
	"hash/crc32"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
)

func (h *Handlers) Root(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"name":   h.app.Cfg.AppName,
		"status": "ok",
		"api":    httpx.AbsoluteURL(h.app.Cfg.AppURL, "/api"),
	})
	return nil
}

// Health is the /up endpoint used by container health checks.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) error {
	if err := h.app.DB.Ping(r.Context()); err != nil {
		return httpx.NewError(http.StatusServiceUnavailable, "Database unavailable.")
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

var seedWords = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// PlaceholderSVG renders the deterministic SVG placeholder for a seed.
func PlaceholderSVG(seed string) string {
	hash := crc32.ChecksumIEEE([]byte(seed))
	h1 := hash % 360
	h2 := (h1 + 45) % 360

	words := []string{}
	for _, w := range seedWords.Split(seed, -1) {
		if w != "" {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		words = []string{"?"}
	}
	initials := words[0][:1]
	if len(words) > 1 {
		initials += words[1][:1]
	}
	initials = strings.ToUpper(initials)

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="800" viewBox="0 0 800 800">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="hsl(%d, 62%%, 56%%)"/>
      <stop offset="1" stop-color="hsl(%d, 68%%, 44%%)"/>
    </linearGradient>
  </defs>
  <rect width="800" height="800" fill="url(#g)"/>
  <circle cx="640" cy="160" r="220" fill="#ffffff" opacity="0.08"/>
  <circle cx="160" cy="660" r="160" fill="#ffffff" opacity="0.06"/>
  <text x="50%%" y="50%%" dy="0.35em" text-anchor="middle"
    font-family="Inter, Segoe UI, Arial, sans-serif" font-size="300" font-weight="700"
    fill="#ffffff" opacity="0.92">%s</text>
</svg>
`, h1, h2, initials)
}

// Placeholder serves locally generated product images (GET /img/{seed}) so
// the storefront never depends on an external image host.
func (h *Handlers) Placeholder(w http.ResponseWriter, r *http.Request) error {
	seed := chi.URLParam(r, "seed")
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write([]byte(PlaceholderSVG(seed)))
	return nil
}

// Storage serves uploaded public files (the former storage:link symlink).
func (h *Handlers) Storage() http.Handler {
	fs := http.StripPrefix("/storage/", http.FileServer(http.Dir(h.app.Disk.Root)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			httpx.JSON(w, http.StatusNotFound, httpx.NotFound())
			return
		}
		fs.ServeHTTP(w, r)
	})
}
