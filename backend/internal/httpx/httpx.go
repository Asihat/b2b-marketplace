// Package httpx contains the HTTP plumbing shared by every handler: JSON
// responses, error rendering in Laravel's JSON format, request parsing and
// the two paginator envelopes the storefront understands.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Error is an HTTP error rendered as {"message": ..., "errors": {...}}.
type Error struct {
	Status  int                 `json:"-"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("%d: %s", e.Status, e.Message) }

func NewError(status int, message string) *Error { return &Error{Status: status, Message: message} }

func NotFound() *Error            { return NewError(http.StatusNotFound, "Not found.") }
func Unauthorized() *Error        { return NewError(http.StatusUnauthorized, "Unauthenticated.") }
func Forbidden(msg string) *Error { return NewError(http.StatusForbidden, msg) }
func Unprocessable(msg string) *Error {
	return NewError(http.StatusUnprocessableEntity, msg)
}

// HandlerFunc is an http.HandlerFunc that may return an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// H adapts a HandlerFunc, translating returned errors into JSON responses.
func H(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			RenderError(w, r, err)
		}
	}
}

func RenderError(w http.ResponseWriter, r *http.Request, err error) {
	var httpErr *Error
	switch {
	case errors.As(err, &httpErr):
		JSON(w, httpErr.Status, httpErr)
	case errors.Is(err, pgx.ErrNoRows):
		JSON(w, http.StatusNotFound, NotFound())
	default:
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		JSON(w, http.StatusInternalServerError, NewError(http.StatusInternalServerError, "Server Error"))
	}
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

// Message renders {"message": "..."}.
func Message(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"message": msg})
}

const maxBody = 32 << 20

// Input parses the request body (JSON, form or multipart) plus, when
// withQuery is set, the query string, into one map — like Request::all().
// Strings are trimmed and empty strings become nil, mirroring Laravel's
// TrimStrings and ConvertEmptyStringsToNull middleware.
func Input(r *http.Request, withQuery bool) (map[string]any, error) {
	data := map[string]any{}

	if withQuery {
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				data[k] = v[len(v)-1]
			}
		}
	}

	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch {
	case ct == "application/json" || strings.HasSuffix(ct, "+json"):
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if err != nil {
			return nil, NewError(http.StatusBadRequest, "Could not read request body.")
		}
		if len(strings.TrimSpace(string(body))) > 0 {
			var parsed any
			if err := json.Unmarshal(body, &parsed); err != nil {
				return nil, NewError(http.StatusBadRequest, "Malformed JSON payload.")
			}
			obj, ok := parsed.(map[string]any)
			if !ok {
				return nil, NewError(http.StatusBadRequest, "JSON payload must be an object.")
			}
			for k, v := range obj {
				data[k] = v
			}
		}
	case ct == "multipart/form-data":
		if err := r.ParseMultipartForm(maxBody); err != nil {
			return nil, NewError(http.StatusBadRequest, "Malformed multipart payload.")
		}
		for k, v := range r.MultipartForm.Value {
			if len(v) > 0 {
				data[k] = v[len(v)-1]
			}
		}
	case ct == "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			return nil, NewError(http.StatusBadRequest, "Malformed form payload.")
		}
		for k, v := range r.PostForm {
			if len(v) > 0 {
				data[k] = v[len(v)-1]
			}
		}
	}

	for k, v := range data {
		data[k] = normalize(k, v)
	}
	return data, nil
}

func normalize(key string, v any) any {
	switch t := v.(type) {
	case string:
		if key == "password" || key == "password_confirmation" {
			if t == "" {
				return nil
			}
			return t
		}
		t = strings.TrimSpace(t)
		if t == "" {
			return nil
		}
		return t
	case []any:
		for i := range t {
			t[i] = normalize("", t[i])
		}
		return t
	case map[string]any:
		for k := range t {
			t[k] = normalize(k, t[k])
		}
		return t
	}
	return v
}

// Query returns a query string value ("" when absent).
func Query(r *http.Request, key string) string { return r.URL.Query().Get(key) }

// Filled mirrors $request->filled(): present and not an empty string.
func Filled(r *http.Request, key string) bool {
	return strings.TrimSpace(r.URL.Query().Get(key)) != ""
}

// Truthy mirrors $request->boolean().
func Truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

func QueryInt(r *http.Request, key string, def int) int {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func QueryInt64(r *http.Request, key string) (int64, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func QueryFloat(r *http.Request, key string) (float64, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// PathInt64 parses a positive integer route parameter.
func PathInt64(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, NotFound()
	}
	return n, nil
}

// ---- Pagination -------------------------------------------------------------

// PageParams reads ?page and ?per_page (capped) like Laravel's paginator.
func PageParams(r *http.Request, defaultPer, maxPer int) (page, perPage int) {
	page = QueryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}
	perPage = QueryInt(r, "per_page", defaultPer)
	if perPage < 1 {
		perPage = defaultPer
	}
	if maxPer > 0 && perPage > maxPer {
		perPage = maxPer
	}
	return page, perPage
}

// PageLink is one entry of Laravel's "links" array.
type PageLink struct {
	URL    *string `json:"url"`
	Label  string  `json:"label"`
	Active bool    `json:"active"`
}

// Paginator is the plain LengthAwarePaginator JSON (admin endpoints).
type Paginator[T any] struct {
	CurrentPage  int        `json:"current_page"`
	Data         []T        `json:"data"`
	FirstPageURL string     `json:"first_page_url"`
	From         *int       `json:"from"`
	LastPage     int        `json:"last_page"`
	LastPageURL  string     `json:"last_page_url"`
	Links        []PageLink `json:"links"`
	NextPageURL  *string    `json:"next_page_url"`
	Path         string     `json:"path"`
	PerPage      int        `json:"per_page"`
	PrevPageURL  *string    `json:"prev_page_url"`
	To           *int       `json:"to"`
	Total        int        `json:"total"`
}

// Collection is the ResourceCollection envelope ({data, links, meta}).
type Collection[T any] struct {
	Data  []T            `json:"data"`
	Links map[string]any `json:"links"`
	Meta  map[string]any `json:"meta"`
}

func pageURL(r *http.Request, page int) string {
	u := *r.URL
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return baseURL(r) + u.Path + "?" + u.RawQuery
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

func NewPaginator[T any](r *http.Request, data []T, total, page, perPage int) Paginator[T] {
	if data == nil {
		data = []T{}
	}
	lastPage := int(math.Max(1, math.Ceil(float64(total)/float64(perPage))))
	p := Paginator[T]{
		CurrentPage:  page,
		Data:         data,
		FirstPageURL: pageURL(r, 1),
		LastPage:     lastPage,
		LastPageURL:  pageURL(r, lastPage),
		Path:         baseURL(r) + r.URL.Path,
		PerPage:      perPage,
		Total:        total,
	}
	if len(data) > 0 {
		from := (page-1)*perPage + 1
		to := from + len(data) - 1
		p.From, p.To = &from, &to
	}
	if page > 1 {
		u := pageURL(r, page-1)
		p.PrevPageURL = &u
	}
	if page < lastPage {
		u := pageURL(r, page+1)
		p.NextPageURL = &u
	}
	p.Links = links(r, page, lastPage)
	return p
}

func links(r *http.Request, page, lastPage int) []PageLink {
	out := []PageLink{{Label: "&laquo; Previous"}}
	if page > 1 {
		u := pageURL(r, page-1)
		out[0].URL = &u
	}
	for i := 1; i <= lastPage; i++ {
		u := pageURL(r, i)
		out = append(out, PageLink{URL: &u, Label: strconv.Itoa(i), Active: i == page})
	}
	next := PageLink{Label: "Next &raquo;"}
	if page < lastPage {
		u := pageURL(r, page+1)
		next.URL = &u
	}
	return append(out, next)
}

func NewCollection[T any](r *http.Request, data []T, total, page, perPage int) Collection[T] {
	p := NewPaginator(r, data, total, page, perPage)
	return Collection[T]{
		Data: p.Data,
		Links: map[string]any{
			"first": p.FirstPageURL,
			"last":  p.LastPageURL,
			"prev":  p.PrevPageURL,
			"next":  p.NextPageURL,
		},
		Meta: map[string]any{
			"current_page": p.CurrentPage,
			"from":         p.From,
			"last_page":    p.LastPage,
			"links":        p.Links,
			"path":         p.Path,
			"per_page":     p.PerPage,
			"to":           p.To,
			"total":        p.Total,
		},
	}
}

// AbsoluteURL joins a path onto the configured application URL.
func AbsoluteURL(appURL, path string) string {
	u, err := url.Parse(appURL)
	if err != nil {
		return appURL + path
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return u.String()
}
