// Command tvimport pulls OHLCV via TradingView official MCP get_ohlcv
// and POSTs them into QuantSaaS /api/v1/klines/import.
//
// Auth: TRADINGVIEW_MCP_TOKEN (OAuth access token) or -file with saved JSON.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	tvSymbol := flag.String("tv-symbol", "", "TradingView symbol EXCHANGE:TICKER (e.g. SSE:510300)")
	qsSymbol := flag.String("qs-symbol", "", "QuantSaaS symbol in k_lines (default: ticker of -tv-symbol)")
	interval := flag.String("interval", "1D", "TV interval: 1m 5m 15m 30m 1h 4h 1D 1W M")
	count := flag.Int("count", 300, "number of bars (max 5000)")
	file := flag.String("file", "", "saved get_ohlcv JSON (skips MCP fetch)")
	mcpURL := flag.String("mcp-url", envOr("TRADINGVIEW_MCP_URL", "https://mcp.tradingview.com/mcp"), "official MCP endpoint")
	mcpToken := flag.String("mcp-token", os.Getenv("TRADINGVIEW_MCP_TOKEN"), "OAuth access token")
	baseURL := flag.String("saas", envOr("QS_BASE_URL", "http://127.0.0.1:8080"), "QuantSaaS base URL")
	email := flag.String("email", os.Getenv("QS_EMAIL"), "login email")
	password := flag.String("password", os.Getenv("QS_PASSWORD"), "login password")
	token := flag.String("token", os.Getenv("QS_TOKEN"), "existing SaaS JWT")
	dryRun := flag.Bool("dry-run", false, "fetch/parse only; do not POST import")
	flag.Parse()

	if *tvSymbol == "" && *file == "" {
		fmt.Fprintln(os.Stderr, "error: -tv-symbol or -file required")
		flag.Usage()
		os.Exit(2)
	}
	if *count <= 0 {
		*count = 300
	}
	if *count > 5000 {
		*count = 5000
	}
	if *qsSymbol == "" && *tvSymbol != "" {
		*qsSymbol = tickerFromTV(*tvSymbol)
	}
	if *qsSymbol == "" {
		fmt.Fprintln(os.Stderr, "error: -qs-symbol required when using -file without -tv-symbol")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var rawBars []tvBar
	var err error
	if *file != "" {
		rawBars, err = loadBarsFromFile(*file)
	} else {
		if strings.TrimSpace(*mcpToken) == "" {
			fmt.Fprintln(os.Stderr, "error: TRADINGVIEW_MCP_TOKEN / -mcp-token required for live MCP fetch")
			fmt.Fprintln(os.Stderr, "hint: OAuth via Claude/Cursor, or use -file with saved get_ohlcv JSON")
			os.Exit(2)
		}
		rawBars, err = fetchOHLCV(ctx, *mcpURL, *mcpToken, *tvSymbol, *interval, *count)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	if len(rawBars) == 0 {
		fmt.Fprintln(os.Stderr, "no bars returned")
		os.Exit(1)
	}

	qsInterval := mapInterval(*interval)
	bars := toImportBars(rawBars)
	fmt.Printf("parsed %d bars symbol=%s interval=%s first_t=%d last_t=%d last_c=%.6f\n",
		len(bars), strings.ToUpper(*qsSymbol), qsInterval, bars[0].OpenTime, bars[len(bars)-1].OpenTime, bars[len(bars)-1].Close)

	if *dryRun {
		fmt.Println("dry-run: skip SaaS import")
		return
	}

	saasTok := strings.TrimSpace(*token)
	if saasTok == "" {
		if *email == "" || *password == "" {
			fmt.Fprintln(os.Stderr, "error: SaaS -token or -email/-password required for import")
			os.Exit(2)
		}
		saasTok, err = saasLogin(ctx, *baseURL, *email, *password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "saas login: %v\n", err)
			os.Exit(1)
		}
	}

	imported, err := saasImport(ctx, *baseURL, saasTok, *qsSymbol, qsInterval, bars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("imported %d bars into %s %s\n", imported, strings.ToUpper(*qsSymbol), qsInterval)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func tickerFromTV(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, ":"); i >= 0 && i+1 < len(s) {
		return s[i+1:]
	}
	return s
}

func mapInterval(tv string) string {
	switch strings.TrimSpace(tv) {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "30m":
		return "30m"
	case "1h":
		return "1h"
	case "4h":
		return "4h"
	case "1D", "1d", "D", "d":
		return "1d"
	case "1W", "1w", "W", "w":
		return "1w"
	case "M", "1M", "1mo", "month":
		return "1M"
	default:
		return strings.ToLower(tv)
	}
}

type tvBar struct {
	T float64 `json:"t"`
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"`
}

type importBar struct {
	OpenTime int64   `json:"open_time"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
}

func toImportBars(in []tvBar) []importBar {
	out := make([]importBar, 0, len(in))
	for _, b := range in {
		if b.C <= 0 && b.O <= 0 {
			continue
		}
		ot := int64(b.T)
		if ot <= 1_000_000_000_000 {
			ot = ot * 1000
		}
	o, h, l, c := b.O, b.H, b.L, b.C
		if c <= 0 {
			c = o
		}
		if o <= 0 {
			o = c
		}
		if h <= 0 {
			h = c
		}
		if l <= 0 {
			l = c
		}
		out = append(out, importBar{OpenTime: ot, Open: o, High: h, Low: l, Close: c, Volume: b.V})
	}
	return out
}

func loadBarsFromFile(path string) ([]tvBar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOHLCVPayload(data)
}

func fetchOHLCV(ctx context.Context, mcpURL, token, symbol, interval string, count int) ([]tvBar, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_ohlcv",
			"arguments": map[string]any{
				"symbol": symbol, "interval": interval, "count": count, "summary": false,
			},
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp status %d: %s", resp.StatusCode, truncate(string(body), 400))
	}
	payload := body
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") || bytes.Contains(body, []byte("data:")) {
		payload = extractSSEData(body)
	}
	return parseOHLCVPayload(payload)
}

func extractSSEData(body []byte) []byte {
	var chunks []json.RawMessage
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}
			chunks = append(chunks, json.RawMessage(data))
		}
	}
	if len(chunks) == 0 {
		return body
	}
	return chunks[len(chunks)-1]
}

func parseOHLCVPayload(data []byte) ([]tvBar, error) {
	data = bytes.TrimSpace(data)
	var direct []tvBar
	if err := json.Unmarshal(data, &direct); err == nil && len(direct) > 0 && (direct[0].T != 0 || direct[0].C != 0) {
		return direct, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	if r, ok := root["result"]; ok {
		return parseOHLCVPayload(r)
	}
	if c, ok := root["content"]; ok {
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(c, &parts); err == nil {
			for _, p := range parts {
				if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
					if bars, err := parseOHLCVPayload([]byte(p.Text)); err == nil && len(bars) > 0 {
						return bars, nil
					}
				}
			}
		}
	}
	for _, key := range []string{"bars", "data", "ohlcv", "candles"} {
		if v, ok := root[key]; ok {
			if bars, err := parseOHLCVPayload(v); err == nil && len(bars) > 0 {
				return bars, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find OHLCV bars in payload")
}

func saasLogin(ctx context.Context, base, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("empty token")
	}
	return out.Token, nil
}

func saasImport(ctx context.Context, base, token, symbol, interval string, bars []importBar) (int, error) {
	payload, err := json.Marshal(map[string]any{"symbol": symbol, "interval": interval, "bars": bars})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/klines/import", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out struct {
		Imported int `json:"imported"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Imported == 0 {
		out.Imported = len(bars)
	}
	return out.Imported, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
