## ADDED Requirements

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
