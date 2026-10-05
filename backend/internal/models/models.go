// Package models holds the persistence structs. JSON tags reproduce the
// Eloquent serialisation the storefront was built against; db tags map the
// columns for pgx row scanning. Relation fields are typed `any` so a loaded
// relation renders as an object/null/[] and an unloaded one is omitted, just
// like Eloquent's whenLoaded().
package models

import (
	"github.com/asihat/b2b-marketplace/backend/internal/jsonx"
)

// Rel wraps a loaded relation so a nil pointer renders as JSON null instead
// of being omitted.
func Rel[T any](p *T) any { return p }

// Slice wraps a loaded has-many relation so nil renders as [].
func Slice[T any](s []T) any {
	if s == nil {
		return []T{}
	}
	return s
}

// ---- Enumerations -----------------------------------------------------------

const (
	TypeB2C = "b2c"
	TypeB2B = "b2b"

	RoleCustomer = "customer"
	RoleManager  = "manager"
	RoleAdmin    = "admin"

	OrderPending    = "pending"
	OrderPaid       = "paid"
	OrderProcessing = "processing"
	OrderShipped    = "shipped"
	OrderCompleted  = "completed"
	OrderCancelled  = "cancelled"

	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentFailed    = "failed"
	PaymentRefunded  = "refunded"

	AnalogEquivalent = "equivalent"
	AnalogSubstitute = "substitute"
	AnalogUpgrade    = "upgrade"
)

var (
	AccountTypes  = []string{TypeB2C, TypeB2B}
	UserRoles     = []string{RoleCustomer, RoleManager, RoleAdmin}
	OrderStatuses = []string{OrderPending, OrderPaid, OrderProcessing, OrderShipped, OrderCompleted, OrderCancelled}
	AnalogTypes   = []string{AnalogEquivalent, AnalogSubstitute, AnalogUpgrade}
	// RevenueStatuses are the order statuses that count as realised revenue.
	RevenueStatuses = []string{OrderPaid, OrderProcessing, OrderShipped, OrderCompleted}
)

// OrderSettled reports whether no further payment is required.
func OrderSettled(status string) bool {
	return status == OrderPaid || status == OrderCompleted
}

// ---- Users & companies ------------------------------------------------------

type User struct {
	ID              int64       `db:"id" json:"id"`
	CompanyID       *int64      `db:"company_id" json:"company_id"`
	Name            string      `db:"name" json:"name"`
	Email           string      `db:"email" json:"email"`
	EmailVerifiedAt *jsonx.Time `db:"email_verified_at" json:"email_verified_at"`
	Password        string      `db:"password" json:"-"`
	Type            string      `db:"type" json:"type"`
	Role            string      `db:"role" json:"role"`
	Phone           *string     `db:"phone" json:"phone"`
	Locale          string      `db:"locale" json:"locale"`
	Currency        string      `db:"currency" json:"currency"`
	IsActive        bool        `db:"is_active" json:"is_active"`
	CreatedAt       jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       jsonx.Time  `db:"updated_at" json:"updated_at"`

	Company any `db:"-" json:"company,omitempty"`
}

var UserColumns = []string{"id", "company_id", "name", "email", "email_verified_at", "password", "type", "role", "phone", "locale", "currency", "is_active", "created_at", "updated_at"}

func (u *User) IsB2B() bool   { return u.Type == TypeB2B }
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

type Company struct {
	ID              int64      `db:"id" json:"id"`
	Name            string     `db:"name" json:"name"`
	Slug            string     `db:"slug" json:"slug"`
	TaxNumber       *string    `db:"tax_number" json:"tax_number"`
	Email           *string    `db:"email" json:"email"`
	Phone           *string    `db:"phone" json:"phone"`
	Country         *string    `db:"country" json:"country"`
	Address         *string    `db:"address" json:"address"`
	DefaultCurrency string     `db:"default_currency" json:"default_currency"`
	DefaultLocale   string     `db:"default_locale" json:"default_locale"`
	IsVerified      bool       `db:"is_verified" json:"is_verified"`
	CreatedAt       jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt       jsonx.Time `db:"updated_at" json:"updated_at"`

	UsersCount    *int64 `db:"users_count" json:"users_count,omitempty"`
	ProductsCount *int64 `db:"products_count" json:"products_count,omitempty"`
	OrdersCount   *int64 `db:"orders_count" json:"orders_count,omitempty"`
}

