package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

func (h *Handlers) OrdersIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := User(r)

	q := &db.Query{}
	services.VisibleOrdersWhere(q, user)
	where := q.SQL()

	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM orders"+where, q.Args()...)
	if err != nil {
		return err
	}
	page, perPage := httpx.PageParams(r, 20, 100)
	perPage = 20
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	orders, err := services.Collect[models.Order](h.app.DB.Query(ctx,
		"SELECT "+db.Columns(models.OrderColumns, "")+" FROM orders"+where+" ORDER BY created_at DESC, id DESC LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}

	ids := services.IDs(orders, func(o models.Order) *int64 { return &o.ID })
	items, err := services.LoadOrderItems(ctx, h.app.DB, ids)
	if err != nil {
		return err
	}
	latest, err := services.LatestPayments(ctx, h.app.DB, ids)
	if err != nil {
		return err
	}
	for i := range orders {
		orders[i].Items = models.Slice(items[orders[i].ID])
		orders[i].Payment = models.Rel(latest[orders[i].ID])
	}

	httpx.JSON(w, http.StatusOK, httpx.NewPaginator(r, orders, total, page, perPage))
	return nil
}

func (h *Handlers) OrdersStore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)

	var lines []services.OrderLine
	if items, ok := v.Array("items", validate.Arr{Required: true, Min: 1}); ok {
		for i, raw := range items {
			obj, _ := raw.(map[string]any)
			sub := v.Sub("items."+itoa(i), obj)
			productID, pidOK := sub.Int("product_id", validate.Num{Required: true})
			qty, qtyOK := sub.Int("quantity", validate.Num{Required: true, Min: validate.F(1)})
			if pidOK {
				exists, err := services.Exists(ctx, h.app.DB, "products", "id = $1", productID)
				if err != nil {
					return err
				}
				if !exists {
					sub.Fail("product_id", "The selected items."+itoa(i)+".product_id is invalid.")
					pidOK = false
				}
			}
			if pidOK && qtyOK {
				lines = append(lines, services.OrderLine{ProductID: productID, Quantity: int(qty)})
			}
		}
	}

	in := services.PlaceInput{}
	if c, ok := v.String("currency_code", validate.Str{Nullable: true, Size: 3}); ok {
		in.CurrencyCode = strings.ToUpper(c)
	}
	in.ContactName, _ = v.OptString("contact_name", validate.Str{Max: 255})
	in.ContactEmail, _ = v.OptString("contact_email", validate.Str{Email: true, Max: 255})
	in.ContactPhone, _ = v.OptString("contact_phone", validate.Str{Max: 50})
	if addr, ok := v.String("shipping_address", validate.Str{Required: true, Max: 1000}); ok {
		in.ShippingAddress = &addr
	}
	in.ShippingCity, _ = v.OptString("shipping_city", validate.Str{Max: 255})
	in.ShippingPostalCode, _ = v.OptString("shipping_postal_code", validate.Str{Max: 32})
	in.ShippingCountry, _ = v.OptString("shipping_country", validate.Str{Size: 2})
	in.Notes, _ = v.OptString("notes", validate.Str{Max: 2000})
	if rate, ok := v.Float("tax_rate", validate.Num{Nullable: true, Min: validate.F(0), Max: validate.F(1)}); ok {
		in.TaxRate = rate
	}
	if err := v.Err(); err != nil {
		return err
	}

	order, err := h.app.Orders.Place(ctx, User(r), lines, in)
	if err != nil {
		var oe *services.OrderError
		if errors.As(err, &oe) {
			return httpx.Unprocessable(oe.Msg)
		}
		return err
	}
	httpx.JSON(w, http.StatusCreated, order)
	return nil
}

func (h *Handlers) OrdersShow(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "order")
	if err != nil {
		return err
	}
	order, err := services.FindOrder(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	if !services.CanAccess(User(r), order) {
		return httpx.Forbidden("This action is unauthorized.")
	}
	items, err := services.LoadOrderItems(ctx, h.app.DB, []int64{id})
	if err != nil {
		return err
	}
	payments, err := services.LoadPayments(ctx, h.app.DB, []int64{id})
	if err != nil {
		return err
	}
	order.Items = models.Slice(items[id])
	order.Payments = models.Slice(payments[id])
	httpx.JSON(w, http.StatusOK, order)
	return nil
}

func itoa(i int) string { return strconv.Itoa(i) }
