## REMOVED Requirements

### Requirement: 独立 admin-api 进程可启动并提供健康检查
**Reason**: 过渡期结束，独立 `admin-api` 二进制与 `apps/admin-api` 入口已删除；健康检查与管理 API 由 `pixoma` 一体托管。
**Migration**: 使用 `pixoma` 的 `/healthz` 与管理 API；`make run-admin-api` / `bin/admin-api` 不再存在。

### Requirement: 管理 HTTP 无鉴权（本期）
**Reason**: 独立 admin-api 删除后管理 HTTP 统一走 `pixoma` 门禁，初始化后要求管理员会话，不再存在无鉴权管理入口。
**Migration**: 使用已初始化平台的登录与管理员会话访问管理 API。

## MODIFIED Requirements

### Requirement: 控制面可一体托管管理 API
`pixoma` 控制面入口 MUST 托管管理 HTTP 能力（健康检查、向导、管理 API、Agent API 同进程）；独立 `admin-api` 二进制已移除，新部署 MUST NOT 依赖或启动独立管理进程。

#### Scenario: 一体入口提供健康检查与管理 API
- **WHEN** 用户仅启动 `pixoma` 且已初始化
- **THEN** 可通过该进程提供的地址访问健康检查与管理 API

#### Scenario: 无独立管理进程可启动
- **WHEN** 运维查找 `apps/admin-api` 或 `bin/admin-api`
- **THEN** 该入口不存在，文档与 Makefile 不再提供独立 admin-api 构建或运行目标
