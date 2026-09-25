# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发指令 |
| **agent** | 本地执行手 | 券商下单，无策略代码 |
| **lab** | 算力实验室 | GA 进化与回测 |

## 开发进度

- [x] Phase 0–13 — 垂直切片
- [x] **P0 — Paper Demo 固化 + CI**
- [x] **真实行情喂入（eastmoney feed）**
- [x] **Live broker 适配骨架**
- [x] **Lab / GA 作业 API**
- [x] **TV MCP → klines/import 小工具（cmd/tvimport）**
- [x] **GitHub Codespaces 可运行（见 docs/CODESPACES.md）**

## 15 分钟 Paper Demo

```bash
export QS_JWT_SECRET=dev-secret-change-me
docker compose up --build -d
until curl -sf http://127.0.0.1:8080/healthz; do sleep 1; done
START_AGENT=1 ./scripts/demo_paper.sh
```

详见 [`docs/DEMO.md`](docs/DEMO.md)。

## 在 GitHub Codespaces 运行（推荐阅读）

Codespaces 上 **`saas` 容器连 Postgres 曾出现 timeout**，已验证可行路径为：

- Compose 只起 **Postgres + Redis**
- 本机 **`go run ./cmd/saas`**（`QS_DB_HOST=127.0.0.1`）
- 浏览器通过 **PORTS → 8080 → Open in Browser**

完整步骤、登入说明与排错：**[`docs/CODESPACES.md`](docs/CODESPACES.md)**

其他文档：行情 [`docs/MARKET.md`](docs/MARKET.md)；券商 [`docs/BROKER.md`](docs/BROKER.md)；Lab [`docs/LAB.md`](docs/LAB.md)；TV 导入 [`docs/TVIMPORT.md`](docs/TVIMPORT.md)。

## 本地验证 / CI

```bash
export QS_JWT_SECRET=dev-secret-change-me
make ci
```

## License

Private / All Rights Reserved
