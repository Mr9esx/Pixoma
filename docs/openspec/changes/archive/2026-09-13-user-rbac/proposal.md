## Why

Pixoma 后台目前只有一个 bootstrap 单管理员账号（默认 `admin`），登录后所有管理接口一律放行，没有角色与权限区分，也不能创建/管理其他系统账号；首次初始化向导改密后直接就进入数据库配置，缺少管理员资料的采集。随着多运维协作与内网共享部署，需要一个面向「登录后台的控制台账号」的多用户体系与 RBAC，并允许按需开放自助注册。

## What Changes

- 新增控制台系统账号模型：账号名、邮箱、昵称、头像、启用状态、角色；作为管理后台登录与鉴权的主体，与已有的消息平台终端用户目录（`users` 资源页）明确区分。
- 新增 RBAC：`Admin` / `Operator` / `Viewer` 三档角色，按角色映射权限点，统一授权门禁对受限管理接口按当前登录账号的权限做 403 校验。
- 首次初始化向导在「更新密码」之后新增一个「如何称呼您」步骤，采集初始管理员的昵称、邮箱、头像，并写入该 admin 系统账号。
- 系统设置新增「用户管理」：管理员可新增账号（账号名、邮箱、昵称、密码）、禁用/启用、删除、重置密码。
- 系统设置新增「是否开放注册」开关（默认关闭）；开启后允许自助注册控制台账号。
- 把 bootstrap 初始 `admin` 迁移进新的系统账号模型，保留凭据并补齐资料。
- 将消息平台终端用户内部表由 `users` 更名为 `channel_users`（`user_external_identities` → `channel_user_external_identities`），通过 `db.RenameLegacy` 幂等迁移保留既有数据。
- 不改变：消息平台终端用户目录页与 API 行为、各业务功能（Case/Channel/Edge 等）自身行为；仅本 change 不含用户自助改密与个人资料自助管理（改密仅由管理员重置）；对消息平台终端用户仅做内部表更名（API/展示不变），对管理接口施加 RBAC 授权。

## Capabilities

### New Capabilities
- `console-user-accounts`: 控制台登录账号模型（账号/邮箱/昵称/头像/启用状态/角色/认证），区别于消息平台终端用户；bootstrap 初始 admin 迁移为系统账号。
- `user-rbac`: `Admin`/`Operator`/`Viewer` 角色与基于角色的授权门禁，按登录账号权限拒绝未授权管理请求（403）。
- `admin-user-management`: 系统设置「用户管理」实现与 API，提供新增、禁用/启用、删除、重置密码。
- `self-registration`: 系统设置「是否开放注册」开关与自助注册流程（注册即创建默认角色 `Viewer`（只读）的可登录系统账号）。

### Modified Capabilities
- `setup-wizard`: 初始化向导在密码步骤之后新增「如何称呼您」管理员资料步骤（昵称/邮箱/头像）。
- `platform-bootstrap`: 引导态中初始 admin 由 bootstrap 单账号扩展为「首个系统账号」，登录以系统账号为准。
- `admin-api-host`: 统一授权门禁从「已登录即可」升级为按角色 RBAC 授权（无权限返回 403）。

## Impact

- 后端：`internal/platform/bootstrap`（账号迁移、向导步骤状态）、`internal/httpapi/setup`（登录/改密/向导步骤/注册）、`internal/httpapi/adminhost`（RBAC 授权中间件）、新增系统账号持久化（业务库 `console_users`）、`internal/platform/settings`（开放注册开关）。
- 前端：`web/admin/src/features/setup`（新增「如何称呼您」步骤）、系统设置「用户管理」页、自助注册页。
- 兼容：消息平台终端用户 API/页面与既有业务 API 路径不变；内部表更名经 `db.RenameLegacy` 幂等迁移，保留既有消息平台用户数据；登录会话升级为标识系统账号 id 与角色。
- 依赖：业务库 AutoMigrate 新增系统账号表；无需新增第三方依赖。
