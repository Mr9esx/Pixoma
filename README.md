# Pixoma

Pixoma 让你随时随地使用自己的 ComfyUI 进行艺术创作。Telegram Bot + Case 目录 + 对话 Session + Task 运行时 + 多计算节点。

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
                     ComfyUI
```

默认路径**不需要 Redis**，也不再让你选 allinone / split。差别只有：Comfy 在不在这台机器上。

| 你怎么用 | 日常差别 | 文件存在哪 | 进程 |
|---|---|---|---|
| **本机** | 一台电脑起 `pixoma`；计算节点需在后台手动新增，再按部署命令跑 agent | 本地目录 `data/blob` | `pixoma` + 手动部署的 `pixoma-edge-agent` |
| **远程** | 控制面在一台机器，GPU 在另一台；Edge **主动连过来领活** | S3 或火山 TOS（禁止本机目录） | `pixoma` + 远端 `pixoma-edge-agent` |

完整架构：[`docs/architecture/`](docs/architecture/)。运行时：[`docs/architecture/runtime.md`](docs/architecture/runtime.md)。

## 新部署（推荐）

快速安装并启动控制面：

```bash
curl -fsSL https://pixoma.miaoplus.com/install.sh | sh
pixoma
```

也可以在源码目录直接启动控制面：

```bash
go run ./apps/pixoma/cmd/pixoma
# 或：make run
```

本地要改管理页面、走完整后台 UI 时用 `make dev`，见下方「本地调试」。只起后端时打开 8080 可能只看到提示页（前端还没打进二进制）。

日志会打出后台地址和**仅首次**的默认管理员账密。打开后台：先登录、改密，再走初始化向导（库、本机/远程、存储、节点、TG Token）。**保存后重启 `pixoma` 才按新配置装配。**

改密之后，启动日志不再打印明文密码。

计算节点不会自动创建：本机或远程都需要在后台「新增节点」后，按部署命令手动运行 `pixoma-edge-agent`。

### 远程 Edge

GPU 机器只需出站访问控制面，不要用本机目录当对象存储。在后台新建计算节点后复制部署命令（每台一把 `AGENT_TOKEN`）：

```bash
export CONTROL_PLANE_URL=http://控制面地址:8080
export AGENT_TOKEN=...          # 该节点在后台显示的 token
export EDGE_ID=...              # 该节点 ID（创建时自动生成）
export EDGE_SUBSCRIBE_TOPICS=   # 可选：逗号分隔的 Topic 列表；不设 = 不消费（需绑定任务队列才能接任务）
export BLOB_DRIVER=s3           # 或 tos，禁止 localfs
export COMFYUI_BASE_URL=http://ComfyUI地址:8188
go run ./apps/edge-agent/cmd/edge-agent
```

### BREAKING

相对旧版 YAML/`runtime_mode`/`queue.driver`/Redis Streams 派发：新安装只走 `pixoma` + 向导 + Edge 拉取。旧 Redis split **不保证原地升级**。独立 `admin-api` 与旧 `apps/bot` 入口已移除，管理 HTTP 由 `pixoma` 一体托管。

管理 API 初始化后需要管理员会话（登录 cookie / Bearer）；未初始化只放行登录与向导。

## 常用环境变量

| 变量 | 谁读 | 作用 |
|---|---|---|
| `DATA_DIR` | pixoma | 引导态与默认 SQLite / blob 目录（默认 `data`） |
| `HTTP_ADDR` | pixoma | 监听地址（默认 `127.0.0.1:8080`） |
| `HTTPS_PROXY` / `HTTP_PROXY` | pixoma | 紧急覆盖出站代理；设置页「网络」也可配 HTTP/SOCKS |
| `COMFYUI_BASE_URL` | pixoma / Edge | 真机 Comfy HTTP 根 |
| `EDGE_ID` | Edge | 领取身份，须与计算节点 id 一致 |
| `EDGE_SUBSCRIBE_TOPICS` | Edge | 订阅 Topic（逗号分隔）；未配置则不消费任何 Topic（节点需绑定任务队列才能接收任务），presence 首报写入，管理端可覆盖 |
| `CONTROL_PLANE_URL` / `PIXOMA_URL` | Edge | 控制面地址 |
| `AGENT_TOKEN` | Edge | 该节点自己的 Agent Token（后台可见） |
| `BLOB_DRIVER` | Edge / 紧急覆盖 | `localfs` / `s3` / `tos` |
| `PIXOMA_ENCRYPTION_KEY` | pixoma | 覆盖内置引导加密 key；请使用长随机值并通过密钥管理系统保存 |
| `METRICS_INTERVAL` | Edge | 系统指标采样/上报间隔（默认 `30s`，下限 `5s`） |
| `METRICS_RETENTION` | 控制面 | `edge_metrics` 保留窗口（默认 `24h`） |
| `STATS_TIMEZONE` | 控制面 | 任务统计归天时区（默认 `Asia/Shanghai`） |
| `TASK_STATS_RETENTION` | 控制面 | 任务统计保留时长（默认 `8760h`，即 365 天） |
| `DB_DRIVER` / `DATABASE_DSN` | backfill | 统计回填命令的数据库驱动与 DSN（默认 sqlite / `DATA_DIR/app.db`） |
| `S3_*` / `TOS_*` | 远程存储 | endpoint / region / bucket / keys |

对象存储密钥不要提交进 git。真网 TOS 门禁：`go test ./internal/platform/blob/tos/ -tags=live_tos -run TestRealTOS_PutGetRoundTrip`。

### 安全部署

- 默认 HTTP 监听地址是 `127.0.0.1:8080`。需要跨主机访问时，显式设置 `HTTP_ADDR`，并优先使用 TLS 反向代理；反向代理应转发真实协议/主机信息，以便 HTTPS 会话 Cookie 正确标记为 `Secure`。
- 管理端点基于会话和 RBAC：viewer 只读，operator/admin 可写，setup 管理端点仅 admin 可用。公网部署时请在反代层再加一层访问控制。
- 本地数据目录默认为 `DATA_DIR/data`；应用会尽量创建 `0700` 目录和 `0600` 文件，避免共享给不可信账号。
- 依赖审计、构建、测试和秘密扫描见 [`CONTRIBUTING.md`](CONTRIBUTING.md) 与 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)。安全披露流程见 [`SECURITY.md`](SECURITY.md)。

### 业务数据库

新部署在初始化向导的「数据库配置」步骤选择 SQLite / MySQL / Postgres 并填写连接；设置页只读展示业务库驱动与 DSN。MySQL 建议 8.0+。业务库连接只在 Setup 向导配置，换库/跨引擎数据迁移不在界面内支持：需要迁移时请走数据迁移后重跑初始化向导。

### 对象存储

初始化向导步骤为 密码 → 数据库 → 对象存储（不再单独选择部署位置，`localfs` 视为本机、`s3`/`tos` 视为远程）。对象存储步骤支持「连通性测试」：`localfs` 校验目录可写；S3/TOS 校验 endpoint/region/bucket/密钥（HeadBucket）。bucket 不存在时会提示是否代为创建，确认后自动创建（需账号具备建桶权限）。`localfs` 跨机使用需把同一目录挂载到所有机器（NFS / SMB）。

同机房多设备可选用「共享目录（SMB / NFS）」驱动 `sharedfs`：先在所有机器上挂载同一共享目录（`mount -t nfs` 或 `mount -t cifs`），再在向导「文件存储配置」里选择并填写挂载路径，连通性测试会校验目录可写；控制面与 Edge 挂载同一目录后按 key 互通读写。S3 端点也可填局域网 MinIO 等 S3 兼容服务地址。

## 跑通 TG 对话（本机）

向导里填好 Bot Token 并重启，或：

```bash
export TG_BOT_TOKEN=你的BotToken
make run
```

在 Telegram 里：

1. `/start` → 弹出主菜单
2. 点图片类入口 → 列出 Case
3. 点某个 Case → 预览 → 开始
4. 输入 prompt（可选字段可跳过）→ 确认生成
5. 完成后 Bot 发回 ComfyUI 生成结果

## 本地调试

一条命令同时起控制面和管理页面：

```bash
make dev
```

启动日志里有两个地址：

- **管理页面**（浏览器打开这个）：`http://127.0.0.1:5173`
- **后台接口**：`http://127.0.0.1:8080`

Ctrl-C 两个一起停。改 `web/admin` 保存后页面会自己刷新；改 Go 需要再跑一次 `make dev`。

只起后端、不看页面时继续用 `make run`。没把前端打进二进制时，打开 8080 会看到提示页，这是发布路径，不是日常调试入口。

发布：`make embed-admin` 之后再构建 `pixoma`，用户只开 8080 就是完整后台。

```bash
make build
make test
make clean            # 清掉 data/，下次启动重新走引导
curl -s localhost:8080/healthz
```

调试用 `make dev`（`pixoma` + Vite）。
