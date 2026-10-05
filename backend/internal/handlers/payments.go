package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

// Pay initiates payment for an order.
func (h *Handlers) Pay(w http.ResponseWriter, r *http.Request) error {
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
	if models.OrderSettled(order.Status) {
		return httpx.Unprocessable("Order is already paid.")
	}

	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	gateway, _ := v.String("gateway", validate.Str{Nullable: true, In: h.app.Payments.Available()})
	if err := v.Err(); err != nil {
		return err
	}

	payment, err := h.app.Orders.Pay(ctx, order, gateway)
	if err != nil {
		return err
	}
	fresh, err := services.FindOrder(ctx, h.app.DB, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"order": fresh, "payment": payment})
	return nil
}

// Callback is the public provider webhook endpoint. Signature verification
// belongs in each gateway driver.
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) error {
	driver, err := h.app.Payments.Driver(chi.URLParam(r, "gateway"))
	if err != nil {
		return httpx.NewError(http.StatusNotFound, err.Error())
	}
	data, err := httpx.Input(r, true)
	if err != nil {
		return err
	}
	result := driver.Callback(data)
	if err := h.app.Orders.ConfirmCallback(r.Context(), result); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": result.Status})
	return nil
}
