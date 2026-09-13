# Brainstorm Summary

- Change: user-rbac
- Date: 2026-08-27

## Confirmed Technical Approach

- 系统账号模型 `console_users`（业务库）：username/email/nickname/avatar_url/role/enabled/must_change_password/password_hash( bcrypt)/时间戳；与渠道终端用户隔离。
- 渠道终端用户内部表更名：`users`→`channel_users`、`user_external_identities`→`channel_user_external_identities`，经 `db.RenameLegacy` 幂等迁移保留数据。
- bootstrap 初始 admin 迁移为首条系统账号（role=admin），登录一律以系统账号为准；引导态仍走 bootstrap 登录。
- 会话从存 username 升级为存 account_id+role；RBAC 在 `adminhost.Gate` 认证后叠加权限点校验，账户/角色/删除类仅 admin，越权 403。
- 用户管理 API 置于新包（与渠道 `users` API 分离，仅 admin 可访问）：新增/禁用/启用/删除/重置密码/列表搜索。
- 系统设置新增「用户管理」页与「是否开放注册」开关；新增自助注册页；向导改密后插入「如何称呼您」步骤。
- 用户已定：**自助注册默认角色 `Viewer`（只读）**；**不含用户自助改密/资料管理，改密仅管理员重置**。

## Key Trade-offs and Risks

- [开放注册攻击面] → 默认关闭，注册默认只读 Viewer，受统一 RBAC 约束，可被管理员禁用/删除。
- [bootstrap→系统账号迁移] → 幂等迁移、回滚保留 bootstrap；既有 remember-me 会话升级后失效需重登（迁移说明明示）。
- [表更名] → `db.RenameLegacy` 幂等 `HasTable(old)&&!HasTable(new)`，保留数据。
- [RBAC 粒度] → 权限点驱动、角色为预置组合，便于后续细粒度扩展。

## Testing Strategy

- 后端单测：会话升级、RBAC 矩阵（401/403/允许/拒绝删除唯一 admin）、表更名迁移保留数据、bootstrap 迁移幂等、注册门控（开关开/关）、默认角色 Viewer。
- 前端 contract：用户管理页、注册开关、向导「如何称呼您」步骤（`*.contract.test.ts` 模式）。

## Spec Patches

- `specs/self-registration/spec.md`：注册默认角色定为 `Viewer`，新增「注册默认只读」场景。
- `specs/console-user-accounts/spec.md`：新增「渠道终端用户表更名保留数据」要求。
