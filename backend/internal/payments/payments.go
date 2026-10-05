// Package payments defines the gateway contract and the two built-in drivers
// (fake sandbox and manual bank transfer). Add Stripe & co. by implementing
// Gateway and registering it with Manager.Extend.
package payments

import (
	"fmt"
	"strings"

	"github.com/asihat/b2b-marketplace/backend/internal/models"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

// Result is the immutable outcome returned by every gateway call.
type Result struct {
	Successful  bool
	Status      string // completed | pending | failed
	Reference   string
	Payload     map[string]any
	RedirectURL string
	Message     string
}

func Completed(reference string, payload map[string]any) Result {
	return Result{Successful: true, Status: models.PaymentCompleted, Reference: reference, Payload: orEmpty(payload)}
}

func Pending(reference, redirectURL string, payload map[string]any) Result {
	return Result{Successful: true, Status: models.PaymentPending, Reference: reference, Payload: orEmpty(payload), RedirectURL: redirectURL}
}

func Failed(message string, payload map[string]any) Result {
	return Result{Successful: false, Status: models.PaymentFailed, Payload: orEmpty(payload), Message: message}
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

// Gateway is the contract every payment provider implements.
type Gateway interface {
	// Name is the machine name, e.g. "stripe".
	Name() string
	// Charge begins a charge for the order.
	Charge(order *models.Order, options map[string]any) Result
	// Callback handles an asynchronous provider webhook.
	Callback(payload map[string]any) Result
	// Refund refunds a previously completed payment by gateway reference.
	Refund(reference string, amount float64) Result
}

// Manager resolves gateway drivers by name.
type Manager struct {
	defaultName string
	drivers     map[string]Gateway
	order       []string
}

func NewManager(defaultName string, drivers ...Gateway) *Manager {
	m := &Manager{defaultName: defaultName, drivers: map[string]Gateway{}}
	for _, d := range drivers {
		m.Extend(d)
	}
	return m
}

// Extend registers a custom gateway at runtime.
func (m *Manager) Extend(g Gateway) {
	if _, exists := m.drivers[g.Name()]; !exists {
		m.order = append(m.order, g.Name())
	}
	m.drivers[g.Name()] = g
}

func (m *Manager) Default() string { return m.defaultName }

// Driver returns the named gateway, or the default when name is empty.
func (m *Manager) Driver(name string) (Gateway, error) {
	if name == "" {
		name = m.defaultName
	}
	g, ok := m.drivers[name]
	if !ok {
		return nil, fmt.Errorf("payment gateway [%s] is not configured", name)
	}
	return g, nil
}

func (m *Manager) Available() []string {
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

// ---- Built-in drivers -------------------------------------------------------

// Fake approves every charge instantly (development / sandbox).
type Fake struct{}

func (Fake) Name() string { return "fake" }

func (Fake) Charge(order *models.Order, _ map[string]any) Result {
	return Completed("fake_"+strx.UUID(), map[string]any{"simulated": true, "order": order.Number})
}

func (Fake) Callback(payload map[string]any) Result {
	return Completed(stringOr(payload["reference"], "fake_callback"), payload)
}

func (Fake) Refund(reference string, amount float64) Result {
	return Completed(reference, map[string]any{"refunded": amount})
}

// Manual is the offline / bank-transfer gateway: charges stay pending until
// an operator confirms the wire through the callback endpoint.
type Manual struct {
	BankAccount string
}

func (Manual) Name() string { return "manual" }

func (m Manual) Charge(order *models.Order, _ map[string]any) Result {
	payload := map[string]any{
		"instructions": "Transfer the order total to the marketplace bank account using the order number as reference.",
		"order":        order.Number,
	}
	if m.BankAccount != "" {
		payload["bank_account"] = m.BankAccount
	}
	return Pending("manual_"+strings.ToUpper(strx.Random(10)), "", payload)
}

func (Manual) Callback(payload map[string]any) Result {
	return Completed(stringOr(payload["reference"], "manual_confirmed"), payload)
}

func (Manual) Refund(reference string, amount float64) Result {
	return Pending(reference, "", map[string]any{"refund_requested": amount})
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}