var CompanyColumns = []string{"id", "name", "slug", "tax_number", "email", "phone", "country", "address", "default_currency", "default_locale", "is_verified", "created_at", "updated_at"}

// NamedRef is the "{id, name}" projection Eloquent produced for
// `with('company:id,name')`-style eager loads.
type NamedRef struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// UserRef is the `user:id,name,email` projection.
type UserRef struct {
	ID    int64  `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

// CompanyBrief is the storefront's seller summary.
type CompanyBrief struct {
	ID         int64  `db:"id" json:"id"`
	Name       string `db:"name" json:"name"`
	Slug       string `db:"slug" json:"slug"`
	IsVerified bool   `db:"is_verified" json:"is_verified"`
}

// ---- Reference data ---------------------------------------------------------

type Currency struct {
	ID           int64      `db:"id" json:"id"`
	Code         string     `db:"code" json:"code"`
	Name         string     `db:"name" json:"name"`
	Symbol       string     `db:"symbol" json:"symbol"`
	ExchangeRate jsonx.Rate `db:"exchange_rate" json:"exchange_rate"`
	IsBase       bool       `db:"is_base" json:"is_base"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	CreatedAt    jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt    jsonx.Time `db:"updated_at" json:"updated_at"`
}

var CurrencyColumns = []string{"id", "code", "name", "symbol", "exchange_rate", "is_base", "is_active", "created_at", "updated_at"}

type Language struct {
	ID         int64      `db:"id" json:"id"`
	Code       string     `db:"code" json:"code"`
	Name       string     `db:"name" json:"name"`
	NativeName string     `db:"native_name" json:"native_name"`
	IsDefault  bool       `db:"is_default" json:"is_default"`
	IsActive   bool       `db:"is_active" json:"is_active"`
	CreatedAt  jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt  jsonx.Time `db:"updated_at" json:"updated_at"`
}

var LanguageColumns = []string{"id", "code", "name", "native_name", "is_default", "is_active", "created_at", "updated_at"}

type Category struct {
	ID               int64             `db:"id" json:"id"`
	ParentID         *int64            `db:"parent_id" json:"parent_id"`
	Slug             string            `db:"slug" json:"slug"`
	Name             string            `db:"name" json:"name"`
	NameTranslations map[string]string `db:"name_translations" json:"name_translations"`
	Position         int32             `db:"position" json:"position"`
	IsActive         bool              `db:"is_active" json:"is_active"`
	CreatedAt        jsonx.Time        `db:"created_at" json:"created_at"`
	UpdatedAt        jsonx.Time        `db:"updated_at" json:"updated_at"`

	ProductsCount *int64 `db:"products_count" json:"products_count,omitempty"`
}

var CategoryColumns = []string{"id", "parent_id", "slug", "name", "name_translations", "position", "is_active", "created_at", "updated_at"}

// TranslatedName returns the localised name with fallback to the default.
func (c *Category) TranslatedName(locale string) string {
	if n, ok := c.NameTranslations[locale]; ok && n != "" {
		return n
	}
	return c.Name
}

type AppSetting struct {
	ID        int64      `db:"id" json:"id"`
	Key       string     `db:"key" json:"key"`
	Value     *string    `db:"value" json:"value"`
	CreatedAt jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt jsonx.Time `db:"updated_at" json:"updated_at"`
}

// ---- Catalog ----------------------------------------------------------------

