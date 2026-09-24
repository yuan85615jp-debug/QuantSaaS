package lunar

import "github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"

func init() {
	strategy.Register(StrategyID, func() strategy.Strategy {
		return NewDefault()
	})
}
