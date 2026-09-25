# Research Ops（落地顺序）

把 AlphaGBM 研究信号、Lab 回测可信度与 Paper 执行串成可重复流程。**不改变铁律：Step 仅在 SaaS；Agent 无策略。**

## 1. 日常姿态（不改策略代码）

外部取得 `vix` / `fear_score` / `marks_cycle`（例如 AlphaGBM skills）后：

```bash
curl -s "http://127.0.0.1:8080/api/v1/research/regime?vix=18&fear_score=42&marks_cycle=55&instance_running=true" \
  -H "Authorization: Bearer $TOKEN"
```

返回 `posture`: `offense|neutral|defense`。**不会下单。**

## 2. Lab：Benchmark + Walk-Forward

每次任务 `result.benchmark`：策略 vs Ghost DCA。

打开 WFO：

```json
{
  "strategy_id": "lunar",
  "symbol": "510300",
  "config": {
    "interval": "1d",
    "bars": 300,
    "pop_size": 20,
    "max_generations": 6,
    "test_mode": true,
    "promote": true,
    "wfo": true,
    "wfo_train_bars": 150,
    "wfo_test_bars": 40,
    "wfo_step_bars": 40,
    "require_wfo_pass": true
  }
}
```

`require_wfo_pass=true` 时门禁失败则不 promote（`promote_skipped`）。

## 3. Paper

见 [CODESPACES.md](CODESPACES.md)、[DEMO.md](DEMO.md)。

## 4. 新策略

研究侧验证后再实现 `internal/strategies/.../step.go`（同一 Step）。
