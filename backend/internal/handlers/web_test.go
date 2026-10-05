package handlers

import (
	"strings"
	"testing"
)

func TestPlaceholderSVGIsDeterministicAndUsesInitials(t *testing.T) {
	a := PlaceholderSVG("usb-c-cable-1m-cbl-usb-c-1m-1")
	b := PlaceholderSVG("usb-c-cable-1m-cbl-usb-c-1m-1")
	if a != b {
		t.Fatal("same seed produced different artwork")
	}
	if !strings.Contains(a, ">UC</text>") {
		t.Fatalf("expected initials UC in %s", a)
	}
	if !strings.Contains(PlaceholderSVG("???"), ">?</text>") {
		t.Fatal("seed without letters should fall back to ?")
	}
	if PlaceholderSVG("alpha") == PlaceholderSVG("beta") {
		t.Fatal("different seeds should differ")
	}
}
