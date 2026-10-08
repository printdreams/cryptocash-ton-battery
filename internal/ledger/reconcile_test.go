package ledger

import "testing"

func TestDeltaFor(t *testing.T) {
	var b, r int64
	for _, step := range []struct {
		op  OpType
		amt int64
	}{
		{OpCredit, 100},
		{OpReserve, 30},
		{OpSettle, 30},
		{OpCredit, 50},
		{OpReserve, 20},
		{OpRelease, 20},
	} {
		db, dr := deltaFor(step.op, step.amt)
		b += db
		r += dr
	}
	if b != 120 || r != 0 {
		t.Fatalf("expected balance 120 reserved 0, got balance=%d reserved=%d", b, r)
	}
}
