# 项目目录架构（DDD + 多应用预留）

> 1. **按 DDD** 分层与限界上下文组织，而不是按技术件堆目录。  
> 2. **Monorepo**：Bot 运行时与后续 **Admin API / Admin Web** 分应用；共享领域上下文，禁止前端/后台直接依赖 TG Adapter。

---

## 1. 限界上下文（Bounded Context）

| 上下文 | 职责 | 谁用 |
|--------|------|------|
| **Catalog** | Case 协议、注册、上下架、菜单/分类查询 | Bot、Admin |
| **Conversation** | Dialog Session、填表锁、草稿 | 主要 Bot；（Admin 只读排查可选） |
| **Runtime** | Task、Orchestrator、Actuator、Comfy 执行 | Bot 运行时；（Admin 查任务/取消/重派） |
| **ChannelTG** | Telegram 入出适配 | **仅 Bot 应用** |
| **Platform** | Queue/Blob/Notify/Instance 端口与基础设施实现 | 各应用组装时注入 |
| **Identity**（预留） | 后台登录、角色权限 | Admin |
| **Billing**（预留） | 积分/订单 | 远期 |

共享内核（Shared Kernel）：稳定 ID、BlobRef、跨上下文事件 DTO（宜薄，避免大泥球）。

---

## 2. 每个上下文的 DDD 分层

每个 BC 内部统一四层（Go 包名）：

```text
<bc>/
  domain/           # 实体、值对象、领域服务、仓储接口、领域事件
  application/      # 应用服务 / 用例（命令·查询）；事务边界
  infrastructure/   # 仓储实现、外部客户端、队列/OSS 适配（实现 domain 端口）
  # 注意：渠道适配（TG）不放进 Catalog/Runtime 的 interfaces，
  # 而放在独立 ChannelTG 或 apps/bot 的 interfaces 层
```

依赖方向（严格）：

```text
interfaces/adapters → application → domain
infrastructure → domain（实现接口）
application → domain；application 通过接口调用 infrastructure（由 wire 注入）
domain 不依赖 application / infrastructure / 任何 UI/渠道 SDK
```

---

## 3. Monorepo 总树（含后台预留）

```text
comfyui_tgbot/
│
├── apps/                              # 可独立部署的应用
│   ├── bot/                           # TG Bot +（可同进程）Orchestrator/Actuator
│   │   └── cmd/comfyui-bot/
│   │       └── main.go                # wire：Conversation + Runtime + ChannelTG + Platform
│   │
│   └── admin-api/                     # 后续：后台 HTTP API（BFF/管理端）
│       └── cmd/admin-api/
│           └── main.go                # wire：Catalog + Runtime(查询/取消) + Identity…
│
├── web/                               # 前端应用（后续）
│   └── admin/                         # 管理后台 SPA（Vue/React 等，实现期再定）
│       ├── package.json
│       └── src/
│
├── internal/                          # 共享库（仅本 repo 应用引用；不对外发 module）
│   ├── sharedkernel/                  # 极薄：ID、BlobRef、通用错误、事件信封
│   │   ├── id.go
│   │   ├── blobref.go
│   │   └── event.go
│   │
│   ├── catalog/                       # BC: Case / Protocol
│   │   ├── domain/
│   │   │   ├── case.go
│   │   │   ├── protocol.go
│   │   │   ├── repository.go        # 接口
│   │   │   └── validator.go         # 领域校验端口（JSON Schema 实现可在 infra）
│   │   ├── application/
│   │   │   ├── commands.go          # RegisterCase, UpdateCase, DisableCase
│   │   │   └── queries.go           # GetMenu, ListCases, GetCase
│   │   └── infrastructure/
│   │       ├── persistence/         # GORM
│   │       └── validation/          # JSON Schema 引擎适配
│   │
│   ├── conversation/                  # BC: Dialog Session
│   │   ├── domain/
│   │   ├── application/
│   │   └── infrastructure/
│   │
│   ├── runtime/                       # BC: Task / Orchestrator / Actuator
│   │   ├── domain/
│   │   │   ├── task.go              # 聚合 + 状态迁移
│   │   │   ├── task_repository.go
│   │   │   └── execution_query.go   # 对账端口（由 actuator infra 实现）
│   │   ├── application/
│   │   │   ├── confirm_run.go       # 或放在 bot 应用服务，调用 catalog+conversation+runtime
│   │   │   ├── orchestrator/        # 调度、applyStatus、对账、风暴防护、notify
│   │   │   └── cancel_task.go
│   │   └── infrastructure/
│   │       ├── persistence/
│   │       ├── actuator/            # HandleDispatch、ledger、Comfy client
│   │       ├── comfyui/
│   │       └── storm/               # limiter/breaker 实现
│   │
│   ├── channel/tg/                    # BC: Telegram（interfaces 层为主）
│   │   ├── application/             # 可选：薄，转调各 BC application
│   │   └── infrastructure/          # go-telegram/bot、render、Update 处理、notify 消费
│   │       ├── bot.go
│   │       ├── render.go
│   │       └── notify_consumer.go
│   │
│   ├── platform/                      # 技术平台（非业务 BC，但集中端口）
│   │   ├── queue/
│   │   │   ├── port.go
│   │   │   └── memory/
│   │   ├── blob/
│   │   │   ├── port.go
│   │   │   └── localfs/
│   │   ├── notify/
│   │   │   └── port.go
│   │   └── instance/
│   │       ├── port.go
│   │       └── static/
│   │
│   ├── identity/                      # 预留 BC（Admin）
│   │   └── .gitkeep
│   │
│   └── packaging/                     # 可选：跨 BC 的 Bot 用例门面（App Service 聚合）
│       └── botapp/                    # ConfirmRun 跨 Catalog+Conversation+Runtime
│           └── facade.go
│
├── configs/
│   ├── bot.example.yaml
│   ├── admin-api.example.yaml         # 预留
│   └── cases/
│
├── data/                              # gitignore，本地 bot 运行时
├── docs/
│   ├── openspec/
│   └── superpowers/
├── scripts/
├── go.work                            # 可选：多 module；一期可用单 go.mod
├── go.mod
├── .gitignore
└── README.md
```

