package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LastClose fetches the latest close for symbol from SaaS GET /api/v1/klines/last.
func LastClose(ctx context.Context, baseURL, token, symbol string) (float64, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	u, err := url.Parse(baseURL + "/api/v1/klines/last")
	if err != nil {
		return 0, err
	}
	q := u.Query()
	q.Set("symbol", symbol)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("klines/last status %d", resp.StatusCode)
	}
	var body struct {
		Close float64 `json:"close"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if body.Close <= 0 {
		return 0, fmt.Errorf("invalid close")
	}
	return body.Close, nil
}
