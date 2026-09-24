# Phase 2 — 基础设施层

已实现：

| 路径 | 职责 |
|------|------|
| `internal/saas/config` | AppRole / Server / DB / Redis / JWT；密钥走环境变量 |
| `configs/config.yaml` | 无密钥模板 |
| `internal/saas/store/models.go` | User、Template、Instance、Portfolio、Runtime、Lot、Trade、Execution、Audit、Gene、EvolutionTask、KLine |
| `internal/saas/store/db.go` | NewDB + AutoMigrate |
| `internal/saas/store/redis.go` | Get/Set/Del；`champion:{strategyID}` |
| `internal/saas/auth` | SignToken / ParseToken |

环境变量：`QS_APP_ROLE`、`QS_DB_PASSWORD`、`QS_DB_HOST`、`QS_REDIS_PASSWORD`、`QS_REDIS_ADDR`、`QS_JWT_SECRET`、`QS_HTTP_ADDR`
