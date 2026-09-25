package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type LiveOpts struct {
	BaseURL   string
	APIKey    string
	APISecret string
	AccountID string
	Timeout   time.Duration
	LotStep   float64
	LotMin    float64
	DryRun    bool
}

type LiveBroker struct {
	mu    sync.Mutex
	opts  LiveOpts
	http  *http.Client
	marks map[string]float64
}

func NewLive(opts LiveOpts) *LiveBroker {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.LotStep <= 0 {
		opts.LotStep = 100
	}
	if opts.LotMin <= 0 {
		opts.LotMin = 100
	}
	if opts.BaseURL == "" || opts.APIKey == "" {
		opts.DryRun = true
	}
	return &LiveBroker{
		opts: opts, http: &http.Client{Timeout: opts.Timeout}, marks: map[string]float64{},
	}
}

func (l *LiveBroker) Name() string {
	if l.opts.DryRun {
		return "live-dryrun"
	}
	return "live"
}

func (l *LiveBroker) SetMarkPrice(symbol string, price float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if price > 0 {
		l.marks[symbol] = price
	}
}

func (l *LiveBroker) PlaceOrder(ctx context.Context, req OrderRequest) (OrderResult, error) {
	out := OrderResult{ClientOrderID: req.ClientOrderID, Symbol: req.Symbol, Side: req.Side, Status: "failed"}
	side := strings.ToUpper(strings.TrimSpace(req.Side))
	if side != "BUY" && side != "SELL" {
		out.ErrorMsg = "invalid side"
		return out, fmt.Errorf("live: invalid side %q", req.Side)
	}
	qty := normalizeLot(req.Qty, l.opts.LotStep, l.opts.LotMin)
	if qty <= 0 {
		out.ErrorMsg = "qty below lot minimum"
		return out, fmt.Errorf("live: qty below lot min")
	}
	payload := map[string]any{
		"client_order_id": req.ClientOrderID,
		"account_id":      l.opts.AccountID,
		"symbol":          req.Symbol,
		"side":            side,
		"qty":             qty,
		"order_type":      strings.ToUpper(strings.TrimSpace(req.OrderType)),
	}
	if payload["order_type"] == "" {
		payload["order_type"] = "MARKET"
	}
	if l.opts.DryRun {
		out.ErrorMsg = "live broker dry-run (no credentials or DryRun=true); order not sent"
		return out, fmt.Errorf("live: dry-run")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		out.ErrorMsg = err.Error()
		return out, err
	}
	url := strings.TrimRight(l.opts.BaseURL, "/") + "/api/v1/orders"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		out.ErrorMsg = err.Error()
		return out, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", l.opts.APIKey)
	if l.opts.APISecret != "" {
		httpReq.Header.Set("X-API-Secret", l.opts.APISecret)
	}
	resp, err := l.http.Do(httpReq)
	if err != nil {
		out.ErrorMsg = err.Error()
		return out, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		out.ErrorMsg = fmt.Sprintf("broker status %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
		return out, fmt.Errorf("live: %s", out.ErrorMsg)
	}
	var parsed struct {
		FilledQty   float64 `json:"filled_qty"`
		FilledPrice float64 `json:"filled_price"`
		Fee         float64 `json:"fee"`
		Status      string  `json:"status"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		out.ErrorMsg = "decode fill: " + err.Error()
		return out, err
	}
	out.FilledQty, out.FilledPrice, out.Fee = parsed.FilledQty, parsed.FilledPrice, parsed.Fee
	out.Status = parsed.Status
	if out.Status == "" {
		out.Status = "filled"
	}
	return out, nil
}

func (l *LiveBroker) Snapshot(ctx context.Context, symbol string) (Position, error) {
	if l.opts.DryRun {
		return Position{Symbol: symbol}, fmt.Errorf("live: dry-run snapshot unavailable")
	}
	url := strings.TrimRight(l.opts.BaseURL, "/") + "/api/v1/accounts/" + l.opts.AccountID + "/positions?symbol=" + symbol
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Position{}, err
	}
	req.Header.Set("X-API-Key", l.opts.APIKey)
	if l.opts.APISecret != "" {
		req.Header.Set("X-API-Secret", l.opts.APISecret)
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return Position{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return Position{}, fmt.Errorf("live snapshot status %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		Symbol string  `json:"symbol"`
		Shares float64 `json:"shares"`
		Cash   float64 `json:"cash"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Position{}, err
	}
	return Position{Symbol: symbol, Shares: parsed.Shares, Cash: parsed.Cash}, nil
}

func normalizeLot(qty, step, min float64) float64 {
	if qty <= 0 {
		return 0
	}
	if step <= 0 {
		if qty < min {
			return 0
		}
		return qty
	}
	q := float64(int(qty/step)) * step
	if q < min {
		return 0
	}
	return q
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
