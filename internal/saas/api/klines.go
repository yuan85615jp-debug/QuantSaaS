package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/market"
)

type importKlinesBody struct {
	Symbol   string            `json:"symbol"`
	Interval string            `json:"interval"`
	Bars     []market.BarInput `json:"bars"`
}

type seedKlinesBody struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	Bars     int     `json:"bars"`
	StartPx  float64 `json:"start_px"`
}

func (s *Server) handleImportKlines(w http.ResponseWriter, r *http.Request) {
	if s.Market == nil {
		writeError(w, http.StatusServiceUnavailable, "market service not configured")
		return
	}
	var body importKlinesBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	n, err := s.Market.ImportBars(body.Symbol, body.Interval, body.Bars)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": n, "symbol": strings.ToUpper(body.Symbol)})
}

func (s *Server) handleSeedKlines(w http.ResponseWriter, r *http.Request) {
	if s.Market == nil {
		writeError(w, http.StatusServiceUnavailable, "market service not configured")
		return
	}
	var body seedKlinesBody
	if err := decodeJSON(r, &body); err != nil {
		body = seedKlinesBody{}
	}
	n, err := s.Market.SeedSynthetic(market.SeedOpts{
		Symbol:   body.Symbol,
		Interval: body.Interval,
		Bars:     body.Bars,
		StartPx:  body.StartPx,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sym := body.Symbol
	if sym == "" {
		sym = "510300"
	}
	px, ts, _ := s.Market.LastClose(sym)
	writeJSON(w, http.StatusOK, map[string]any{
		"seeded": n, "symbol": strings.ToUpper(sym),
		"last_close": px, "last_open_time": ts,
	})
}

func (s *Server) handleListKlines(w http.ResponseWriter, r *http.Request) {
	if s.Market == nil {
		writeError(w, http.StatusServiceUnavailable, "market service not configured")
		return
	}
	symbol := r.URL.Query().Get("symbol")
	interval := r.URL.Query().Get("interval")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol required")
		return
	}
	rows, err := s.Market.ListBars(symbol, interval, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows, "count": len(rows)})
}

func (s *Server) handleLastKline(w http.ResponseWriter, r *http.Request) {
	if s.Market == nil {
		writeError(w, http.StatusServiceUnavailable, "market service not configured")
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol required")
		return
	}
	px, ts, err := s.Market.LastClose(symbol)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"symbol": strings.ToUpper(symbol), "close": px, "open_time": ts,
	})
}
