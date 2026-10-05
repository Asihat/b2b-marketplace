package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/payments"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

// OrderError is a business-rule violation while placing an order (rendered
// as 422 by the handler), as opposed to an infrastructure failure.
type OrderError struct{ Msg string }

func (e *OrderError) Error() string { return e.Msg }

type OrderLine struct {
	ProductID int64
	Quantity  int
}

type PlaceInput struct {
	CurrencyCode       string
	TaxRate            float64
	ContactName        *string
	ContactEmail       *string
	ContactPhone       *string
	ShippingAddress    *string
	ShippingCity       *string
	ShippingPostalCode *string
	ShippingCountry    *string
	Notes              *string
}

type OrderService struct {
	db       *pgxpool.Pool
	currency *CurrencyService
	payments *payments.Manager
}

func NewOrderService(pool *pgxpool.Pool, currency *CurrencyService, pm *payments.Manager) *OrderService {
	return &OrderService{db: pool, currency: currency, payments: pm}
}

// Place creates an order for a user from product/quantity lines. Stock is
// reserved row by row under FOR UPDATE locks inside one transaction.
func (s *OrderService) Place(ctx context.Context, user *models.User, lines []OrderLine, in PlaceInput) (*models.Order, error) {
	if len(lines) == 0 {
		return nil, &OrderError{"Cannot place an empty order."}
	}

	currency := strings.ToUpper(in.CurrencyCode)
	if currency == "" {
		currency = user.Currency
	}
	if currency == "" {
		currency = s.currency.Base(ctx).Code
	}

	orderType := models.TypeB2C
	if user.IsB2B() {
		orderType = models.TypeB2B
	}
	contactName := in.ContactName
	if contactName == nil {
		contactName = Ptr(user.Name)
	}
	contactEmail := in.ContactEmail
	if contactEmail == nil {
		contactEmail = Ptr(user.Email)
	}
	contactPhone := in.ContactPhone
	if contactPhone == nil {
		contactPhone = user.Phone
	}

	var order *models.Order
	err := db.Tx(ctx, s.db, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		var orderID int64
		err := tx.QueryRow(ctx, `
			INSERT INTO orders (number, user_id, company_id, type, status, currency_code,
				contact_name, contact_email, contact_phone,
				shipping_address, shipping_city, shipping_postal_code, shipping_country, notes,
				created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
			RETURNING id`,
			"ORD-"+strings.ToUpper(strx.Random(8)), user.ID, user.CompanyID, orderType, models.OrderPending, currency,
			contactName, contactEmail, contactPhone,
			in.ShippingAddress, in.ShippingCity, in.ShippingPostalCode, in.ShippingCountry, in.Notes,
			now,
		).Scan(&orderID)
		if err != nil {
			return err
		}

		subtotal := 0.0
		for _, line := range lines {
			product, err := One[models.Product](tx.Query(ctx,
				"SELECT "+db.Columns(models.ProductColumns, "")+" FROM products WHERE id = $1 AND is_active = true FOR UPDATE", line.ProductID))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return &OrderError{fmt.Sprintf("Product %d is not available.", line.ProductID)}
				}
				return err
			}
			qty := line.Quantity
			if qty < 1 {
				qty = 1
			}

			if product.IsB2BOnly && !user.IsB2B() {
				return &OrderError{fmt.Sprintf("Product %s is available to business accounts only.", product.SKU)}
			}
			if qty < int(product.MinOrderQty) {
				return &OrderError{fmt.Sprintf("Minimum order quantity for %s is %d.", product.SKU, product.MinOrderQty)}
			}
			if int(product.Stock) < qty {
				return &OrderError{fmt.Sprintf("Insufficient stock for %s.", product.SKU)}
			}

			unitPrice, err := s.currency.PriceForProduct(ctx, tx, product, currency, qty)
			if err != nil {
				return err
			}
			lineTotal := Round(unitPrice*float64(qty), 4)
			subtotal += lineTotal

			if _, err := tx.Exec(ctx, `
				INSERT INTO order_items (order_id, product_id, name, sku, quantity, unit_price, line_total, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
				orderID, product.ID, product.Name, product.SKU, qty, unitPrice, lineTotal, now); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE products SET stock = stock - $1, updated_at = $2 WHERE id = $3`, qty, now, product.ID); err != nil {
				return err
			}
		}

		tax := Round(subtotal*in.TaxRate, 4)
		if _, err := tx.Exec(ctx, `UPDATE orders SET subtotal = $1, tax_total = $2, grand_total = $3, updated_at = $4 WHERE id = $5`,
			subtotal, tax, Round(subtotal+tax, 4), now, orderID); err != nil {
			return err
		}

		order, err = FindOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		items, err := LoadOrderItems(ctx, tx, []int64{orderID})
		if err != nil {
			return err
		}
		order.Items = models.Slice(items[orderID])
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Pay charges an order through a gateway and records the payment.
func (s *OrderService) Pay(ctx context.Context, order *models.Order, gateway string) (*models.Payment, error) {
	driver, err := s.payments.Driver(gateway)
	if err != nil {
		return nil, err
	}
	result := driver.Charge(order, nil)
	completed := result.Status == models.PaymentCompleted

	now := time.Now().UTC()
	var paidAt *time.Time
	if completed {
		paidAt = &now
	}
	var reference *string
	if result.Reference != "" {
		reference = &result.Reference
	}

	var payment *models.Payment
	err = db.Tx(ctx, s.db, func(tx pgx.Tx) error {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO payments (order_id, gateway, status, currency_code, amount, reference, payload, paid_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9) RETURNING id`,
			order.ID, driver.Name(), result.Status, order.CurrencyCode, order.GrandTotal.Float(), reference, result.Payload, paidAt, now,
		).Scan(&id)
		if err != nil {
			return err
		}
		if completed {
			if _, err := tx.Exec(ctx, `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`, models.OrderPaid, now, order.ID); err != nil {
				return err
			}
		}
		payment, err = FindPayment(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// ConfirmCallback applies a gateway callback result to the referenced payment.
func (s *OrderService) ConfirmCallback(ctx context.Context, result payments.Result) error {
	if result.Reference == "" {
		return nil
	}
	return db.Tx(ctx, s.db, func(tx pgx.Tx) error {
		payment, err := One[models.Payment](tx.Query(ctx,
			"SELECT "+db.Columns(models.PaymentColumns, "")+" FROM payments WHERE reference = $1 ORDER BY id LIMIT 1", result.Reference))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		now := time.Now().UTC()
		completed := result.Status == models.PaymentCompleted
		paidAt := (*time.Time)(nil)
		if payment.PaidAt != nil {
			t := payment.PaidAt.Time
			paidAt = &t
		}
		if completed {
			paidAt = &now
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = $1, paid_at = $2, payload = $3, updated_at = $4 WHERE id = $5`,
			result.Status, paidAt, result.Payload, now, payment.ID); err != nil {
			return err
		}
		if completed {
			if _, err := tx.Exec(ctx, `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`, models.OrderPaid, now, payment.OrderID); err != nil {
				return err
			}
		}
		return nil
	})
}

// CanAccess is the OrderPolicy: admins always; otherwise the buyer or a B2B
// colleague from the same company.
func CanAccess(user *models.User, order *models.Order) bool {
	if user.IsAdmin() || order.UserID == user.ID {
		return true
	}
	return user.IsB2B() && order.CompanyID != nil && user.CompanyID != nil && *order.CompanyID == *user.CompanyID
}

// VisibleOrdersWhere is Order::visibleTo(): B2B buyers see their company's
// orders, everyone else only their own.
func VisibleOrdersWhere(q *db.Query, user *models.User) {
	if user.IsB2B() && user.CompanyID != nil {
		q.Wheref("company_id = %s", *user.CompanyID)
		return
	}
	q.Wheref("user_id = %s", user.ID)
}
