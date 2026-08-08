# ComfyUI Telegram Bot

Go monorepo：DDD 限界上下文（Catalog / Conversation / Runtime / ChannelTG / Platform）。

## 架构简述

```text
User (TG From upsert) ──► Session (chat 采集态) ──► Task (生成任务)
                                                      │
                                              Orchestrator
                                              (round-robin)
                                                      │
                                              实例池 Pool
                                   ┌──────────────────┼──────────────────┐
                                   ▼                  ▼                  ▼
                              ComfyUI A          ComfyUI B          … / Mock
```

- **User**：按 `tg_user_id` upsert，供 Session 关联。
- **Session**：绑定 `user_id` + `chat_id`，长期保留；ConfirmRun 创建 Task 时写 `session_id`。
- **Task**：落库；派发后带 `instance_id`；通知经 Session 取 `chat_id`。
- **实例池**：`comfy_instances` CRUD + 健康探测；仅健康实例参与 round-robin；无可用实例不投递。

事件链：ConfirmRun → `task.created` → Orchestrator → `dispatch.*` → Actuator → `task.status` → 发图。

完整架构文档（系统总览 / 限界上下文 / 运行时 / 数据模型与 ER）：见 [`docs/architecture/`](docs/architecture/)。
数据表与 ER 专章：[`docs/architecture/data-model.md`](docs/architecture/data-model.md)。
## 跑通 TG 对话（默认 mock）

```bash
export TG_BOT_TOKEN=你的BotToken
make run-mock
# 或：go run ./apps/bot/cmd/comfyui-bot
```

配置：`configs/bot.yaml`（可从 `configs/bot.example.yaml` 复制）。

在 Telegram 里：

1. `/start` → 弹出主菜单（图片 / 视频 / 充值…，与产品图一致）
2. 点 **🖼 图片** → 列出 Case（二次元 / 写真 / Logo）
3. 点某个 Case → **预览** → **开始 Case**
4. 输入 prompt（可选字段可「跳过」）→ **确认生成**
5. 完成后 Bot 发回图片（mock 模式为示例 PNG）

种子 Case：`configs/cases/*.json`（启动自动导入/更新）。

## 启动种子（Comfy 实例）

启动时会把配置 upsert 进 `comfy_instances`：

| 配置 | 作用 |
|---|---|
| `comfyui_base_url` + `default_instance_id` | 单实例种子（缺省 id=`local`） |
| `comfy_instances` | 多实例种子列表（优先于仅用 base_url 的单实例 upsert） |
| `health_probe_interval` | 健康探测周期（如 `30s`） |

`comfy_instances` 示例见 `configs/bot.example.yaml`。

## Mock 开关

```yaml
comfy_mock: true   # in-process Mock；system/queue 带 mock: true
comfy_mock: false  # 真实 HTTP，走 comfyui_base_url / 各实例 base_url
```

或环境变量 / Make：

```bash
make run-mock          # COMFY_MOCK=1
make run               # COMFY_MOCK=0（需可达 ComfyUI）
# 等价：COMFY_MOCK=0|1|true|false
```

## 实例管理与观测（HTTP，admin-api）

> **无鉴权警示**：`/api/v1/comfy-instances*` **当前无鉴权**，仅可在本机或可信内网暴露；勿对公网开放。
>
> 实例管理已从 bot（`:8080`）迁到 **admin-api（默认 `127.0.0.1:8081`）**，两边共用同一 `database_dsn` / `data/app.db`。

```bash
# 启动管理面（与 bot 同库）
make run-admin-api
# 或：go run ./apps/admin-api/cmd/admin-api

# 列表
curl -s localhost:8081/api/v1/comfy-instances

# 创建 / 更新
curl -s -X POST localhost:8081/api/v1/comfy-instances \
  -H 'Content-Type: application/json' \
  -d '{"id":"gpu-1","base_url":"http://127.0.0.1:8188","enabled":true}'

# Comfy system / queue（Mock 时返回 mock: true）
curl -s localhost:8081/api/v1/comfy-instances/gpu-1/system
curl -s localhost:8081/api/v1/comfy-instances/gpu-1/queue

# 本系统已派发到该实例的 Task（pending 且无 instance_id 的不出现）
curl -s 'localhost:8081/api/v1/comfy-instances/gpu-1/tasks?limit=20'
```

其它：

```bash
make build
make test
curl localhost:8080/healthz   # bot 探活
curl localhost:8081/healthz   # admin-api 探活
```

User / Session / Task / 实例均落 SQLite；**重启进程后数据仍在**（默认 DSN 见 `configs/bot.example.yaml` / `configs/admin-api.example.yaml`）。
