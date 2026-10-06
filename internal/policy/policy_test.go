package policy

import "testing"

func baseCfg() Config {
	return Config{
		MaxOutMessages: 4,
		FeeCapNano:     1000000000,
		NanoPerCharge:  10000000,
		MarginCharges:  1,
		MinCharge:      1,
	}
}

func okFacts() Facts {
	return Facts{
		Emulated:         true,
		EmulationSuccess: true,
		TotalFeesNano:    25000000,
		OutMessages:      1,
		Destinations:     []string{"0:dest"},
		AvailableCharges: 100,
	}
}

func TestHappyPath(t *testing.T) {
	d := Evaluate(okFacts(), baseCfg())
	if !d.SupportedByBattery || !d.AllowedByBattery || d.RejectReason != "" {
		t.Fatalf("expected allowed, got %+v", d)
	}
	if d.Charge != 4 {
		t.Fatalf("expected charge 4 (ceil(25000000/10000000)=3 + margin 1), got %d", d.Charge)
	}
}

func TestStaticReason(t *testing.T) {
	f := okFacts()
	f.StaticReason = "ttl-too-short"
	d := Evaluate(f, baseCfg())
	if d.SupportedByBattery || d.AllowedByBattery || d.RejectReason != "ttl-too-short" {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestTooManyMessages(t *testing.T) {
	f := okFacts()
	f.OutMessages = 10
	d := Evaluate(f, baseCfg())
	if d.SupportedByBattery || d.RejectReason != "too-many-messages" {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestDestinationBlocked(t *testing.T) {
	f := okFacts()
	f.Destinations = []string{"0:evil"}
	cfg := baseCfg()
	cfg.BlockedDestinations = []string{"0:EVIL"}
	d := Evaluate(f, cfg)
	if d.SupportedByBattery || d.RejectReason != "destination-blocked" {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestEmulationFailed(t *testing.T) {
	f := okFacts()
	f.EmulationSuccess = false
	d := Evaluate(f, baseCfg())
	if d.AllowedByBattery || d.RejectReason != "emulation-failed" {
		t.Fatalf("unexpected: %+v", d)
	}
	if !d.SupportedByBattery {
		t.Fatalf("expected supported true for dynamic failure")
	}
}

func TestFeeTooHigh(t *testing.T) {
	f := okFacts()
	f.TotalFeesNano = 2000000000
	d := Evaluate(f, baseCfg())
	if d.AllowedByBattery || d.RejectReason != "fee-too-high" {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestInsufficientCharges(t *testing.T) {
	f := okFacts()
	f.AvailableCharges = 1
	d := Evaluate(f, baseCfg())
	if d.AllowedByBattery || d.RejectReason != "insufficient-charges" {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestRateLimited(t *testing.T) {
	f := okFacts()
	f.RateLimited = true
	d := Evaluate(f, baseCfg())
	if d.AllowedByBattery || d.RejectReason != "rate-limited" {
		t.Fatalf("unexpected: %+v", d)
	}
}
