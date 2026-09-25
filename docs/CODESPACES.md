# GitHub Codespaces 運行指南（已驗證）

本文記錄在 **GitHub Codespaces** 上成功跑通 QuantSaaS 網頁的實際步驟與踩坑，方便下次复現。

> 驗證環境：GitHub Codespaces + 倉庫 `main` 分支  
> 成功標誌：瀏覽器出現「QuantSaaS / 登入・註冊」頁，且終端 `curl http://127.0.0.1:8080/healthz` 有 JSON 回應。

---

## 結論先看

在 Codespaces 上**推薦**：

1. **Postgres + Redis** 用 Docker Compose 起  
2. **SaaS 進程** 用本機 `go run`（連 `127.0.0.1`），**不要**依賴 `saas` 容器連 Docker 服務名 `postgres`  
3. 瀏覽器只開 **PORTS 面板轉發的 8080**，不要開 5432 / 6379

原因：實測 `saas` 容器經 Docker 內網連 `postgres:5432` 會 **connection timed out** 後 fatal 退出，導致宿主 `8080` Connection refused。Postgres/Redis 映射到宿主的 `127.0.0.1:5432` / `6379` 是正常的。

---

## 一、啟動資料庫

```bash
cd /workspaces/QuantSaaS   # 路徑以實際 Codespace 工作區為準

docker compose up -d postgres redis
until docker compose exec -T postgres pg_isready -U quantsaas; do sleep 1; done
docker compose ps
```

應只看到 `postgres`、`redis` 為 `Up (healthy)`。

若曾起過會失敗的 `saas` 容器，先停掉以免佔埠：

```bash
docker compose stop saas
```

---

## 二、本機運行 SaaS（推薦路徑）

```bash
export QS_JWT_SECRET=dev-secret-change-me
export QS_DB_HOST=127.0.0.1
export QS_DB_PASSWORD=quantsaas
export QS_REDIS_ADDR=127.0.0.1:6379
export QS_HTTP_ADDR=:8080

go run ./cmd/saas -config configs/config.yaml
```

**保持此終端不要關。** 另開終端自檢：

```bash
curl -s http://127.0.0.1:8080/healthz
```

有 JSON 回應 = HTTP 服務已就緒。

---

## 三、用瀏覽器打開網頁（PORTS 轉發）

Codespaces 裡服務聽在容器/虛擬機內部的 `127.0.0.1:8080`，本機瀏覽器**不能**直接當普通 `localhost` 用，必須走 GitHub 埠轉發。

### 步驟

1. 編輯器**最下方**分頁列點 **PORTS**（若無：`Ctrl+Shift+P` / `Cmd+Shift+P` → `Ports: Focus on Ports View`）
2. 確認有 **Port 8080**；沒有則點 **Forward a Port**，輸入 `8080`
3. 在 8080 那一列：
   - 點 **Forwarded Address** 旁的地球圖示 **Open in Browser**  
   - 或右鍵 → **Open in Browser**
4. 若打不開：將 8080 的 **Visibility** 改為 **Public** 再試

### 網址形態

```text
https://<codespace名稱>-8080.app.github.dev/
```

**請以 PORTS 面板顯示的網址為準**，不要手打或沿用舊名稱。

### 埠對照（勿用瀏覽器開 DB/Redis）

| 埠 | 服務 | 瀏覽器 |
|----|------|--------|
| **8080** | SaaS 網頁 + API | ✅ 只開這個 |
| 5432 | Postgres | ❌ |
| 6379 | Redis | ❌ |

開錯 6379 常見現象：`HTTP ERROR 502`。  
8080 未監聽或未轉發：`404` / `Connection refused`。

---

## 四、登入 / 註冊

頁面標題：**QuantSaaS · 登入 / 註冊**

### 自行註冊（推薦）

1. email：任意合法格式（如 `test@example.com`）
2. password：**至少 6 字元**
3. 先點 **註冊**，再 **登入**（若實作已自動登入則直接進主畫面）

### Demo 預設帳（僅在跑過 demo 腳本後存在）

| 欄位 | 值 |
|------|-----|
| email | `demo@quantsaas.local` |
| password | `demo1234` |

未跑過 `scripts/demo_paper.sh` 時請用自行註冊。

---

## 五、可選：Paper Demo 端到端

SaaS 的 `go run` 保持運行，**另開終端**：

```bash
cd /workspaces/QuantSaaS
export QS_JWT_SECRET=dev-secret-change-me
export SAAS_URL=http://127.0.0.1:8080
chmod +x scripts/*.sh
START_AGENT=1 ./scripts/demo_paper.sh
```

成功時終端會提示實例 RUNNING → fill → 帳本變化。詳見 [DEMO.md](DEMO.md)。

---

## 六、已知問題與排查

### A. `saas` 容器：連 Postgres timeout

日誌特徵：

```text
failed to connect to host=postgres ... dial tcp 172.18.0.x:5432: connection timed out
fatal: database
```

`docker compose ps` 只有 postgres/redis，或 saas 反覆退出；`curl :8080` 為 Connection refused。

**處理：** 使用本文「二、本機運行 SaaS」，不要依賴 `docker compose up saas`。

### B. 瀏覽器 502 且網址含 `-6379`

開成 Redis 埠。改開 **8080** 轉發位址。

### C. 瀏覽器 404

- 終端 `curl :8080/healthz` 失敗 → 先起 SaaS
- `curl` 成功但仍 404 → PORTS 未轉發或網址手打錯誤 → 用面板 **Open in Browser**

### D. `go run` 佔用埠

```bash
docker compose stop saas   # 釋放被容器佔用的 8080
```

---

## 七、與本機 Docker 全量路徑的差異

| 環境 | 建議 |
|------|------|
| 本機 Docker Desktop / 一般 Linux | `docker compose up --build -d` 後跑 `demo_paper.sh`（見 [DEMO.md](DEMO.md)） |
| **GitHub Codespaces** | Postgres/Redis 用 Compose；**SaaS 用 `go run` + `127.0.0.1`**；瀏覽器走 PORTS 8080 |

---

## 八、快速檢查清單

- [ ] `docker compose ps`：postgres、redis healthy
- [ ] `go run ./cmd/saas ...` 終端仍在跑、無 fatal
- [ ] `curl -s http://127.0.0.1:8080/healthz` 有 JSON
- [ ] PORTS 有 8080，Visibility 可先 Public
- [ ] 瀏覽器網址為 `*-8080.app.github.dev`，出現登入頁
- [ ] 註冊後可進入實例/帳本畫面
