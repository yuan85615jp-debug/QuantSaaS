package main

import (
	"encoding/json"
	"testing"
)

func TestMapInterval(t *testing.T) {
	if mapInterval("1D") != "1d" {
		t.Fatalf("1D")
	}
	if mapInterval("1m") != "1m" {
		t.Fatalf("1m")
	}
}

func TestTickerFromTV(t *testing.T) {
	if tickerFromTV("SSE:510300") != "510300" {
		t.Fatal("ticker")
	}
}

func TestParseDirectBars(t *testing.T) {
	raw := `[{"t":1700000000,"o":1,"h":2,"l":0.5,"c":1.5,"v":100}]`
	bars, err := parseOHLCVPayload([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 || bars[0].C != 1.5 {
		t.Fatalf("%+v", bars)
	}
	out := toImportBars(bars)
	if out[0].OpenTime != 1700000000*1000 {
		t.Fatalf("ms convert %d", out[0].OpenTime)
	}
}

func TestParseRPCEnvelope(t *testing.T) {
	payload := map[string]any{
		"result": map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": `[{"t":1700000000,"o":1,"h":1,"l":1,"c":2,"v":0}]`},
			},
		},
	}
	b, _ := json.Marshal(payload)
	bars, err := parseOHLCVPayload(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 || bars[0].C != 2 {
		t.Fatalf("%+v", bars)
	}
}
