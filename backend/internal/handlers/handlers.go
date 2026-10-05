// Package handlers implements every HTTP endpoint of the marketplace API.
// Response shapes intentionally match the former Laravel controllers and
// resources so the React storefront runs unchanged.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/auth"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

type Handlers struct {
	app *services.App
}

func New(app *services.App) *Handlers { return &Handlers{app: app} }

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxTokenID
	ctxLocale
)

// User returns the authenticated user, or nil for guests.
func User(r *http.Request) *models.User {
	u, _ := r.Context().Value(ctxUser).(*models.User)
	return u
}

func tokenID(r *http.Request) int64 {
	id, _ := r.Context().Value(ctxTokenID).(int64)
	return id
}

// Locale returns the locale negotiated for this request.
func Locale(r *http.Request) string {
	l, _ := r.Context().Value(ctxLocale).(string)
	if l == "" {
		return "en"
	}
	return l
}

func pathID(r *http.Request, name string) (int64, error) {
	return httpx.PathInt64(chi.URLParam(r, name))
}

// currencyParam mirrors `$request->query('currency', $user?->currency ?? 'USD')`.
func currencyParam(r *http.Request) string {
	if c := strings.TrimSpace(httpx.Query(r, "currency")); c != "" {
		return strings.ToUpper(c)
	}
	if u := User(r); u != nil && u.Currency != "" {
		return strings.ToUpper(u.Currency)
	}
	return "USD"
}

func qtyParam(r *http.Request) int {
	qty := httpx.QueryInt(r, "qty", 1)
	if qty < 1 {
		qty = 1
	}
	return qty
}

// ---- Middleware --------------------------------------------------------------

// OptionalAuth resolves the bearer token when present but lets guests through
// (the former OptionalSanctum middleware). Disabled accounts are treated as
// guests.
func (h *Handlers) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if bearer, ok := strings.CutPrefix(header, "Bearer "); ok && strings.TrimSpace(bearer) != "" {
			userID, tokID, err := auth.ResolveToken(r.Context(), h.app.DB, bearer)
			if err == nil {
				user, err := services.FindUser(r.Context(), h.app.DB, userID)
				if err == nil && user.IsActive {
					ctx := context.WithValue(r.Context(), ctxUser, user)
					ctx = context.WithValue(ctx, ctxTokenID, tokID)
					r = r.WithContext(ctx)
				}
			} else if err != auth.ErrInvalidToken {
				httpx.RenderError(w, r, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects guests with 401 (auth:sanctum).
func (h *Handlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if User(r) == nil {
			httpx.JSON(w, http.StatusUnauthorized, httpx.Unauthorized())
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin restricts a route to the platform admin role (EnsureAdmin).
func (h *Handlers) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := User(r)
		if u == nil || !u.IsAdmin() {
			httpx.JSON(w, http.StatusForbidden, httpx.Forbidden("Admin access required."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithLocale negotiates the locale (SetLocale middleware).
func (h *Handlers) WithLocale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := h.app.Locale.Resolve(r, User(r))
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxLocale, locale)))
	})
}

// CORS allows any origin: the API is consumed by a separately hosted SPA
// with bearer tokens (no cookies). Lock down in production as needed.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Accept-Language, X-Requested-With")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recover turns panics into 500 JSON responses.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "path", r.URL.Path, "panic", rec)
				httpx.JSON(w, http.StatusInternalServerError, httpx.NewError(http.StatusInternalServerError, "Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Logger writes one structured line per request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request", "method", r.Method, "path", r.URL.RequestURI(), "status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}
