package emulate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	Success       bool
	TotalFeesNano int64
	OutMessages   int
	Destinations  []string
}

type apiAddress struct {
	Address string `json:"address"`
}

type apiTransaction struct {
	Success   bool       `json:"success"`
	Aborted   bool       `json:"aborted"`
	TotalFees int64      `json:"total_fees"`
	Account   apiAddress `json:"account"`
}

type apiTrace struct {
	Transaction apiTransaction `json:"transaction"`
	Children    []apiTrace     `json:"children"`
}

func (c *Client) EmulateTrace(ctx context.Context, bocBase64 string, ignoreSignature bool) (*Result, error) {
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
		return nil, fmt.Errorf("%s", e.Error)
	}

	var trace apiTrace
	if err := json.NewDecoder(resp.Body).Decode(&trace); err != nil {
		return nil, err
	}

	return summarize(trace), nil
}

func summarize(root apiTrace) *Result {
	res := &Result{Success: true, Destinations: []string{}}
	walk(root, res, true)
	return res
}

func walk(t apiTrace, res *Result, isRoot bool) {
	res.TotalFeesNano += t.Transaction.TotalFees
	if !t.Transaction.Success || t.Transaction.Aborted {
		res.Success = false
	}
	if !isRoot && t.Transaction.Account.Address != "" {
		res.Destinations = append(res.Destinations, t.Transaction.Account.Address)
		res.OutMessages++
	}
	for _, child := range t.Children {
		walk(child, res, false)
	}
}
