package quant

// Bar is a single OHLCV candle. Strategies that set Manifest.IsSpot==true
// must not depend on Bar inside Step(); ACL strips to close series outside.
type Bar struct {
	OpenTime int64 // unix ms
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

// Closes extracts close prices (dimensionless series for indicators).
func Closes(bars []Bar) []float64 {
	out := make([]float64, len(bars))
	for i := range bars {
		out[i] = bars[i].Close
	}
	return out
}

// Times extracts open timestamps in ms.
func Times(bars []Bar) []int64 {
	out := make([]int64, len(bars))
	for i := range bars {
		out[i] = bars[i].OpenTime
	}
	return out
}
