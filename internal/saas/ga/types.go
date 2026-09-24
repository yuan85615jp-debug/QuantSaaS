package ga

import "github.com/yuan85615jp-debug/QuantSaaS/internal/quant"

type Config struct {
	PopSize        int
	MaxGenerations int
	TournamentSize int
	EliteRatio     float64
	MutationProb   float64
	MutationScale  float64
	WAll, W5y, W2y, W6m float64
	FatalMaxDD     float64
	Spawn          quant.SpawnPoint
	TestMode       bool
}

func DefaultConfig() Config {
	return Config{
		PopSize: 50, MaxGenerations: 15, TournamentSize: 3, EliteRatio: 0.05,
		MutationProb: 0.15, MutationScale: 1.0,
		WAll: 0.40, W5y: 0.30, W2y: 0.20, W6m: 0.10,
		FatalMaxDD: 0.88, Spawn: quant.DefaultSpawnPoint(),
	}
}

func (c Config) Normalize() Config {
	if c.TestMode {
		c.PopSize, c.MaxGenerations = 10, 3
	}
	if c.PopSize < 4 {
		c.PopSize = 4
	}
	if c.MaxGenerations < 1 {
		c.MaxGenerations = 1
	}
	if c.TournamentSize < 2 {
		c.TournamentSize = 2
	}
	if c.EliteRatio <= 0 {
		c.EliteRatio = 0.05
	}
	if c.MutationProb <= 0 {
		c.MutationProb = 0.15
	}
	if c.WAll+c.W5y+c.W2y+c.W6m == 0 {
		c.WAll, c.W5y, c.W2y, c.W6m = 0.4, 0.3, 0.2, 0.1
	}
	if c.FatalMaxDD <= 0 {
		c.FatalMaxDD = 0.88
	}
	if c.Spawn.InitialCapital <= 0 {
		c.Spawn = quant.DefaultSpawnPoint()
	}
	return c
}

type Individual struct {
	Gene         []float64
	Score        float64
	MaxDD        float64
	WindowScores [4]float64
	Trades       int
}

type Result struct {
	Best       Individual
	Generation int
	History    []float64
}
