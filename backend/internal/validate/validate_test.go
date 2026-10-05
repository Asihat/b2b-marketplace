package validate

import (
	"errors"
	"testing"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
)

func TestRequiredAndMessages(t *testing.T) {
	v := New(map[string]any{"email": "nope", "password": nil})
	v.String("email", Str{Required: true, Email: true})
	v.String("password", Str{Required: true})
	v.String("name", Str{Required: true})

	var httpErr *httpx.Error
	if err := v.Err(); !errors.As(err, &httpErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if httpErr.Status != 422 {
		t.Fatalf("status = %d", httpErr.Status)
	}
	if httpErr.Message != "The email field must be a valid email address. (and 2 more errors)" {
		t.Fatalf("message = %q", httpErr.Message)
	}
	if got := httpErr.Errors["name"][0]; got != "The name field is required." {
		t.Fatalf("name error = %q", got)
	}
}

func TestOptionalAndNullable(t *testing.T) {
	v := New(map[string]any{"brand": nil, "stock": "7", "flag": "1", "size": 3.0})
	if s, present := v.OptString("brand", Str{Max: 10}); !present || s != nil {
		t.Fatalf("brand should be present and null, got %v %v", s, present)
	}
	if _, present := v.OptString("missing", Str{}); present {
		t.Fatal("missing key must not be present")
	}
	if n, ok := v.Int("stock", Num{}); !ok || n != 7 {
		t.Fatalf("stock = %d %v", n, ok)
	}
	if b, ok := v.Bool("flag", false); !ok || !b {
		t.Fatal("flag should be true")
	}
	if n, ok := v.Int("size", Num{Min: F(1)}); !ok || n != 3 {
		t.Fatal("size should parse from float")
	}
	if err := v.Err(); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestInAndSizeAndSubErrors(t *testing.T) {
	v := New(map[string]any{"type": "b2x", "code": "usd1"})
	v.String("type", Str{In: []string{"b2b", "b2c"}})
	v.String("code", Str{Size: 3})
	sub := v.Sub("prices.0", map[string]any{"min_qty": 0})
	sub.Int("min_qty", Num{Required: true, Min: F(1)})

	var httpErr *httpx.Error
	errors.As(v.Err(), &httpErr)
	if httpErr == nil {
		t.Fatal("expected error")
	}
	if httpErr.Errors["type"][0] != "The selected type is invalid." {
		t.Fatalf("type error = %q", httpErr.Errors["type"][0])
	}
	if httpErr.Errors["code"][0] != "The code field must be 3 characters." {
		t.Fatalf("code error = %q", httpErr.Errors["code"][0])
	}
	if _, ok := httpErr.Errors["prices.0.min_qty"]; !ok {
		t.Fatalf("nested error missing: %v", httpErr.Errors)
	}
}
