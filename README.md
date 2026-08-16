# Pixoma

Go monorepo：Telegram Bot + Case 目录 + 对话 Session + Task 运行时 + 多 ComfyUI 实例。

## 架构简述

```text
User (TG) ──► Session ──► Task
                            │
                     pixoma 控制面
                     （调度 / 管理后台 / Agent API）
                            │
                     DB 可领取 + 长轮询 claim
                            │
                     pixoma-edge-agent
                            │
                     ComfyUI / Mock
```

默认路径**不需要 Redis**，也不再让你选 allinone / split。差别只有：Comfy 在不在这台机器上。

| 你怎么用 | 日常差别 | 文件存在哪 | 进程 |
|---|---|---|---|
| **本机** | 一台电脑起 `pixoma`，它会自动拉起本机 Edge | 本地目录 `data/blob` | `pixoma` + 自动 spawn 的 `pixoma-edge-agent` |
| **远程** | 控制面在一台机器，GPU 在另一台；Edge **主动连过来领活** | S3 或火山 TOS（禁止本机目录） | `pixoma` + 远端 `pixoma-edge-agent` |

完整架构：[`docs/architecture/`](docs/architecture/)。运行时：[`docs/architecture/runtime.md`](docs/architecture/runtime.md)。

## 新部署（推荐）

空目录启动控制面：

```bash
go run ./apps/pixoma/cmd/pixoma
# 或：make run-mock
```

本地要改管理页面、走完整后台 UI 时用 `make dev`，见下方「本地调试」。只起后端时打开 8080 可能只看到提示页（前端还没打进二进制）。

日志会打出后台地址和**仅首次**的默认管理员账密。打开后台：先登录、改密，再走初始化向导（库、本机/远程、存储、节点、TG Token）。**保存后重启 `pixoma` 才按新配置装配。**

改密之后，启动日志不再打印明文密码。

本机路径会自动拉起 Edge（`EDGE_AUTO_SPAWN=0` 可关）。没有真 Comfy 时保持 `COMFY_MOCK=1`（默认），向导里也可勾 Mock。

### 远程 Edge

GPU 机器只需出站访问控制面，不要用本机目录当对象存储：

```bash
export CONTROL_PLANE_URL=http://控制面地址:8080
export AGENT_TOKEN=...          # 控制面 data/agent.token
export INSTANCE_ID=gpu-1
export BLOB_DRIVER=s3           # 或 tos，禁止 localfs
export COMFY_MOCK=1             # 真机改 0
go run ./apps/edge-agent/cmd/edge-agent
```

### BREAKING

相对旧版 YAML/`runtime_mode`/`queue.driver`/Redis Streams 派发：新安装只走 `pixoma` + 向导 + Edge 拉取。旧 Redis split **不保证原地升级**。独立 `admin-api` 仅过渡期保留，不再是新部署默认路径。

管理 API 初始化后需要管理员会话（登录 cookie / Bearer）；未初始化只放行登录与向导。

## 常用环境变量

| 变量 | 谁读 | 作用 |
|---|---|---|
| `DATA_DIR` | pixoma | 引导态与默认 SQLite / blob 目录（默认 `data`） |
| `HTTP_ADDR` | pixoma | 监听地址（默认 `127.0.0.1:8080`） |
| `TG_BOT_TOKEN` / `TELEGRAM_BOT_TOKEN` | 紧急覆盖 | Telegram Token（向导也会落库） |
| `COMFY_MOCK` | pixoma / Edge | `1`/`true` = Mock；`0`/`false` = 真 Comfy |
| `COMFYUI_BASE_URL` | pixoma / Edge | 真机 Comfy HTTP 根 |
| `INSTANCE_ID` | Edge | 领取身份，须与实例池 id 一致 |
| `CONTROL_PLANE_URL` / `PIXOMA_URL` | Edge | 控制面地址 |
| `AGENT_TOKEN` | Edge | 共享 Agent Token |
| `BLOB_DRIVER` | Edge / 紧急覆盖 | `localfs` / `s3` / `tos` |
| `EDGE_AUTO_SPAWN` | pixoma | 本机是否自动拉起 Edge（默认开） |
| `EDGE_AGENT_BIN` | pixoma | Edge 二进制路径 |
| `S3_*` / `TOS_*` | 远程存储 | endpoint / region / bucket / keys |

对象存储密钥不要提交进 git。真网 TOS 门禁：`go test ./internal/platform/blob/tos/ -tags=live_tos -run TestRealTOS_PutGetRoundTrip`。

## 跑通 TG 对话（默认 mock / 本机）

向导里填好 Bot Token 并重启，或：

```bash
export TG_BOT_TOKEN=你的BotToken
make run-mock
```

在 Telegram 里：

1. `/start` → 弹出主菜单
2. 点图片类入口 → 列出 Case
3. 点某个 Case → 预览 → 开始
4. 输入 prompt（可选字段可跳过）→ 确认生成
5. 完成后 Bot 发回图片（mock 为示例 PNG）

种子 Case：`configs/cases/*.json`。

## Mock 开关

```bash
make run-mock          # COMFY_MOCK=1
make run               # COMFY_MOCK=0（需可达 ComfyUI）
```

向导、环境变量、`comfy_mock` 都会进同一条执行链路；Mock 与真机必须一起能跑通。

## 本地调试

一条命令同时起控制面和管理页面（真 Comfy，和 `make run` 一样）：

```bash
make dev
```

启动日志里有两个地址：

- **管理页面**（浏览器打开这个）：`http://127.0.0.1:5173`
- **后台接口**：`http://127.0.0.1:8080`

Ctrl-C 两个一起停。改 `web/admin` 保存后页面会自己刷新；改 Go 需要再跑一次 `make dev`。

只起后端、不看页面时继续用 `make run` / `make run-mock`。没把前端打进二进制时，打开 8080 会看到提示页，这是发布路径，不是日常调试入口。

发布：`make embed-admin` 之后再构建 `pixoma`，用户只开 8080 就是完整后台。

```bash
make build
make test
curl -s localhost:8080/healthz
```

独立 `admin-api` 仅过渡期保留，调试请用 `make dev`。
