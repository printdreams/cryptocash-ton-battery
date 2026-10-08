package price

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Oracle struct {
	baseURL        string
	coinID         string
	currency       string
	centsPerCharge int64
	fallbackNano   int64
	ttl            time.Duration
	hc             *http.Client

	mu          sync.Mutex
	cachedNano  int64
	cachedPrice float64
	fetchedAt   time.Time
}

func NewOracle(baseURL, coinID, currency string, centsPerCharge, fallbackNano int64, ttl time.Duration) *Oracle {
	return &Oracle{
		baseURL:        strings.TrimRight(baseURL, "/"),
		coinID:         coinID,
		currency:       strings.ToLower(currency),
		centsPerCharge: centsPerCharge,
		fallbackNano:   fallbackNano,
		ttl:            ttl,
		hc:             &http.Client{Timeout: 10 * time.Second},
	}
}

type Quote struct {
	NanoPerCharge  int64   `json:"nanoPerCharge"`
	TonPrice       float64 `json:"tonPrice"`
	Currency       string  `json:"currency"`
	CentsPerCharge int64   `json:"centsPerCharge"`
	Fallback       bool    `json:"fallback"`
}

func (o *Oracle) Quote(ctx context.Context) Quote {
	o.mu.Lock()
	fresh := o.cachedNano > 0 && time.Since(o.fetchedAt) < o.ttl
	cachedNano := o.cachedNano
	cachedPrice := o.cachedPrice
	o.mu.Unlock()

	if fresh {
		return Quote{NanoPerCharge: cachedNano, TonPrice: cachedPrice, Currency: o.currency, CentsPerCharge: o.centsPerCharge}
	}

	p, err := o.fetchPrice(ctx)
	if err != nil || p <= 0 {
		if cachedNano > 0 {
			return Quote{NanoPerCharge: cachedNano, TonPrice: cachedPrice, Currency: o.currency, CentsPerCharge: o.centsPerCharge}
		}
		return Quote{NanoPerCharge: o.fallbackNano, Currency: o.currency, CentsPerCharge: o.centsPerCharge, Fallback: true}
	}

	nano := nanoPerCharge(p, o.centsPerCharge)
	if nano <= 0 {
		nano = o.fallbackNano
	}

	o.mu.Lock()
	o.cachedNano = nano
	o.cachedPrice = p
	o.fetchedAt = time.Now()
	o.mu.Unlock()

	return Quote{NanoPerCharge: nano, TonPrice: p, Currency: o.currency, CentsPerCharge: o.centsPerCharge}
}

func (o *Oracle) NanoPerCharge(ctx context.Context) int64 {
	return o.Quote(ctx).NanoPerCharge
}

func (o *Oracle) fetchPrice(ctx context.Context) (float64, error) {
	endpoint := fmt.Sprintf("%s/api/v3/simple/price?ids=%s&vs_currencies=%s",
		o.baseURL, url.QueryEscape(o.coinID), url.QueryEscape(o.currency))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := o.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("price http %d", resp.StatusCode)
	}

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data[o.coinID][o.currency], nil
}

func nanoPerCharge(priceFiatPerTON float64, centsPerCharge int64) int64 {
	if priceFiatPerTON <= 0 {
		return 0
	}
	return int64(float64(centsPerCharge) * 1e7 / priceFiatPerTON)
}