一期可先只实现 `apps/bot`；`apps/admin-api` 与 `web/admin` 建空目录或 README 占位即可，**不要**把管理逻辑写进 `channel/tg`。

---

## 4. 应用如何组装（DDD 组合根）

```text
apps/bot:
  ChannelTG.interfaces
    → packaging/botapp 或各 BC application
      → catalog / conversation / runtime domain
  wire: platform.queue.memory, blob.localfs
  同进程可选：runtime.application.orchestrator + runtime.infrastructure.actuator

apps/admin-api（后续）:
  HTTP interfaces（chi/gin handlers）
    → catalog.application（Case CRUD）
    → runtime.application（任务查询、取消、重派）
    → identity.application
  不引用 channel/tg
```

**ConfirmRun** 属于跨上下文应用服务，建议放在：

- `internal/packaging/botapp`（推荐，避免 Conversation 依赖 Runtime 或反向），或  
- `apps/bot` 内的 application 层  

不要放进 `domain`。

---

## 5. 与后台的边界（现在就要定）

| 能力 | Bot | Admin API | Admin Web |
|------|-----|-----------|-----------|
| Case 注册/编辑/上下架 | 只读用 Catalog 查询 | **写 + 读** | UI |
| Session 填表 | **主路径** | 只读/排障（可选） | 可选 |
| 跑任务 / 回 TG | **主路径** | 不发 TG | — |
| 任务列表/取消/重派 | 用户侧 ListMyTasks | **运营侧** | UI |
| 登录权限 | TG chat_id | Identity | 登录页 |

Admin **复用** `catalog` / `runtime` 的 application + domain，**新增** HTTP interfaces 与 identity；**禁止** import `channel/tg`。

---

## 6. 依赖与包规则（DDD）

1. `*.domain` 零依赖基础设施与渠道。  
2. 跨 BC 调用：优先 **应用层编排** 或 **领域事件/集成事件**（经 platform.queue），避免 domain 直接引用另一 BC 的 infrastructure。  
3. Catalog 被 Bot 与 Admin 共用 → API 保持稳定，不塞 TG 概念（无 `chat_id` 进 Case 聚合）。  
4. `web/admin` 只通过 **admin-api HTTP** 通信，不直连 DB、不引用 Go internal。  

---

## 7. 一期落地最小集（在完整树上裁剪实现）

必须实现目录：

- `apps/bot`
- `internal/{sharedkernel,catalog,conversation,runtime,channel/tg,platform,packaging/botapp}`

占位即可：

- `apps/admin-api/README.md`
- `web/admin/README.md`
- `internal/identity/.gitkeep`

---

## 8. 相对旧草案的变更

| 旧 | 新（DDD） |
|----|-----------|
| `internal/domain/case` 平铺 | `internal/catalog/{domain,application,infrastructure}` |
| `internal/app` 全局 | 分到各 BC application + `packaging/botapp` |
| `internal/adapter/tg` | `internal/channel/tg` |
| `cmd/comfyui-tgbot` | `apps/bot/cmd/...`；预留 `apps/admin-api` |
| 无前端位置 | `web/admin` |
| orchestrator/actuator 顶层包 | 收入 `runtime/application` 与 `runtime/infrastructure` |
