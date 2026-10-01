package ledger

import "testing"

func TestApplyOpCredit(t *testing.T) {
	b, r, err := applyOp(100, 0, 50, OpCredit)
	if err != nil || b != 150 || r != 0 {
		t.Fatalf("credit: got b=%d r=%d err=%v", b, r, err)
	}
}

func TestApplyOpReserveAndSettle(t *testing.T) {
	b, r, err := applyOp(100, 0, 30, OpReserve)
	if err != nil || b != 100 || r != 30 {
		t.Fatalf("reserve: got b=%d r=%d err=%v", b, r, err)
	}
	b, r, err = applyOp(b, r, 30, OpSettle)
	if err != nil || b != 70 || r != 0 {
		t.Fatalf("settle: got b=%d r=%d err=%v", b, r, err)
	}
}

func TestApplyOpReserveInsufficient(t *testing.T) {
	if _, _, err := applyOp(100, 90, 20, OpReserve); err != ErrInsufficientBalance {
		t.Fatalf("expected insufficient-charges, got %v", err)
	}
}

func TestApplyOpRelease(t *testing.T) {
	b, r, err := applyOp(100, 40, 40, OpRelease)
	if err != nil || b != 100 || r != 0 {
		t.Fatalf("release: got b=%d r=%d err=%v", b, r, err)
	}
}

func TestApplyOpSettleTooMuch(t *testing.T) {
	if _, _, err := applyOp(100, 10, 20, OpSettle); err != ErrInsufficientReserved {
		t.Fatalf("expected insufficient-reserved, got %v", err)
	}
}

func TestApplyOpInvalidAmount(t *testing.T) {
	if _, _, err := applyOp(100, 0, 0, OpCredit); err != ErrInvalidAmount {
		t.Fatalf("expected invalid-amount, got %v", err)
	}
}
