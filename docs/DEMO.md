# Paper Demo（15 分钟路径）

目标：任何人 clone 后看到

```
实例 RUNNING → 下单 → Paper fill → 账本变化
```

## 路径 A：Docker（推荐）

```bash
export QS_JWT_SECRET=dev-secret-change-me
docker compose up --build -d

# 等 health
until curl -sf http://127.0.0.1:8080/healthz; do sleep 1; done

# 一键：写 agent 配置、后台启 agent、下单、断言账本
START_AGENT=1 ./scripts/demo_paper.sh
```

成功时打印 `SUCCESS: instance N RUNNING → trade filled → portfolio updated`。

## 路径 B：本地二进制

```bash
export QS_JWT_SECRET=dev-secret-change-me
export QS_DB_PASSWORD=quantsaas

# 需要本机 Postgres + Redis（或 docker compose up postgres redis -d）
go run ./cmd/saas -config configs/config.yaml

# 另一终端
START_AGENT=1 ./scripts/demo_paper.sh
```

## 手动分步

```bash
./scripts/demo_paper.sh          # 不启 agent 时，到 trade 会 503
./scripts/run_agent_paper.sh     # 终端 B
# 再 POST trades 或 START_AGENT=1 重跑
```

## 验证命令

```bash
make ci
```
