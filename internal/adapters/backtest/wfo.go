package backtest

import (
	"fmt"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"
)

type WFOConfig struct {
	TrainBars int
	TestBars  int
	StepBars  int
}

type WFOWindow struct {
	Index      int     `json:"index"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	ROI        float64 `json:"roi"`
	MaxDD      float64 `json:"max_dd"`
	TradeCount int     `json:"trade_count"`
	ExcessROI  float64 `json:"excess_roi"`
	BeatDCA    bool    `json:"beat_dca"`
}

type WFOResult struct {
	Windows     []WFOWindow `json:"windows"`
	MedianROI   float64     `json:"median_roi"`
	MedianMaxDD float64     `json:"median_max_dd"`
	MeanExcess  float64     `json:"mean_excess_roi"`
	WindowsBeat int         `json:"windows_beat_dca"`
	WindowCount int         `json:"window_count"`
	PassGate    bool        `json:"pass_gate"`
	GateReason  string      `json:"gate_reason,omitempty"`
}

func RunWFO(strat strategy.Strategy, closes []float64, times []int64, spawn quant.SpawnPoint, microReserve float64, cfg WFOConfig) (WFOResult, error) {
	out := WFOResult{}
	n := len(closes)
	if cfg.TestBars <= 0 {
		return out, fmt.Errorf("test_bars must be > 0")
	}
	if cfg.TrainBars < 0 {
		cfg.TrainBars = 0
	}
	step := cfg.StepBars
	if step <= 0 {
		step = cfg.TestBars
	}
	start := cfg.TrainBars
	if start+cfg.TestBars > n {
		return out, fmt.Errorf("not enough bars: need train(%d)+test(%d), have %d", cfg.TrainBars, cfg.TestBars, n)
	}
	var rois, dds, excess []float64
	idx := 0
	for start+cfg.TestBars <= n {
		end := start + cfg.TestBars
		cSlice := closes[start:end]
		var tSlice []int64
		if times != nil && end <= len(times) {
			tSlice = times[start:end]
		}
		res := Run(strat, cSlice, tSlice, spawn, microReserve)
		rep := Report(res, cSlice, tSlice, spawn)
		w := WFOWindow{Index: idx, Start: start, End: end, ROI: res.ROI, MaxDD: res.MaxDD, TradeCount: res.TradeCount, ExcessROI: rep.ExcessROI, BeatDCA: rep.BeatDCA}
		out.Windows = append(out.Windows, w)
		rois = append(rois, res.ROI)
		dds = append(dds, res.MaxDD)
		excess = append(excess, rep.ExcessROI)
		if rep.BeatDCA {
			out.WindowsBeat++
		}
		idx++
		start += step
	}
	out.WindowCount = len(out.Windows)
	if out.WindowCount == 0 {
		return out, fmt.Errorf("no WFO windows produced")
	}
	out.MedianROI = median(rois)
	out.MedianMaxDD = median(dds)
	sum := 0.0
	for _, e := range excess {
		sum += e
	}
	out.MeanExcess = sum / float64(len(excess))
	out.PassGate = false
	switch {
	case out.MedianMaxDD > 0.25:
		out.GateReason = "median_max_dd>0.25"
	case out.WindowsBeat*2 >= out.WindowCount && out.MeanExcess >= 0:
		out.PassGate = true
		out.GateReason = "majority_beat_dca_and_nonneg_mean_excess"
	case out.MedianROI > 0 && out.MeanExcess > 0:
		out.PassGate = true
		out.GateReason = "positive_median_roi_and_mean_excess"
	default:
		out.GateReason = "insufficient_oos_edge"
	}
	return out, nil
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	for i := 1; i < len(cp); i++ {
		j := i
		for j > 0 && cp[j] < cp[j-1] {
			cp[j], cp[j-1] = cp[j-1], cp[j]
			j--
		}
	}
	m := len(cp) / 2
	if len(cp)%2 == 0 {
		return (cp[m-1] + cp[m]) / 2
	}
	return cp[m]
}
