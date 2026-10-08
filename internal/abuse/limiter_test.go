package abuse

import "testing"

func TestRateLimit(t *testing.T) {
	l := NewLimiter(2, 100)
	if ok, _ := l.Allow("u1"); !ok {
		t.Fatal("first should pass")
	}
	if ok, _ := l.Allow("u1"); !ok {
		t.Fatal("second should pass")
	}
	ok, reason := l.Allow("u1")
	if ok || reason != "rate-limited" {
		t.Fatalf("third should be rate-limited, got ok=%v reason=%q", ok, reason)
	}
	if ok, _ := l.Allow("u2"); !ok {
		t.Fatal("different user should pass")
	}
}

func TestVelocityCap(t *testing.T) {
	l := NewLimiter(1000, 2)
	l.Allow("u1")
	l.Allow("u1")
	ok, reason := l.Allow("u1")
	if ok || reason != "velocity-exceeded" {
		t.Fatalf("expected velocity-exceeded, got ok=%v reason=%q", ok, reason)
	}
}
