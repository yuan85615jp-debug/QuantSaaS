# Lab / GA 作业 API

`app_role: lab|dev` 时开放进化作业端点。Agent / 交易链路与 Lab 物理隔离：Lab 只写 `gene_records` / `evolution_tasks`，不发 TradeCommand。

## 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/lab/tasks` | 创建作业 |
| GET | `/api/v1/lab/tasks` | 列表 |
| GET | `/api/v1/lab/tasks/{id}` | 详情 |
| POST | `/api/v1/lab/tasks/{id}/run` | 异步启动 GA |

### 创建

```bash
curl -X POST http://127.0.0.1:8080/api/v1/lab/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "strategy_id": "lunar",
    "symbol": "510300",
    "config": {
      "interval": "1d",
      "bars": 250,
      "pop_size": 20,
      "max_generations": 5,
      "test_mode": true,
      "promote": true
    }
  }'
```

### 启动

```bash
curl -X POST http://127.0.0.1:8080/api/v1/lab/tasks/1/run \
  -H "Authorization: Bearer $TOKEN"
```

轮询 `GET /api/v1/lab/tasks/1` 直到 `status=completed|failed`。成功时 `result` 含 `gene_id` / `score` / `param_pack`。

## 数据前提

GA 需要本地 `k_lines`。可先：

```bash
go run ./cmd/seed -real -symbol 510300 -interval 1d -bars 300
```

## 权限

- `saas` 角色：403（不允许进化写）
- `lab` / `dev`：允许
