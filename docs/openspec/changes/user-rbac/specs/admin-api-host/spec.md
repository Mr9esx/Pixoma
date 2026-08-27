## MODIFIED Requirements

### Requirement: 初始化后管理接口需管理员会话
平台完成初始化后，admin 管理 HTTP MUST 要求有效管理员会话（或等价凭证）；未认证请求 MUST 被拒绝。未初始化阶段 MUST 仅放行登录与向导所必需的接口（见 `platform-bootstrap` / `setup-wizard`）。受限管理接口 MUST 进一步按当前登录系统账号的角色/权限进行授权校验，未持有所需权限的请求 MUST 返回 403 且不执行业务副作用；账号管理、角色管理与删除类接口 MUST 仅允许 `Admin`。

#### Scenario: 初始化后无会话访问被拒
- **WHEN** 平台已初始化，客户端未提供管理员会话调用受保护管理 API
- **THEN** 请求因未认证被拒绝

#### Scenario: 无权限访问被拒
- **WHEN** 客户端持有有效会话但角色无某受限接口所需权限（如 `Viewer` 调用用户删除接口）
- **THEN** 请求返回 403 且不执行任何业务副作用

#### Scenario: 管理员访问全量
- **WHEN** `Admin` 角色调用任意管理接口
- **THEN** 请求通过授权校验并正常执行
