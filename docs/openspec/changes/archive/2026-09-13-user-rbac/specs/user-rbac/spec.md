## ADDED Requirements

### Requirement: 预置角色
系统 MUST 预置三档系统账号角色：`Admin`（全量权限）、`Operator`（业务运维但不含账号管理、角色管理与删除类操作）、`Viewer`（只读）。角色能力以权限点集合表示，作为后续细粒度自定义角色的基础。

#### Scenario: 各角色默认能力
- **WHEN** 系统创建账号并为该账号分配 `Admin`/`Operator`/`Viewer` 角色
- **THEN** 该账号获得对应权限点集合：Admin 全量、Operator（除账号管理/角色管理/删除之外的全部运维）、Viewer 只读

### Requirement: 基于角色的授权门禁
受限管理接口 MUST 按当前登录系统账号的权限点做授权校验；未持有所需权限的请求 MUST 返回 403，且不执行任何业务副作用。账号管理、角色管理、注册开关与用户删除类接口 MUST 仅允许 `Admin`（或等价权限点）。

#### Scenario: 越权访问被拒
- **WHEN** 角色为 `Viewer` 或 `Operator` 且不具备某接口所需权限的账号调用该受限管理接口
- **THEN** 请求返回 403 且不产生副作用

#### Scenario: 管理员具备全部权限
- **WHEN** 角色为 `Admin` 的账号调用任意管理接口
- **THEN** 请求通过授权校验并正常执行

#### Scenario: 未登录访问受限接口
- **WHEN** 客户端未携带有效系统账号会话调用受限管理接口
- **THEN** 请求返回 401 且不执行任何操作
