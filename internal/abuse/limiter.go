package abuse

import (
	"sync"
	"time"
)

type counter struct {
	resetAt time.Time
	n       int
}

type Limiter struct {
	perMin int
	perDay int
	mu     sync.Mutex
	minW   map[string]*counter
	dayW   map[string]*counter
}

func NewLimiter(perMin, perDay int) *Limiter {
	return &Limiter{
		perMin: perMin,
		perDay: perDay,
		minW:   map[string]*counter{},
		dayW:   map[string]*counter{},
	}
}

func (l *Limiter) Allow(userID string) (bool, string) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	mc := ensure(l.minW, userID, now, time.Minute)
	dc := ensure(l.dayW, userID, now, 24*time.Hour)

	if l.perMin > 0 && mc.n >= l.perMin {
		return false, "rate-limited"
	}
	if l.perDay > 0 && dc.n >= l.perDay {
		return false, "velocity-exceeded"
	}

	mc.n++
	dc.n++
	return true, ""
}

func ensure(m map[string]*counter, id string, now time.Time, d time.Duration) *counter {
	c := m[id]
	if c == nil || now.After(c.resetAt) {
		c = &counter{resetAt: now.Add(d)}
		m[id] = c
	}
	return c
}
