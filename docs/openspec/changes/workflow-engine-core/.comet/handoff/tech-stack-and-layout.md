# Tech Stack & Layout（讨论稿）

## 已定

- Language: Go
- ORM: GORM
- DB: SQLite → 可切 MySQL
- Phase1: 单进程可跑通；OSS=本地 FS；MQ/调度/Actuator 单实例闭环
- TG: `github.com/go-telegram/bot`
- 协议校验: JSON Schema + 媒体/OSS 扩展
- Queue: **Port/Adapter**；Phase1 **Memory**；二期换 MQ 只加适配器

## 推荐选型

| 领域 | 推荐 | 备选 | 理由 |
|------|------|------|------|
| 模块路径 | `github.com/<org>/comfyui_tgbot`（按你仓库定） | — | 标准 Go module |
| 配置 | `caarlos0/env` + YAML 可选 | koanf / viper | 简单、12-factor 友好 |
| 日志 | 标准库 `log/slog` | zap | 无额外依赖，结构化够用 |
| HTTP | `go-chi/chi` | stdlib / gin | 轻、中间件清晰；Webhook + 健康检查 |
| TG Bot | **`github.com/go-telegram/bot`**（已定） | gotgbot | 零依赖、贴近官方 API；适合薄 Adapter |
| DB 驱动 | GORM + `gorm.io/driver/sqlite`（modernc 纯 Go 优先） | mattn/cgo sqlite | 部署省 cgo |
| MySQL 预留 | `gorm.io/driver/mysql` | — | DSN 切换 |
| 协议校验 | **JSON Schema 引擎** + 少量媒体/OSS 引用扩展 | 纯自研引擎 | 动态表单用标准 Schema |
| HTTP DTO 校验 | 可选 `go-playground/validator` | — | 仅固定入参 |
| Blob/OSS | 自研 `BlobStore` 接口 + LocalFS；二期 MinIO/S3 | 一期直接绑 S3 | 与 Queue 同理：端口隔离 |
| MQ | **`Queue` 端口** + Phase1 **Memory** 适配器 | 业务直连 MQ SDK | 切换只换适配器 |
| ComfyUI | 自研 HTTP client（net/http） | 第三方封装 | API 面窄 |
| 测试 | testify + httptest + sqlite :memory: | — | — |
| 迁移 | GORM AutoMigrate（一期） | golang-migrate | 上 MySQL 前可再引入 |

## 一期进程形态（已修正）

**角色三分**：Bot · **Scheduler（通常单点）** · **Actuator（跟 ComfyUI 实例走）**

开发默认 **一个进程 all-in-one**：三个角色同进程 goroutine；单 ComfyUI ⇒ 一个 Scheduler + 一个 Actuator。

二期拆开时：

```text
bot        — 可多副本
scheduler  — 1（或主备），负责路由与投递
actuator   — 每 ComfyUI 实例一份（sidecar / 同机）
```

不要把「可水平扩的大 worker」做成 Scheduler+全体 Actuator 绑死；扩的是 Actuator，不是 Scheduler。

详见 `process-roles.md`。

## Queue 抽象（已定）

目标：一期 Memory，二期换 MQ **只动 `internal/queue` 适配器 + 配置**，不改 App / Session / Task 状态机语义。

```text
domain/app/scheduler/actuator
        │  只依赖端口
        ▼
internal/queue          ← Port：Publish / Subscribe / Ack
        │
   ┌────┼────┐
   ▼    ▼    ▼
Memory  NATS  Rabbit…   ← Adapter（一期只实现 Memory）
```

端口约定：

- 消息体用自有 DTO/JSON（`task_id`、`topic`、payload 引用），不暴露 channel/amqp 类型
- `Publish(ctx, topic, msg)`、`Subscribe(ctx, topic, handler)`；成功 Ack，失败可重试
- 消费者按 `task_id` + 状态机 **幂等**
- Topic 为业务名（如 `comfy.dispatch` / `comfy.status`）；与 MQ exchange 映射在适配器配置
- Scheduler/Actuator **禁止** import 具体 MQ SDK

Phase1 Memory：进程内 channel + worker goroutine。

目录上建议：

```text
internal/queue/
  port.go           # interface
  message.go        # 业务消息 DTO
  memory/           # Phase1
  # nats/ rabbit/   # Phase2 再加
```

BlobStore 同样用端口，避免以后换 OSS 大面积改动。

## 目录草案

```text
comfyui_tgbot/
├── cmd/
│   └── comfyui-tgbot/          # main：bot | worker | all
├── internal/
│   ├── config/
│   ├── db/
│   ├── domain/
│   │   ├── protocol/
│   │   ├── case/
│   │   ├── session/
│   │   └── task/
│   ├── app/
│   ├── adapter/tg/
│   ├── comfyui/
│   ├── blob/                   # BlobStore + localfs
│   ├── queue/                  # port + memory
│   ├── scheduler/
│   ├── actuator/
│   └── httpapi/
├── configs/
├── data/
├── docs/
├── go.mod
└── README.md
```

## 依赖方向（不允许反依赖）

```text
adapter/tg → app → domain
scheduler / actuator → domain/task + comfyui + blob + queue.port
app → blob
domain 不 import adapter、comfyui、tg、具体 MQ SDK
```

## 待你拍板的点

1. TG：已定 `go-telegram/bot`
2. Queue：已定 Memory + Port/Adapter
3. 进程：开发 **all-in-one = Bot+Scheduler+Actuator 同进程**；拆分时按三角色，**Scheduler 单点、Actuator 随 ComfyUI 实例**（已按你的纠正修正）
