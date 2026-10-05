package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestPaginatorEnvelope(t *testing.T) {
	r := httptest.NewRequest("GET", "http://api.test/api/products?search=x&page=2", nil)
	p := NewPaginator(r, []int{1, 2}, 5, 2, 2)

	if p.LastPage != 3 || *p.From != 3 || *p.To != 4 || p.Total != 5 {
		t.Fatalf("unexpected paginator %+v", p)
	}
	if *p.NextPageURL != "http://api.test/api/products?page=3&search=x" {
		t.Fatalf("next = %s", *p.NextPageURL)
	}
	if *p.PrevPageURL != "http://api.test/api/products?page=1&search=x" {
		t.Fatalf("prev = %s", *p.PrevPageURL)
	}
	if p.Path != "http://api.test/api/products" {
		t.Fatalf("path = %s", p.Path)
	}

	empty := NewPaginator(r, []int(nil), 0, 1, 20)
	if empty.From != nil || empty.LastPage != 1 || len(empty.Data) != 0 {
		t.Fatalf("empty paginator %+v", empty)
	}

	c := NewCollection(r, []int{1}, 1, 1, 20)
	if c.Meta["total"] != 1 || c.Links["prev"] != (*string)(nil) {
		t.Fatalf("collection meta %+v links %+v", c.Meta, c.Links)
	}
}

func TestInputNormalisesLikeLaravel(t *testing.T) {
	r := httptest.NewRequest("POST", "/x", stringsReader(`{"name":"  Jo ","city":"","password":"","nested":{"a":" "},"list":[" x ",""]}`))
	r.Header.Set("Content-Type", "application/json")
	data, err := Input(r, false)
	if err != nil {
		t.Fatal(err)
	}
	if data["name"] != "Jo" || data["city"] != nil || data["password"] != nil {
		t.Fatalf("unexpected %v", data)
	}
	if data["nested"].(map[string]any)["a"] != nil || data["list"].([]any)[0] != "x" || data["list"].([]any)[1] != nil {
		t.Fatalf("nested normalisation failed: %v", data)
	}
}
