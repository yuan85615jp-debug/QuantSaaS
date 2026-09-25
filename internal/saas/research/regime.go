package research

import "fmt"

// Posture is a coarse offense/defense recommendation for operators (not an order signal).
type Posture string

const (
	PostureOffense Posture = "offense"
	PostureNeutral Posture = "neutral"
	PostureDefense Posture = "defense"
)

type Input struct {
	VIX             *float64 `json:"vix,omitempty"`
	FearScore       *float64 `json:"fear_score,omitempty"`
	MarksCycle      *float64 `json:"marks_cycle,omitempty"`
	InstanceRunning bool     `json:"instance_running,omitempty"`
}

type Advice struct {
	Posture    Posture  `json:"posture"`
	Score      float64  `json:"score"`
	Summary    string   `json:"summary"`
	Actions    []string `json:"actions"`
	InputsUsed []string `json:"inputs_used"`
	Disclaimer string   `json:"disclaimer"`
}

func Advise(in Input) Advice {
	var weighted, weight float64
	used := []string{}
	if in.MarksCycle != nil {
		v := clamp(*in.MarksCycle, 0, 100)
		weighted += v * 0.45
		weight += 0.45
		used = append(used, "marks_cycle")
	}
	if in.VIX != nil {
		def := clamp((*in.VIX-12)/28*100, 0, 100)
		weighted += def * 0.30
		weight += 0.30
		used = append(used, "vix")
	}
	if in.FearScore != nil {
		def := 100 - clamp(*in.FearScore, 0, 100)
		weighted += def * 0.25
		weight += 0.25
		used = append(used, "fear_score")
	}
	score := 50.0
	if weight > 0 {
		score = weighted / weight
	}
	a := Advice{Score: score, InputsUsed: used, Disclaimer: "Research advisory only. Does not submit orders; Agent remains execution-only."}
	switch {
	case score <= 35:
		a.Posture = PostureOffense
		a.Summary = "Research blend leans offense (lower defense score)."
		a.Actions = []string{"OK to keep lunar instances RUNNING if capital plan allows", "Prefer completing paper/WFO gates before promoting genes", "Avoid raising micro_reserve solely out of fear"}
	case score >= 65:
		a.Posture = PostureDefense
		a.Summary = "Research blend leans defense (elevated stress or late-cycle)."
		a.Actions = []string{"Consider pausing new instance capital_quota increases", "Keep Agent dry_run / paper until conditions normalize", "Review MaxDD gates before Lab promote=true"}
		if in.InstanceRunning {
			a.Actions = append(a.Actions, "Running instances may stay up; tighten manual SendTrade size")
		}
	default:
		a.Posture = PostureNeutral
		a.Summary = "Research blend is neutral — no strong offense/defense tilt."
		a.Actions = []string{"Maintain current RUNNING set", "Continue seed/sync k_lines and routine Lab test_mode jobs"}
	}
	if len(used) == 0 {
		a.Summary = "No research inputs provided; default neutral."
		a.Actions = []string{"Pass vix, fear_score, and/or marks_cycle query params (from AlphaGBM skills)", "Example: GET /api/v1/research/regime?vix=18&fear_score=40&marks_cycle=55"}
	}
	return a
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ParseOptionalFloat(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	return &f, nil
}