type Product struct {
	ID          int64       `db:"id" json:"id"`
	CategoryID  *int64      `db:"category_id" json:"category_id"`
	CompanyID   *int64      `db:"company_id" json:"company_id"`
	SKU         string      `db:"sku" json:"sku"`
	Slug        string      `db:"slug" json:"slug"`
	Name        string      `db:"name" json:"name"`
	Description *string     `db:"description" json:"description"`
	Brand       *string     `db:"brand" json:"brand"`
	Unit        string      `db:"unit" json:"unit"`
	BasePrice   jsonx.Money `db:"base_price" json:"base_price"`
	Stock       int32       `db:"stock" json:"stock"`
	MinOrderQty int32       `db:"min_order_qty" json:"min_order_qty"`
	IsB2BOnly   bool        `db:"is_b2b_only" json:"is_b2b_only"`
	IsActive    bool        `db:"is_active" json:"is_active"`
	CreatedAt   jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   jsonx.Time  `db:"updated_at" json:"updated_at"`

	Category     any    `db:"-" json:"category,omitempty"`
	Company      any    `db:"-" json:"company,omitempty"`
	Images       any    `db:"-" json:"images,omitempty"`
	Prices       any    `db:"-" json:"prices,omitempty"`
	Translations any    `db:"-" json:"translations,omitempty"`
	AnalogsCount *int64 `db:"analogs_count" json:"analogs_count,omitempty"`
}

var ProductColumns = []string{"id", "category_id", "company_id", "sku", "slug", "name", "description", "brand", "unit", "base_price", "stock", "min_order_qty", "is_b2b_only", "is_active", "created_at", "updated_at"}

type ProductImage struct {
	ID        int64      `db:"id" json:"id"`
	ProductID int64      `db:"product_id" json:"product_id"`
	URL       string     `db:"url" json:"url"`
	Alt       *string    `db:"alt" json:"alt"`
	Position  int32      `db:"position" json:"position"`
	IsPrimary bool       `db:"is_primary" json:"is_primary"`
	CreatedAt jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt jsonx.Time `db:"updated_at" json:"updated_at"`
}

var ProductImageColumns = []string{"id", "product_id", "url", "alt", "position", "is_primary", "created_at", "updated_at"}

// PrimaryImageURL returns the primary image (falling back to the first).
func PrimaryImageURL(images []ProductImage) *string {
	for i := range images {
		if images[i].IsPrimary {
			return &images[i].URL
		}
	}
	if len(images) > 0 {
		return &images[0].URL
	}
	return nil
}

type ProductPrice struct {
	ID           int64       `db:"id" json:"id"`
	ProductID    int64       `db:"product_id" json:"product_id"`
	CurrencyCode string      `db:"currency_code" json:"currency_code"`
	MinQty       int32       `db:"min_qty" json:"min_qty"`
	Price        jsonx.Money `db:"price" json:"price"`
	CreatedAt    jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    jsonx.Time  `db:"updated_at" json:"updated_at"`
}

var ProductPriceColumns = []string{"id", "product_id", "currency_code", "min_qty", "price", "created_at", "updated_at"}

type ProductTranslation struct {
	ID          int64      `db:"id" json:"id"`
	ProductID   int64      `db:"product_id" json:"product_id"`
	Locale      string     `db:"locale" json:"locale"`
	Name        string     `db:"name" json:"name"`
	Description *string    `db:"description" json:"description"`
	CreatedAt   jsonx.Time `db:"created_at" json:"created_at"`
	UpdatedAt   jsonx.Time `db:"updated_at" json:"updated_at"`
}

var ProductTranslationColumns = []string{"id", "product_id", "locale", "name", "description", "created_at", "updated_at"}

// TranslatedName picks the translation for locale, falling back to the default name.
func TranslatedName(p *Product, translations []ProductTranslation, locale string) string {
	for i := range translations {
		if translations[i].Locale == locale && translations[i].Name != "" {
			return translations[i].Name
		}
	}
	return p.Name
}

