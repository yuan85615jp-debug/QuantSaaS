package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// EastMoneyProvider pulls A-share / ETF klines from eastmoney push2his (no API key).
type EastMoneyProvider struct {
	HTTP    *http.Client
	BaseURL string // optional override for tests
}

func NewEastMoney() *EastMoneyProvider {
	return &EastMoneyProvider{
		HTTP:    &http.Client{Timeout: 15 * time.Second},
		BaseURL: "https://push2his.eastmoney.com",
	}
}

func (p *EastMoneyProvider) Name() string { return "eastmoney" }

func eastMoneySecID(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	s = strings.TrimSuffix(s, ".SS")
	s = strings.TrimSuffix(s, ".SZ")
	s = strings.TrimSuffix(s, ".SH")
	if len(s) == 0 {
		return ""
	}
	if strings.HasPrefix(s, "5") || strings.HasPrefix(s, "6") || strings.HasPrefix(s, "9") {
		return "1." + s
	}
	return "0." + s
}

func eastMoneyKLT(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1m":
		return "1"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h", "60m":
		return "60"
	case "1d", "d", "day":
		return "101"
	case "1w", "week":
		return "102"
	default:
		return "1"
	}
}

type emKlineResp struct {
	Data *struct {
		Code   string   `json:"code"`
		Klines []string `json:"klines"`
	} `json:"data"`
}

func (p *EastMoneyProvider) FetchKlines(ctx context.Context, symbol, interval string, limit int) ([]BarInput, error) {
	secid := eastMoneySecID(symbol)
	if secid == "" {
		return nil, fmt.Errorf("invalid symbol %q", symbol)
	}
	if limit <= 0 {
		limit = 120
	}
	if limit > 500 {
		limit = 500
	}
	klt := eastMoneyKLT(interval)

	base := p.BaseURL
	if base == "" {
		base = "https://push2his.eastmoney.com"
	}
	u, _ := url.Parse(strings.TrimRight(base, "/") + "/api/qt/stock/kline/get")
	q := u.Query()
	q.Set("secid", secid)
	q.Set("fields1", "f1,f2,f3,f4,f5,f6")
	q.Set("fields2", "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61")
	q.Set("klt", klt)
	q.Set("fqt", "1")
	q.Set("end", "20500101")
	q.Set("lmt", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "QuantSaaS/1.0 (+market-feed)")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	req.Header.Set("Accept", "application/json")

	client := p.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var parsed emKlineResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("eastmoney decode: %w", err)
	}
	if parsed.Data == nil || len(parsed.Data.Klines) == 0 {
		return nil, fmt.Errorf("eastmoney: empty klines for %s", symbol)
	}

	out := make([]BarInput, 0, len(parsed.Data.Klines))
	for _, line := range parsed.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		ot, err := parseEMTime(parts[0], interval)
		if err != nil {
			continue
		}
		o, _ := strconv.ParseFloat(parts[1], 64)
		c, _ := strconv.ParseFloat(parts[2], 64)
		h, _ := strconv.ParseFloat(parts[3], 64)
		l, _ := strconv.ParseFloat(parts[4], 64)
		v, _ := strconv.ParseFloat(parts[5], 64)
		if c <= 0 {
			continue
		}
		out = append(out, BarInput{
			OpenTime: ot, Open: o, High: h, Low: l, Close: c, Volume: v,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eastmoney: no parseable bars for %s", symbol)
	}
	return out, nil
}

func parseEMTime(s, interval string) (int64, error) {
	s = strings.TrimSpace(s)
	layouts := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02"}
	var t time.Time
	var err error
	loc := time.FixedZone("CST", 8*3600)
	for _, layout := range layouts {
		t, err = time.ParseInLocation(layout, s, loc)
		if err == nil {
			if strings.EqualFold(interval, "1d") || strings.EqualFold(interval, "d") || strings.EqualFold(interval, "day") {
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
			}
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("bad time %q", s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
