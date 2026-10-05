package strx

import "testing"

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"USB-C Cable 1m CBL-USB-C-1M":               "usb-c-cable-1m-cbl-usb-c-1m",
		"Hydraulic Hose 1/2\" SAE 1SN HYDRAULICH-1": "hydraulic-hose-12-sae-1sn-hydraulich-1",
		"  Hello   World  ":                         "hello-world",
		"Crème Brûlée":                              "creme-brulee",
		"Садовый инвентарь":                         "sadovyi-inventar",
		"user@example":                              "user-at-example",
		"NEW-SKU 1":                                 "new-sku-1",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRandom(t *testing.T) {
	a, b := Random(40), Random(40)
	if len(a) != 40 || len(b) != 40 || a == b {
		t.Fatalf("Random produced %q and %q", a, b)
	}
	if u := UUID(); len(u) != 36 || u[14] != '4' {
		t.Fatalf("UUID %q is not a v4 uuid", u)
	}
}
