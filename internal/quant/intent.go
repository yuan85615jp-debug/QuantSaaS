package quant

// Side of a theoretical order produced by Step().
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
	SideFlat Side = "FLAT"
)

type LotKind string

const (
	LotDead   LotKind = "DEAD_STACK"
	LotFloat  LotKind = "FLOATING"
	LotSealed LotKind = "COLD_SEALED"
)

// OrderIntent is a pure decision from Step(); execution is outside the strategy.
type OrderIntent struct {
	Side   Side
	Engine string  // MACRO | MICRO
	Qty    float64 // shares (pre lot-normalize)
	Note   string
}

// ReleaseIntent moves Dead → Float on the SaaS ledger only (no TradeCommand).
type ReleaseIntent struct {
	Qty  float64
	Note string
}

// StrategyInput is the only snapshot Step() may read.
type StrategyInput struct {
	NowMs             int64
	Price             float64
	Closes            []float64
	Times             []int64
	Portfolio         Portfolio
	Spawn             SpawnPoint
	Runtime           map[string]float64
	MicroReservePct   float64
	Sigmoid           SigmoidParams
	Kp, Kv, Ka        float64
	MinTradeThreshold float64
}

// StrategyOutput is the only thing Step() may produce.
type StrategyOutput struct {
	Orders   []OrderIntent
	Releases []ReleaseIntent
	Runtime  map[string]float64
	Signal   float64
	TargetW  float64
}
