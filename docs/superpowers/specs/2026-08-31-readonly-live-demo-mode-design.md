---
comet_change: readonly-live-demo-mode
role: technical-design
canonical_spec: openspec
---

# Readonly Live Demo Mode Design

## Context

Pixoma 常规启动会创建或打开 bootstrap 数据库、加载业务配置、迁移业务数据库、初始化 Blob、启动 Bot 与 Edge 运行时。公开 Live Demo 必须避免这些真实副作用，同时保留后台页面的查询体验。

## Goals / Non-Goals

**Goals**

- `-livedemo` 启动隔离的只读 Demo 服务。
- 不创建、打开或持久化 bootstrap、业务数据库和 Blob 存储。
- 使用进程内 ephemeral SQLite 复用现有模型、GORM 仓储和查询行为。
- 注入覆盖全后台的确定性演示数据。
- 固定 `admin / 123456` 登录，并在登录页提醒。
- 服务端强制只读，隐藏设置入口，拒绝配置和凭据变更。

**Non-Goals**

- 不持久化 Demo 变更。
- 不连接 Telegram、真实 ComfyUI、Edge Agent、MySQL、PostgreSQL 或外部对象存储。
- 不改变常规模式的数据模型、启动流程或权限体系。

## Architecture

`main()` 先用标准 `flag` 包解析 `-livedemo`。该参数存在时进入独立的 `runLiveDemo` 分支，不执行 `bootstrap.Open` 和常规 `run`。

```text
CLI -livedemo
  -> runLiveDemo
  -> open ephemeral SQLite
  -> AutoMigrate + seed demo data
  -> assemble readonly demo router
  -> serve HTTP
```

Demo 数据层使用 `gorm.io/driver/glebarez/sqlite` 打开类似 `file:pixoma_demo?mode=memory&cache=shared` 的 DSN，并限制连接池复用同一内存库。进程退出后数据消失。业务表复用常规模式的 GORM 模型；Cases、Channels、Topics、Edges、Tasks、Sessions、Users、Stats 和 Menu Cards 复用现有仓储与 handler。

Demo 模式不装配 bootstrap 依赖的 setup handler。它独立实现最小认证面：

- `GET /api/v1/setup/status`：返回 initialized、authenticated、`live_demo: true`。
- `POST /api/v1/setup/login`：校验固定账号并签发内存会话。
- `POST /api/v1/setup/logout`：删除内存会话。
- `GET /api/v1/setup/me`：返回演示管理员和 `live_demo: true`。

Demo Auth Gate 使用内存会话表校验请求，并向 handler context 注入演示管理员。它不写 console user、bootstrap 管理员或密码哈希。

## Data Flow

启动时按确定性 ID 和时间戳创建演示数据，使节点、案例、任务、会话、用户、频道、话题和统计之间存在可解释关系。Dashboard 和详情页可以跨资源查询，但不产生运行副作用。

```text
Browser
  -> Demo Router
  -> Demo Auth Gate
  -> Readonly Middleware
  -> Existing Admin Handler
  -> GORM Repository
  -> Ephemeral SQLite
```

## Readonly Enforcement

Readonly Middleware 位于认证后、业务 handler 前：

- `GET`、`HEAD`、`OPTIONS` 默认可查询。
- `POST`、`PUT`、`PATCH`、`DELETE` 默认拒绝。
- 仅 `/api/v1/setup/login` 和 `/api/v1/setup/logout` 允许 `POST`。
- 设置、配置、注册、修改密码、重启、重试、取消和 Token 轮换路径显式拒绝。
- 所有拒绝响应统一返回 `403`、错误文案和 `readonly_live_demo` 错误码。

前端通过 `status.live_demo` 或 `me.live_demo` 识别 Demo 模式。登录页展示账号密码提醒；侧边栏过滤设置项；变更控件尽量隐藏。前端状态不是安全边界，服务端拦截是唯一强制边界。

## Error Handling

- 登录失败返回 `401` 和中文错误文案。
- 未认证访问受保护 API 返回 `401`。
- Demo 写操作返回 `403 readonly_live_demo`。
- Demo 运行期数据库错误返回 `500`，但不落盘诊断数据。

## Testing Strategy

Go 单元与集成测试覆盖：

- `-livedemo` 分支不调用 bootstrap 且不创建持久化文件。
- ephemeral SQLite 可迁移、注入和查询演示数据。
- 固定凭据登录成功，错误凭据失败。
- 未认证请求返回 `401`，认证后变更请求返回 `403 readonly_live_demo`。
- 登录和登出可放行；设置、注册、改密和运行副作用接口被拒绝。
- 主要资源 `GET` 返回演示数据。

前端测试覆盖：

- `SetupStatus` 与 `CurrentUser` 透传 `live_demo`。
- 登录页渲染固定凭据提醒。
- Demo 模式下侧边栏不渲染设置入口。

## Risks / Trade-offs

- [写路径遗漏] → 中间件默认拒绝非安全方法，维护极小放行清单。
- [内存 SQLite 连接断开导致数据消失] → 限制连接池并保持单个进程级连接。
- [复用 handler 意外触发副作用] → Demo 中间件先于 handler 执行，并补充副作用路径测试。
- [演示数据不够真实] → 使用中文示例、确定性关系和与统计页一致的聚合数据。
