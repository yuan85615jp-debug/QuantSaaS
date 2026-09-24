package ga

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
)

type Engine struct {
	Evo Evolvable
	Cfg Config
	RNG *rand.Rand
}

func NewEngine(evo Evolvable, cfg Config, seed int64) *Engine {
	cfg = cfg.Normalize()
	return &Engine{Evo: evo, Cfg: cfg, RNG: rand.New(rand.NewSource(seed))}
}

func (e *Engine) Run(closes []float64, times []int64) Result {
	cfg := e.Cfg
	pop := e.initPop()
	e.evaluateAll(pop, closes, times)
	history := make([]float64, 0, cfg.MaxGenerations)
	bestEver := pop[0]
	noImprove := 0
	mutProb, mutScale := cfg.MutationProb, cfg.MutationScale

	for gen := 0; gen < cfg.MaxGenerations; gen++ {
		sort.Slice(pop, func(i, j int) bool { return pop[i].Score > pop[j].Score })
		if pop[0].Score > bestEver.Score {
			bestEver = cloneInd(pop[0])
			noImprove = 0
		} else {
			noImprove++
			if noImprove >= 4 {
				mutProb = math.Min(0.55, mutProb*1.25)
				mutScale = math.Min(3.0, mutScale*1.25)
			}
		}
		history = append(history, pop[0].Score)

		next := make([]Individual, 0, cfg.PopSize)
		eliteN := int(math.Max(1, math.Ceil(float64(cfg.PopSize)*cfg.EliteRatio)))
		for i := 0; i < eliteN && i < len(pop); i++ {
			next = append(next, cloneInd(pop[i]))
		}
		for len(next) < cfg.PopSize {
			p1 := e.tournament(pop)
			p2 := e.tournament(pop)
			child := e.crossover(p1.Gene, p2.Gene)
			child = e.mutate(child, mutProb, mutScale)
			child = e.Evo.Clamp(child)
			next = append(next, Individual{Gene: child})
		}
		pop = next
		e.evaluateAll(pop, closes, times)
	}

	sort.Slice(pop, func(i, j int) bool { return pop[i].Score > pop[j].Score })
	if pop[0].Score > bestEver.Score {
		bestEver = cloneInd(pop[0])
	}
	return Result{Best: bestEver, Generation: cfg.MaxGenerations, History: history}
}

func (e *Engine) initPop() []Individual {
	cfg := e.Cfg
	pop := make([]Individual, cfg.PopSize)
	pop[0] = Individual{Gene: e.Evo.Clamp(e.Evo.Sample(e.RNG))}
	seed := pop[0].Gene
	for i := 1; i < cfg.PopSize; i++ {
		r := e.RNG.Float64()
		switch {
		case r < 0.1:
			pop[i] = Individual{Gene: e.Evo.Clamp(append([]float64(nil), seed...))}
		case r < 0.5:
			g := e.mutate(append([]float64(nil), seed...), 0.15, 1.5)
			pop[i] = Individual{Gene: e.Evo.Clamp(g)}
		default:
			pop[i] = Individual{Gene: e.Evo.Clamp(e.Evo.Sample(e.RNG))}
		}
	}
	return pop
}

func (e *Engine) evaluateAll(pop []Individual, closes []float64, times []int64) {
	workers := runtime.NumCPU()
	if workers > len(pop) {
		workers = len(pop)
	}
	if workers < 1 {
		workers = 1
	}
	ch := make(chan int, len(pop))
	for i := range pop {
		ch <- i
	}
	close(ch)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range ch {
				sc, dd, wins, tr := e.Evo.Evaluate(pop[i].Gene, closes, times, e.Cfg)
				pop[i].Score, pop[i].MaxDD, pop[i].WindowScores, pop[i].Trades = sc, dd, wins, tr
			}
		}()
	}
	wg.Wait()
}

func (e *Engine) tournament(pop []Individual) Individual {
	best := pop[e.RNG.Intn(len(pop))]
	for i := 1; i < e.Cfg.TournamentSize; i++ {
		c := pop[e.RNG.Intn(len(pop))]
		if c.Score > best.Score {
			best = c
		}
	}
	return best
}

func (e *Engine) crossover(a, b []float64) []float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		if e.RNG.Float64() < 0.5 {
			out[i] = a[i]
		} else {
			out[i] = b[i]
		}
	}
	return out
}

func (e *Engine) mutate(g []float64, prob, scale float64) []float64 {
	out := append([]float64(nil), g...)
	for i := range out {
		if e.RNG.Float64() < prob {
			out[i] += e.RNG.NormFloat64() * scale * (0.05 + 0.1*math.Abs(out[i]))
		}
	}
	return out
}

func cloneInd(in Individual) Individual {
	return Individual{
		Gene: append([]float64(nil), in.Gene...), Score: in.Score, MaxDD: in.MaxDD,
		WindowScores: in.WindowScores, Trades: in.Trades,
	}
}
