package print

import "testing"

func TestBufferNano(t *testing.T) {
	got := BufferNano(Config{BufferBaseNano: 50000000, MarginBPS: 15000})
	if got != 75000000 {
		t.Fatalf("expected 75000000 (0.05 * 1.5), got %d", got)
	}
}

func TestBufferNanoNoMargin(t *testing.T) {
	got := BufferNano(Config{BufferBaseNano: 50000000, MarginBPS: 0})
	if got != 50000000 {
		t.Fatalf("expected base when margin unset, got %d", got)
	}
}

func TestNanoToTON(t *testing.T) {
	cases := map[int64]string{
		75000000:   "0.075",
		50000000:   "0.05",
		1000000000: "1",
		1500000000: "1.5",
		0:          "0",
	}
	for nano, want := range cases {
		if got := NanoToTON(nano); got != want {
			t.Fatalf("NanoToTON(%d)=%q, want %q", nano, got, want)
		}
	}
}
