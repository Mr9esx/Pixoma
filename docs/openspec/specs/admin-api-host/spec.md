# admin-api-host Specification

## Purpose
定义独立 admin-api 进程作为无鉴权管理入口的宿主职责，以及与 bot 进程在管理 HTTP 上的边界。
## Requirements
### Requirement: CORS 允许管理前端联调
admin-api MUST 支持 CORS，使 `web/admin` 开发源可在浏览器中调用其管理 API。

#### Scenario: 浏览器预检或跨域请求
- **WHEN** 管理前端所在源向 admin-api 发起跨域请求（含预检）
- **THEN** 服务返回允许该联调场景的 CORS 响应头

### Requirement: bot 不再承载管理 CRUD/观测路由
bot 进程 MUST NOT 再挂载已迁移的管理 CRUD/观测 HTTP 路由；该类能力仅由 admin-api 提供。

#### Scenario: bot 上管理路径不可用
- **WHEN** 客户端对 bot 的原管理路径（如 `/api/v1/comfy-instances`）发起请求
- **THEN** 该路径不再提供原管理 API 行为（例如 404 或不存在对应路由）

### Requirement: 初始化后管理接口需管理员会话
平台完成初始化后，admin 管理 HTTP MUST 要求有效管理员会话（或等价凭证）；未认证请求 MUST 被拒绝。未初始化阶段 MUST 仅放行登录与向导所必需的接口（见 `platform-bootstrap` / `setup-wizard`）。受限管理接口 MUST 进一步按当前登录系统账号的角色/权限进行授权校验，未持有所需权限的请求 MUST 返回 403 且不执行任何业务副作用；账号管理、角色管理与删除类接口 MUST 仅允许 `Admin`。

#### Scenario: 初始化后无会话访问被拒
- **WHEN** 平台已初始化，客户端未提供管理员会话调用受保护管理 API
- **THEN** 请求因未认证被拒绝

#### Scenario: 无权限访问被拒
- **WHEN** 客户端持有有效会话但角色无某受限接口所需权限（如 `Viewer` 调用用户删除接口）
- **THEN** 请求返回 403 且不执行任何业务副作用

#### Scenario: 管理员访问全量
- **WHEN** `Admin` 角色调用任意管理接口
- **THEN** 请求通过授权校验并正常执行

### Requirement: 控制面可一体托管管理 API
`pixoma` 控制面入口 MUST 托管管理 HTTP 能力（健康检查、向导、管理 API、Agent API 同进程）；独立 `admin-api` 二进制已移除，新部署 MUST NOT 依赖或启动独立管理进程。

#### Scenario: 一体入口提供健康检查与管理 API
- **WHEN** 用户仅启动 `pixoma` 且已初始化
- **THEN** 可通过该进程提供的地址访问健康检查与管理 API

#### Scenario: 无独立管理进程可启动
- **WHEN** 运维查找 `apps/admin-api` 或 `bin/admin-api`
- **THEN** 该入口不存在，文档与 Makefile 不再提供独立 admin-api 构建或运行目标

