# Phase 3 — 量化数学基础层

包路径：`internal/quant`（纯函数，无 IO）

| 文件 | 内容 |
|------|------|
| math.go | LogReturn / EMA / SMA / MaxDrawdown / ROI / Clamp |
| bar.go | Bar + Closes/Times |
| portfolio.go | Dead/Float/ColdSealed、TotalEquity、SpendableCNY、CurrentMicroWeight |
| spawn.go | SpawnPoint、费率、LotStep 截断 |
| sigmoid.go | TargetWeight、TheoreticalCNY、PDESignal |
| dca.go | GhostDCA + MonthBoundaries |
| intent.go | StrategyInput / StrategyOutput / OrderIntent / ReleaseIntent |

单元测试：`go test ./internal/quant/ -count=1`（本地已通过）
