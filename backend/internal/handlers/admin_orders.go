package handlers

import (
	"net/http"
	"time"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

// attachOrderRefs loads user:id,name,email and company:id,name.
func (h *Handlers) attachOrderRefs(r *http.Request, orders []models.Order) error {
	ctx := r.Context()
	users, err := services.LoadUserRefs(ctx, h.app.DB, services.IDs(orders, func(o models.Order) *int64 { return &o.UserID }))
	if err != nil {
		return err
	}
	companies, err := services.LoadNamedRefs(ctx, h.app.DB, "companies", services.IDs(orders, func(o models.Order) *int64 { return o.CompanyID }))
	if err != nil {
		return err
	}
	for i := range orders {
		orders[i].User = models.Rel(users[orders[i].UserID])
		var c *models.NamedRef
		if orders[i].CompanyID != nil {
			c = companies[*orders[i].CompanyID]
		}
		orders[i].Company = models.Rel(c)
	}
	return nil
}

func (h *Handlers) AdminOrdersIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := &db.Query{}
	if s := httpx.Query(r, "status"); s != "" {
		q.Wheref("status = %s", s)
	}
	if t := httpx.Query(r, "type"); t != "" {
		q.Wheref("type = %s", t)
	}
	q.ILike(httpx.Query(r, "search"), "number")
	where := q.SQL()

	total, err := services.Count(ctx, h.app.DB, "SELECT COUNT(*) FROM orders"+where, q.Args()...)
	if err != nil {
		return err
	}
	page, perPage := httpx.PageParams(r, 20, 20)
	limit, offset := q.Arg(perPage), q.Arg((page-1)*perPage)
	orders, err := services.Collect[models.Order](h.app.DB.Query(ctx,
		"SELECT "+db.Columns(models.OrderColumns, "")+" FROM orders"+where+" ORDER BY created_at DESC, id DESC LIMIT "+limit+" OFFSET "+offset, q.Args()...))
	if err != nil {
		return err
	}
	if err := h.attachOrderRefs(r, orders); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, httpx.NewPaginator(r, orders, total, page, perPage))
	return nil
}

func (h *Handlers) adminOrderDetail(r *http.Request, id int64) (*models.Order, error) {
	ctx := r.Context()
	order, err := services.FindOrder(ctx, h.app.DB, id)
	if err != nil {
		return nil, err
	}
	items, err := services.LoadOrderItems(ctx, h.app.DB, []int64{id})
	if err != nil {
		return nil, err
	}
	payments, err := services.LoadPayments(ctx, h.app.DB, []int64{id})
	if err != nil {
		return nil, err
	}
	order.Items = models.Slice(items[id])
	order.Payments = models.Slice(payments[id])
	list := []models.Order{*order}
	if err := h.attachOrderRefs(r, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (h *Handlers) AdminOrdersShow(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "order")
	if err != nil {
		return err
	}
	order, err := h.adminOrderDetail(r, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, order)
	return nil
}

func (h *Handlers) AdminOrdersUpdateStatus(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := pathID(r, "order")
	if err != nil {
		return err
	}
	if _, err := services.FindOrder(ctx, h.app.DB, id); err != nil {
		return err
	}
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	status, _ := v.String("status", validate.Str{Required: true, In: models.OrderStatuses})
	if err := v.Err(); err != nil {
		return err
	}
	if _, err := h.app.DB.Exec(ctx, `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`, status, time.Now().UTC(), id); err != nil {
		return err
	}

	order, err := services.FindOrder(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	items, err := services.LoadOrderItems(ctx, h.app.DB, []int64{id})
	if err != nil {
		return err
	}
	order.Items = models.Slice(items[id])
	httpx.JSON(w, http.StatusOK, order)
	return nil
}
