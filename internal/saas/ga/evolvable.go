package ga

import "math/rand"

// Evolvable is the strategy-facing GA contract (opaque gene vector).
type Evolvable interface {
	StrategyID() string
	Dim() int
	Sample(rng *rand.Rand) []float64
	Clamp(gene []float64) []float64
	Evaluate(gene []float64, closes []float64, times []int64, cfg Config) (score, maxDD float64, windows [4]float64, trades int)
}
