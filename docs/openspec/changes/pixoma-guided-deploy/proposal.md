## Why

当前部署依赖多进程、YAML/环境变量，以及 `allinone`/`split`、queue、blob 组合矩阵，上手成本高。需要一版「下载即跑、后台向导配完」的体验：默认无 Redis、无用户可见 queue，执行面统一为 Edge，派活改为数据库领取 + 长轮询。

**不拆 change 的原因（用户确认）**：本版一次性交付引导启动、配置落库、Edge 拉取与向导，避免半成品拓扑并存过久；接受单 change 体量大、验证周期长。

## What Changes

- 引入控制面一体入口（`pixoma`）：零配置可启动；日志输出后台地址与默认账号密码；未初始化仅开放登录与向导。
- 配置从「YAML 为主」转为「引导态 + 业务库 settings」；向导完成库、本机/远程、存储、节点/Comfy、渠道等步骤。
- **BREAKING（默认路径）**：取消用户必选的 `runtime_mode` / `queue.driver`；默认派发不再依赖 Redis Streams / 进程内 memory 跨进程总线。
- Edge 统一为 `pixoma-edge-agent`：主动连接控制面，长轮询/领取任务（claim + lease），回报状态；同机可用 localfs，远程用对象存储。
- 本机可自动拉起 Edge（推荐）或文档手起；远程仅提供安装说明与节点登记。
- 保留紧急覆盖（env）用于运维；旧 YAML 可过渡期兼容，但新部署以向导为准。

## Capabilities

### New Capabilities

- `platform-bootstrap`：零配置启动、引导态存储、默认管理员凭证与首启门闩、控制面一体二进制边界。
- `setup-wizard`：后台初始化向导步骤、校验矩阵（本机 vs 远程）、设置读写 API。
- `agent-pull-dispatch`：基于 DB 的任务领取/租约/心跳与 Edge↔控制面拉取协议（替代默认跨进程 queue）。

### Modified Capabilities

- `cross-process-queue`：默认部署不再要求 Redis/memory 作为跨进程总线；queue 配置项对用户隐藏或移除。
- `edge-agent`：改为拉取式领取与状态回传；二进制命名/最小配置对齐 `pixoma-edge-agent`。
- `shared-blob-store`：以「本机共用目录 / 远程 OSS」表达，不再绑定 allinone|split 矩阵文案。
- `task-orchestrator`：派发改为「任务可被 Edge claim」，不再默认 Publish 到 Redis topic。
- `admin-api-host` / `admin-web-shell`：未初始化门闩、向导路由与默认登录体验。

## Impact

- 进程：`apps/bot`、`apps/admin-api`、`apps/edge-agent` 与未来 `pixoma` / `pixoma-edge-agent` 入口；`web/admin` 向导页。
- 运行时：`queue` 适配器降级为可选/内部；新增 agent HTTP API；Task 状态机增加 lease 字段语义。
- 配置：`botconfig` / `adminconfig` 读取路径迁移；新增 settings 持久化。
- 文档：README 部署章节改为新故事；架构 `runtime` / `overview` 需同步。
- 依赖：默认路径可去掉 Redis 运行时依赖；对象存储仍为远程必需。
