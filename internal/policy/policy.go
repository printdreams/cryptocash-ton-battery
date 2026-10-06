package policy

import "strings"

type Config struct {
	MaxOutMessages      int
	BlockedDestinations []string
	FeeCapNano          int64
	NanoPerCharge       int64
	MarginCharges       int64
	MinCharge           int64
}

type Facts struct {
	StaticReason     string
	Emulated         bool
	EmulationSuccess bool
	TotalFeesNano    int64
	OutMessages      int
	Destinations     []string
	AvailableCharges int64
	RateLimited      bool
}

type Decision struct {
	SupportedByBattery bool
	AllowedByBattery   bool
	RejectReason       string
	Charge             int64
}

func Evaluate(f Facts, cfg Config) Decision {
	d := Decision{SupportedByBattery: true, AllowedByBattery: true}

	if f.StaticReason != "" {
		d.SupportedByBattery = false
		d.AllowedByBattery = false
		d.RejectReason = f.StaticReason
		return d
	}

	if cfg.MaxOutMessages > 0 && f.OutMessages > cfg.MaxOutMessages {
		d.SupportedByBattery = false
		d.AllowedByBattery = false
		d.RejectReason = "too-many-messages"
		return d
	}

	if isBlocked(f.Destinations, cfg.BlockedDestinations) {
		d.SupportedByBattery = false
		d.AllowedByBattery = false
		d.RejectReason = "destination-blocked"
		return d
	}

	if !f.Emulated || !f.EmulationSuccess {
		d.AllowedByBattery = false
		d.RejectReason = "emulation-failed"
		return d
	}

	if cfg.FeeCapNano > 0 && f.TotalFeesNano > cfg.FeeCapNano {
		d.AllowedByBattery = false
		d.RejectReason = "fee-too-high"
		return d
	}

	charge := ComputeCharge(f.TotalFeesNano, cfg)
	d.Charge = charge

	if f.RateLimited {
		d.AllowedByBattery = false
		d.RejectReason = "rate-limited"
		return d
	}

	if f.AvailableCharges < charge {
		d.AllowedByBattery = false
		d.RejectReason = "insufficient-charges"
		return d
	}

	return d
}

func ComputeCharge(feesNano int64, cfg Config) int64 {
	per := cfg.NanoPerCharge
	if per <= 0 {
		per = 1
	}
	c := (feesNano + per - 1) / per
	c += cfg.MarginCharges
	if c < cfg.MinCharge {
		c = cfg.MinCharge
	}
	return c
}

func isBlocked(destinations, blocked []string) bool {
	for _, d := range destinations {
		for _, b := range blocked {
			if strings.EqualFold(strings.TrimSpace(d), strings.TrimSpace(b)) {
				return true
			}
		}
	}
	return false
}
