package catalog

import "testing"

func TestAll(t *testing.T) {
	if len(All()) != 3 {
		t.Fatalf("expected 3 products, got %d", len(All()))
	}
}

func TestLookup(t *testing.T) {
	p, ok := Lookup("charges_500")
	if !ok || p.Charges != 500 {
		t.Fatalf("expected charges_500 -> 500, got %+v ok=%v", p, ok)
	}
	if _, ok := Lookup("nope"); ok {
		t.Fatalf("expected miss for unknown id")
	}
}
