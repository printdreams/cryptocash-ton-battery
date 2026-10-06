package ton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	endpoint string
	apiKey   string
	hc       *http.Client
}

func NewClient(endpoint, apiKey string) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		hc:       &http.Client{Timeout: 10 * time.Second},
	}
}

type WalletInfo struct {
	Balance  int64
	Seqno    int
	Deployed bool
}

func (c *Client) GetWalletInformation(ctx context.Context, addr string) (*WalletInfo, error) {
	endpoint := fmt.Sprintf("%s/api/v2/getWalletInformation?address=%s", c.endpoint, url.QueryEscape(addr))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		OK     bool `json:"ok"`
		Result struct {
			Balance      json.Number `json:"balance"`
			AccountState string      `json:"account_state"`
			Seqno        *int        `json:"seqno"`
		} `json:"result"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if !body.OK {
		return nil, fmt.Errorf("toncenter: %s", body.Error)
	}

	balance, _ := strconv.ParseInt(body.Result.Balance.String(), 10, 64)
	info := &WalletInfo{
		Balance:  balance,
		Deployed: body.Result.AccountState == "active",
	}
	if body.Result.Seqno != nil {
		info.Seqno = *body.Result.Seqno
	}
	return info, nil
}
