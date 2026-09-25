# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发指令 |
| **agent** | 本地执行手 | 券商下单，无策略代码 |
| **lab** | 算力实验室 | GA 进化与回测 |

## 开发进度

- [x] Phase 0–13 — 垂直切片（引擎 / 实例 / Agent / WS / API / Ticker / SPA / Docker / K线）
- [x] **P0 — Paper Demo 固化 + CI**
- [x] **真实行情喂入（eastmoney feed）**

## 15 分钟 Paper Demo

```bash
export QS_JWT_SECRET=dev-secret-change-me
docker compose up --build -d
until curl -sf http://127.0.0.1:8080/healthz; do sleep 1; done
START_AGENT=1 ./scripts/demo_paper.sh
```

期望输出含：`SUCCESS: instance … portfolio updated`。详见 [`docs/DEMO.md`](docs/DEMO.md)。行情源见 [`docs/MARKET.md`](docs/MARKET.md)。

## 本地验证 / CI

```bash
export QS_JWT_SECRET=dev-secret-change-me
make ci
```

GitHub Actions：`.github/workflows/ci.yml`

## License

Private / All Rights Reserved