// AnalogLink is the product_analogs pivot.
type AnalogLink struct {
	ProductID int64   `db:"product_id"`
	AnalogID  int64   `db:"analog_id"`
	Type      string  `db:"type"`
	Note      *string `db:"note"`
}

// ---- Orders & payments ------------------------------------------------------

type Order struct {
	ID                 int64       `db:"id" json:"id"`
	Number             string      `db:"number" json:"number"`
	UserID             int64       `db:"user_id" json:"user_id"`
	CompanyID          *int64      `db:"company_id" json:"company_id"`
	Type               string      `db:"type" json:"type"`
	Status             string      `db:"status" json:"status"`
	CurrencyCode       string      `db:"currency_code" json:"currency_code"`
	Subtotal           jsonx.Money `db:"subtotal" json:"subtotal"`
	TaxTotal           jsonx.Money `db:"tax_total" json:"tax_total"`
	GrandTotal         jsonx.Money `db:"grand_total" json:"grand_total"`
	ContactName        *string     `db:"contact_name" json:"contact_name"`
	ContactEmail       *string     `db:"contact_email" json:"contact_email"`
	ContactPhone       *string     `db:"contact_phone" json:"contact_phone"`
	ShippingAddress    *string     `db:"shipping_address" json:"shipping_address"`
	ShippingCity       *string     `db:"shipping_city" json:"shipping_city"`
	ShippingPostalCode *string     `db:"shipping_postal_code" json:"shipping_postal_code"`
	ShippingCountry    *string     `db:"shipping_country" json:"shipping_country"`
	Notes              *string     `db:"notes" json:"notes"`
	CreatedAt          jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          jsonx.Time  `db:"updated_at" json:"updated_at"`

	Items    any `db:"-" json:"items,omitempty"`
	Payment  any `db:"-" json:"payment,omitempty"`
	Payments any `db:"-" json:"payments,omitempty"`
	User     any `db:"-" json:"user,omitempty"`
	Company  any `db:"-" json:"company,omitempty"`
}

var OrderColumns = []string{"id", "number", "user_id", "company_id", "type", "status", "currency_code", "subtotal", "tax_total", "grand_total", "contact_name", "contact_email", "contact_phone", "shipping_address", "shipping_city", "shipping_postal_code", "shipping_country", "notes", "created_at", "updated_at"}

type OrderItem struct {
	ID        int64       `db:"id" json:"id"`
	OrderID   int64       `db:"order_id" json:"order_id"`
	ProductID *int64      `db:"product_id" json:"product_id"`
	Name      string      `db:"name" json:"name"`
	SKU       string      `db:"sku" json:"sku"`
	Quantity  int32       `db:"quantity" json:"quantity"`
	UnitPrice jsonx.Money `db:"unit_price" json:"unit_price"`
	LineTotal jsonx.Money `db:"line_total" json:"line_total"`
	CreatedAt jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt jsonx.Time  `db:"updated_at" json:"updated_at"`
}

var OrderItemColumns = []string{"id", "order_id", "product_id", "name", "sku", "quantity", "unit_price", "line_total", "created_at", "updated_at"}

type Payment struct {
	ID           int64       `db:"id" json:"id"`
	OrderID      int64       `db:"order_id" json:"order_id"`
	Gateway      string      `db:"gateway" json:"gateway"`
	Status       string      `db:"status" json:"status"`
	CurrencyCode string      `db:"currency_code" json:"currency_code"`
	Amount       jsonx.Money `db:"amount" json:"amount"`
	Reference    *string     `db:"reference" json:"reference"`
	Payload      any         `db:"payload" json:"payload"`
	PaidAt       *jsonx.Time `db:"paid_at" json:"paid_at"`
	CreatedAt    jsonx.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    jsonx.Time  `db:"updated_at" json:"updated_at"`
}

var PaymentColumns = []string{"id", "order_id", "gateway", "status", "currency_code", "amount", "reference", "payload", "paid_at", "created_at", "updated_at"}
