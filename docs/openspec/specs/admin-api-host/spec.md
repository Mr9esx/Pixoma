# admin-api-host Specification

## Purpose
定义独立 admin-api 进程作为无鉴权管理入口的宿主职责，以及与 bot 进程在管理 HTTP 上的边界。
## Requirements
### Requirement: 独立 admin-api 进程可启动并提供健康检查
系统 MUST 提供可独立启动的 admin-api 服务进程，并暴露健康检查端点以供运维探测。

#### Scenario: 健康检查成功
- **WHEN** 运维对 admin-api 发起健康检查请求
- **THEN** 服务返回成功响应，表明进程已就绪

### Requirement: 管理 HTTP 无鉴权（本期）
本期 admin-api 的管理 HTTP MUST 在无鉴权模式下可用，以便本地与可信内网联调；文档 MUST 警示不得对公网暴露。

#### Scenario: 无鉴权访问管理接口
- **WHEN** 客户端在未提供凭证的情况下调用 admin-api 管理接口
- **THEN** 请求不得因“缺少鉴权”被拒绝（仍可因业务校验失败被拒绝）

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
平台完成初始化后，admin 管理 HTTP MUST 要求有效管理员会话（或等价凭证）；未认证请求 MUST 被拒绝。未初始化阶段 MUST 仅放行登录与向导所必需的接口（见 `platform-bootstrap` / `setup-wizard`）。

#### Scenario: 初始化后无会话访问被拒
- **WHEN** 平台已初始化，客户端未提供管理员会话调用受保护管理 API
- **THEN** 请求因未认证被拒绝

### Requirement: 控制面可一体托管管理 API
`pixoma` 控制面入口 MUST 能托管原 admin-api 的管理 HTTP 能力（可同进程），使新部署不必再单独配置并启动第二个管理进程作为默认路径。过渡期 MAY 保留独立 `admin-api` 二进制。

#### Scenario: 一体入口提供健康检查与管理 API
- **WHEN** 用户仅启动 `pixoma` 且已初始化
- **THEN** 可通过该进程提供的地址访问健康检查与管理 API

