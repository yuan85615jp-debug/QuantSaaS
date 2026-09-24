package strategy

import "github.com/yuan85615jp-debug/QuantSaaS/internal/quant"

// Manifest describes a strategy template (product sheet).
type Manifest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	IsSpot  bool   `json:"is_spot"`
}

// Strategy is the pure decision interface. Implementations must be pure:
// no network, no DB, no timers, no file I/O.
type Strategy interface {
	Manifest() Manifest
	Step(in quant.StrategyInput) quant.StrategyOutput
}
