// Package server assembles the HTTP router: middleware, public storefront
// routes, authenticated routes and the admin panel API.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/handlers"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
)

func NewRouter(app *services.App) http.Handler {
	h := handlers.New(app)
	H := httpx.H

	r := chi.NewRouter()
	r.Use(handlers.Recover, handlers.Logger, handlers.CORS)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusNotFound, httpx.NotFound())
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusMethodNotAllowed, httpx.NewError(http.StatusMethodNotAllowed, "Method Not Allowed"))
	})

	// Web routes.
	r.Get("/", H(h.Root))
	r.Get("/up", H(h.Health))
	r.Get("/img/{seed}", H(h.Placeholder))
	r.Handle("/storage/*", h.Storage())

	r.Route("/api", func(api chi.Router) {
		api.Use(h.OptionalAuth, h.WithLocale)

		// Public storefront (B2C + browsing).
		api.Post("/auth/register", H(h.Register))
		api.Post("/auth/login", H(h.Login))

		api.Get("/categories", H(h.Categories))
		api.Get("/currencies", H(h.Currencies))
		api.Get("/languages", H(h.Languages))
		api.Get("/settings", H(h.Settings))
		api.Get("/payment-gateways", H(h.Gateways))

		api.Get("/products", H(h.ProductsIndex))
		api.Get("/products/{idOrSlug}", H(h.ProductShow))
		api.Get("/products/{product}/analogs", H(h.ProductAnalogs))

		// Provider webhooks (no auth; verified inside each gateway driver).
		api.Post("/payments/{gateway}/callback", H(h.Callback))

		// Authenticated (B2C + B2B).
		api.Group(func(authed chi.Router) {
			authed.Use(h.RequireAuth)
			authed.Get("/auth/me", H(h.Me))
			authed.Post("/auth/logout", H(h.Logout))

			authed.Get("/orders", H(h.OrdersIndex))
			authed.Post("/orders", H(h.OrdersStore))
			authed.Get("/orders/{order}", H(h.OrdersShow))
			authed.Post("/orders/{order}/pay", H(h.Pay))
		})

		// Admin panel (role: admin).
		api.Route("/admin", func(admin chi.Router) {
			admin.Use(h.RequireAuth, h.RequireAdmin)

			admin.Get("/dashboard", H(h.AdminDashboard))
			admin.Get("/settings", H(h.AdminSettings))
			admin.Put("/settings", H(h.AdminSettingsUpdate))
			admin.Post("/settings/icon", H(h.AdminSettingsUploadIcon))
			admin.Delete("/settings/icon", H(h.AdminSettingsRemoveIcon))

			admin.Get("/users", H(h.AdminUsersIndex))
			admin.Post("/users", H(h.AdminUsersStore))
			admin.Put("/users/{user}", H(h.AdminUsersUpdate))
			admin.Delete("/users/{user}", H(h.AdminUsersDestroy))

			admin.Get("/orders", H(h.AdminOrdersIndex))
			admin.Get("/orders/{order}", H(h.AdminOrdersShow))
			admin.Put("/orders/{order}/status", H(h.AdminOrdersUpdateStatus))

			admin.Get("/products", H(h.AdminProductsIndex))
			admin.Post("/products", H(h.AdminProductsStore))
			admin.Get("/products/{product}", H(h.AdminProductsShow))
			admin.Put("/products/{product}", H(h.AdminProductsUpdate))
			admin.Delete("/products/{product}", H(h.AdminProductsDestroy))

			// Per-product price overrides & B2B volume tiers.
			admin.Get("/products/{product}/prices", H(h.AdminPricesIndex))
			admin.Put("/products/{product}/prices", H(h.AdminPricesSync))
			admin.Post("/products/{product}/prices", H(h.AdminPricesStore))
			admin.Put("/products/{product}/prices/{price}", H(h.AdminPricesUpdate))
			admin.Delete("/products/{product}/prices/{price}", H(h.AdminPricesDestroy))

			// Analog / cross-reference links (kept symmetric on both products).
			admin.Get("/products/{product}/analogs", H(h.AdminAnalogsIndex))
			admin.Post("/products/{product}/analogs", H(h.AdminAnalogsStore))
			admin.Put("/products/{product}/analogs/{analog}", H(h.AdminAnalogsUpdate))
			admin.Delete("/products/{product}/analogs/{analog}", H(h.AdminAnalogsDestroy))

			admin.Get("/categories", H(h.AdminCategoriesIndex))
			admin.Post("/categories", H(h.AdminCategoriesStore))
			admin.Put("/categories/{category}", H(h.AdminCategoriesUpdate))
			admin.Delete("/categories/{category}", H(h.AdminCategoriesDestroy))

			admin.Get("/currencies", H(h.AdminCurrenciesIndex))
			admin.Post("/currencies", H(h.AdminCurrenciesStore))
			admin.Put("/currencies/{currency}", H(h.AdminCurrenciesUpdate))
			admin.Delete("/currencies/{currency}", H(h.AdminCurrenciesDestroy))

			admin.Get("/companies", H(h.AdminCompaniesIndex))
			admin.Put("/companies/{company}", H(h.AdminCompaniesUpdate))
		})
	})

	return r
}
