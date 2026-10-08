package price

import "testing"

func TestNanoPerCharge(t *testing.T) {
	got := nanoPerCharge(2.5, 1)
	if got != 4000000 {
		t.Fatalf("expected 4000000 (1 cent at $2.5/TON), got %d", got)
	}
}

func TestNanoPerChargeCheaperTon(t *testing.T) {
	got := nanoPerCharge(1.0, 1)
	if got != 10000000 {
		t.Fatalf("expected 10000000 (1 cent at $1/TON), got %d", got)
	}
}

func TestNanoPerChargeZeroPrice(t *testing.T) {
	if got := nanoPerCharge(0, 1); got != 0 {
		t.Fatalf("expected 0 for non-positive price, got %d", got)
	}
}
