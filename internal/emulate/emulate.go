package emulate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		hc:      &http.Client{Timeout: 20 * time.Second},
	}
}

type Result struct {
	Success          bool
	TotalFeesNano    int64
	ForwardValueNano int64
	OutMessages      int
	Destinations     []string
}

type apiAddress struct {
	Address string `json:"address"`
}

type apiMessage struct {
	Destination   apiAddress `json:"destination"`
	Value         int64      `json:"value"`
	OpCode        string     `json:"op_code"`
	DecodedOpName string     `json:"decoded_op_name"`
}

type apiTransaction struct {
	Success   bool         `json:"success"`
	Aborted   bool         `json:"aborted"`
	TotalFees int64        `json:"total_fees"`
	Account   apiAddress   `json:"account"`
	OutMsgs   []apiMessage `json:"out_msgs"`
}

type apiTrace struct {
	Transaction apiTransaction `json:"transaction"`
	Children    []apiTrace     `json:"children"`
}

func (c *Client) EmulateTrace(ctx context.Context, bocBase64 string, ignoreSignature bool) (*Result, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(600 * time.Millisecond):
			}
		}
		res, err := c.emulateOnce(ctx, bocBase64, ignoreSignature)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !isTransient(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) emulateOnce(ctx context.Context, bocBase64 string, ignoreSignature bool) (*Result, error) {
	endpoint := c.baseURL + "/v2/traces/emulate"
	if ignoreSignature {
		endpoint += "?ignore_signature_check=true"
	}

	payload, err := json.Marshal(map[string]string{"boc": bocBase64})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = fmt.Sprintf("emulate http %d", resp.StatusCode)
		}
		return nil, &httpError{status: resp.StatusCode, msg: e.Error}
	}

	var trace apiTrace
	if err := json.NewDecoder(resp.Body).Decode(&trace); err != nil {
		return nil, err
	}

	return summarize(trace), nil
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func isTransient(err error) bool {
	var he *httpError
	if errors.As(err, &he) {
		if he.status == http.StatusTooManyRequests || he.status >= 500 {
			return true
		}
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "rate limit") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "adnl") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "eof")
}

func summarize(root apiTrace) *Result {
	res := &Result{Destinations: []string{}}

	if !txOK(root.Transaction) {
		return res
	}
	if len(root.Children) == 0 {
		return res
	}

	userTx := root.Children[0].Transaction
	res.Success = txOK(userTx)
	res.TotalFeesNano = userTx.TotalFees
	res.OutMessages = len(userTx.OutMsgs)
	for _, m := range userTx.OutMsgs {
		if m.Destination.Address != "" {
			res.Destinations = append(res.Destinations, m.Destination.Address)
		}
		if isForwardMsg(m) {
			res.ForwardValueNano += m.Value
		}
	}
	return res
}

func txOK(t apiTransaction) bool {
	return t.Success && !t.Aborted
}

func isForwardMsg(m apiMessage) bool {
	switch strings.ToLower(m.DecodedOpName) {
	case "jetton_transfer", "nft_transfer":
		return true
	}
	switch strings.TrimPrefix(strings.ToLower(m.OpCode), "0x") {
	case "0f8a7ea5", "f8a7ea5", "5fcc3d14":
		return true
	}
	return false
}

type KeyStatus struct {
	KeyConfigured bool
	Endpoint      string
	WithKeyStatus int
	BadKeyStatus  int
	Valid         bool
	Detail        string
}

func (c *Client) probeStatus(ctx context.Context, key string) (int, error) {
	endpoint := c.baseURL + "/v2/blockchain/masterchain-head"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func (c *Client) CheckKey(ctx context.Context) *KeyStatus {
	ks := &KeyStatus{
		KeyConfigured: c.apiKey != "",
		Endpoint:      c.baseURL + "/v2/blockchain/masterchain-head",
	}

	code1, err1 := c.probeStatus(ctx, c.apiKey)
	ks.WithKeyStatus = code1

	code2, _ := c.probeStatus(ctx, "cryptocash-invalid-key-probe")
	ks.BadKeyStatus = code2

	if code1 == 0 {
		if err1 != nil {
			ks.Detail = "tonapi unreachable: " + err1.Error()
		} else {
			ks.Detail = "tonapi unreachable"
		}
		return ks
	}
	if !ks.KeyConfigured {
		ks.Detail = "no TON_API_KEY set — anonymous tier"
		return ks
	}

	switch {
	case code1 == http.StatusOK && code2 == http.StatusUnauthorized:
		ks.Valid = true
		ks.Detail = "key accepted; server rejects wrong keys"
	case code1 == http.StatusUnauthorized:
		ks.Detail = "key rejected (401) — bogus or wrong key"
	case code1 == http.StatusOK && code2 == http.StatusOK:
		ks.Valid = true
		ks.Detail = "key accepted (200); endpoint also allows anonymous, so tier not fully proven here"
	default:
		ks.Detail = fmt.Sprintf("inconclusive: withKey=%d badKey=%d", code1, code2)
	}
	return ks
}
