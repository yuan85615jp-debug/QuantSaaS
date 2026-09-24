package quant

import "math"

// LogReturn returns ln(p1/p0). Safe for p0<=0 or p1<=0 → 0.
func LogReturn(p0, p1 float64) float64 {
	if p0 <= 0 || p1 <= 0 {
		return 0
	}
	return math.Log(p1 / p0)
}

// LogReturns computes consecutive log returns for a price series.
func LogReturns(prices []float64) []float64 {
	if len(prices) < 2 {
		return nil
	}
	out := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		out[i-1] = LogReturn(prices[i-1], prices[i])
	}
	return out
}

// Clamp confines x to [lo, hi].
func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// Mean arithmetic mean; empty → 0.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, v := range xs {
		s += v
	}
	return s / float64(len(xs))
}

// StdDev sample standard deviation; n<2 → 0.
func StdDev(xs []float64) float64 {
	n := len(xs)
	if n < 2 {
		return 0
	}
	m := Mean(xs)
	var ss float64
	for _, v := range xs {
		d := v - m
		ss += d * d
	}
	return math.Sqrt(ss / float64(n-1))
}

// EMA exponential moving average. period <= 0 returns copy of xs.
func EMA(xs []float64, period int) []float64 {
	n := len(xs)
	out := make([]float64, n)
	if n == 0 || period <= 0 {
		copy(out, xs)
		return out
	}
	alpha := 2.0 / (float64(period) + 1.0)
	out[0] = xs[0]
	for i := 1; i < n; i++ {
		out[i] = alpha*xs[i] + (1-alpha)*out[i-1]
	}
	return out
}

// SMA simple moving average; values before period-1 are partial means.
func SMA(xs []float64, period int) []float64 {
	n := len(xs)
	out := make([]float64, n)
	if n == 0 || period <= 0 {
		copy(out, xs)
		return out
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += xs[i]
		if i >= period {
			sum -= xs[i-period]
			out[i] = sum / float64(period)
		} else {
			out[i] = sum / float64(i+1)
		}
	}
	return out
}

// MaxDrawdown returns max peak-to-trough decline as a positive ratio of equity curve.
func MaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}
	peak := equity[0]
	maxDD := 0.0
	for _, v := range equity {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			dd := (peak - v) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// ROI simple (end-start)/start; start<=0 → 0.
func ROI(start, end float64) float64 {
	if start <= 0 {
		return 0
	}
	return (end - start) / start
}
