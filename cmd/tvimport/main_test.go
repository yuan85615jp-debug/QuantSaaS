package main

import (
	"encoding/json"
	"testing"
)

func TestMapInterval(t *testing.T) {
	if mapInterval("1D") != "1d" {
		t.Fatalf("1D")
	}
}

func TestParseSymbolLine(t *testing.T) {
	p, err := parseSymbolLine("NASDAQ:AAPL")
	if err != nil || p.QS != "AAPL" || p.TV != "NASDAQ:AAPL" {
		t.Fatalf("%+v %v", p, err)
	}
	p, err = parseSymbolLine("SSE:510300=510300")
	if err != nil || p.TV != "SSE:510300" || p.QS != "510300" {
		t.Fatalf("%+v %v", p, err)
	}
	p, err = parseSymbolLine("SSE:510300:ETF510300")
	if err != nil || p.TV != "SSE:510300" || p.QS != "ETF510300" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestParseRPCEnvelope(t *testing.T) {
	payload := map[string]any{
		"result": map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "[{\"t\":1700000000,\"o\":1,\"h\":1,\"l\":1,\"c\":2,\"v\":0}]"},
			},
		},
	}
	b, _ := json.Marshal(payload)
	bars, err := parseOHLCVPayload(b)
	if err != nil || len(bars) != 1 || bars[0].C != 2 {
		t.Fatalf("%+v %v", bars, err)
	}
}

func TestToImportBarsMS(t *testing.T) {
	out := toImportBars([]tvBar{{T: 1700000000, C: 1, O: 1, H: 1, L: 1}})
	if out[0].OpenTime != 1700000000*1000 {
		t.Fatal(out[0].OpenTime)
	}
}
