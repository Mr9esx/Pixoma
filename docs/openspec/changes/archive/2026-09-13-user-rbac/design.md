## Context

现有后台只有 bootstrap 单管理员：`bootstrap.db` 存一个 `admin` 账号的 bcrypt 哈希与 `must_change_password`，`setup.Sessions` 按用户名签发会话，`adminhost.Gate` 统一门禁只区分「已登录/未登录」。首次向导步骤为 `password → database → storage`，`WizardStep` 记录进度。消息平台终端用户目录（`internal/httpapi/users`）是只读的 TG 用户视图，与登录后台的系统账号无关。系统设置持久化在业务库 `settings.Store`。

目标是引入控制台多用户账号体系 + RBAC：首启向导改密后采集初始 admin 资料（昵称/邮箱/头像），系统设置提供用户管理（新增/禁用/删除/改密）与「是否开放注册」开关。约束：不新增第三方依赖；复用既有 blob、settings、统一鉴权 gate；除非必要不改业务接口路径。

## Goals / Non-Goals

**Goals:**
- 建立控制台系统账号模型（账号/邮箱/昵称/头像/启用状态/角色），作为登录与鉴权主体。
- RBAC 授权：`Admin`/`Operator`/`Viewer` 三档角色，统一 gate 按权限点拒绝未授权请求（403）。
- 系统设置「用户管理」：新增、禁用/启用、删除、重置密码。
- 系统设置「是否开放注册」开关 + 自助注册流程。
- 首启向导改密后新增「如何称呼您」步骤，写初始 admin 资料。

**Non-Goals:**
- 不改造消息平台终端用户目录（`users` 资源页）及其 API。
- 不做角色自定义/细粒度自定义权限组合（以权限点驱动、角色为预置组合）。
- 不做公开公网用户体系；注册对象为后台控制台账号。
- 不新增密码找回、二次验证、登录失败锁等账号安全旁路能力（可作为后续 change）。

## Decisions

### 1. 系统账号独立于消息平台终端用户
新建 `console_users` 表（Domain `ConsoleUser`）承载控制台账号，不复用 `internal/identity` 的消息平台用户聚合。现有 `users` 资源页语义（消息平台终端用户）保持不变。为避免语义混淆，消息平台终端用户的内部表由 `users` 更名为 `channel_users`（配套外身份映射表 `user_external_identities` → `channel_user_external_identities`）。
**备选**（放弃）：扩展 identity.User —— 会混淆「后台登录账号」与「TG 终端用户」两个完全不同的身份域。

### 2. 账号持久化到业务库
`console_users` 由业务库 AutoMigrate 创建；密码存 bcrypt 哈希。登录、改密、用户管理均操作该表。
**备选**（放弃）：账号塞进 bootstrap.db —— bootstrap 是引导态本地单库，无法支持多用户与业务库一致的多节点/共享部署。

### 3. bootstrap 初始 admin 迁移为系统账号
初始化时把 bootstrap 的 `admin` 凭据（用户名、bcrypt 哈希、`must_change_password`）迁为 `console_users` 首个记录（角色 `Admin`），登录以系统账号为准；bootstrap 仅承担「未初始化引导态」的时序门闩。向导「如何称呼您」步骤即为此初始 admin 补昵称/邮箱/头像。
**备选**（放弃）：bootstrap 与系统账号双源并存长期 —— 会造成登录认证分叉，难维护。

### 4. RBAC 简单角色 + 权限点驱动
预置三档角色：`Admin`（全量）、`Operator`（业务运维，不含账号/角色管理与删除）、`Viewer`（只读）。接口按所需权限点声明，授权中间件用「角色→权限点」集合判断，未授权返回 403。实现上以 permission 集为事实，角色只是权限组合，便于未来扩展细粒度自定义角色。
**备选**（放弃）：仅布尔「admin/非 admin」—— 满足不了只读运维与分权的诉求。

### 5. 会话升级为标识系统账号
`setup.Sessions` 签发内容从「用户名」升级为「系统账号 id + 角色」，登录请求以系统账号认证；升级/迁移后既有会话失效需重新登录（可接受）。RBAC 中间件从会话读取账号 id 与角色，规避每次请求改库。
**备选**（放弃）：每次请求查库取角色 —— 简单但增加 DB 命中与一致性问题。

### 6. 开放注册开关默认关闭
`settings` 新增 `allow_self_registration`（默认 `false`）。开启时提供自助注册接口与注册页；注册成功创建系统账号（**默认角色 `Viewer`（只读）**）并可直接登录，受统一 RBAC 约束。关闭时注册请求被拒（403/409）。
**备选**（放弃）：不做注册仅靠管理员开账号 —— 无法满足「开放注册」需求。

### 7. 授权门禁落地位置
在 `adminhost.Gate` 认证通过后叠加 RBAC：账号管理/角色/注册相关接口要求 `Admin`（或对应权限点）；`setup` 登录与向导步骤按现状放行引导态。用户管理 API 在 `internal/httpapi/users` 之外新增 `internal/httpapi/accountauth`（或 `adminusers`）路由，避免与消息平台 `users` 汉语语义混淆。

### 8. 消息平台终端用户表更名迁移
消息平台终端用户内部表 `users`/`user_external_identities` 更名为 `channel_users`/`channel_user_external_identities`，仅改 GORM `TableName()`（无裸 SQL 依赖）。在 `db.RenameLegacy` 中追加幂等重命名（`HasTable(old) && !HasTable(new)` 时 `RenameTable`），于 AutoMigrate 前执行，保留既有数据；新装库幂等跳过。
**备选**（放弃）：继续沿用 `users` —— 与新增的 `console_users` 并存时语义易混淆。

## Risks / Trade-offs

- [开放注册扩大攻击面] → 默认关闭；开启时注册账号受统一 RBAC 约束，可被管理员禁用/删除。
- [bootstrap→系统账号迁移破坏既有会话] → 升级说明建议重新登录；迁移逻辑幂等、失败可回滚（bootstrap 保留）。
- [角色粒度过粗或过细] → 以权限点驱动实现，角色仅预置组合，后续可微调矩阵而不改中间件。
- [双源认证过渡期混淆] → 约定：未初始化阶段仍走 bootstrap 登录；初始化完成后登录/鉴权一律以系统账号为准。

## Migration Plan

1. 后端先落地：`console_users` 表 + 迁移 bootstrap admin + RBAC 中间件 + 用户管理 API + 注册接口 + settings 开关 + `users`→`channel_users` 更名迁移（`db.RenameLegacy`）。
2. 前端落地：系统设置「用户管理」页与「开放注册」开关、自助注册页，最后接入向导「如何称呼您」步骤。
3. 既有无初始化部署：启动时自动迁移 bootstrap admin 为系统账号并保持原凭据；未完成向导的补齐「如何称呼您」步骤。
4. 回滚：关闭 RBAC 门禁降级为「已登录即可」，保留账号数据不破坏；关闭注册开关即可终止自助注册。

## Open Questions

均已在本轮评审中收敛：
- 自助注册默认角色：**`Viewer`（只读）**，注册即可登录但仅只读，不自动授运维/管理权限。
- 自助改密与个人资料管理：**本 change 不包含**，改密仅由管理员重置；如需自助改密/资料编辑作为后续 change。

## 升级与迁移说明

### 消息平台用户表更名
- `users` → `channel_users`、`user_external_identities` → `channel_user_external_identities`。
- 由 `db.RenameLegacy` 幂等执行：仅当旧表存在且新表不存在时才重建并保留数据；冷启动/新库自动跳过。无需人工 DDL。

### bootstrap admin 迁移为控制台账号
- 应用启动时 `MigrateBootstrapAdmin` 幂等：当 `console_users` 尚无该账号时，用 bootstrap 首启凭据创建 `role=admin` 的账号（用户名、bcrypt 密码哈希、`must_change_password` 状态原样保留）。
- 首启向导「如何称呼您」填写的昵称/邮箱/头像先落在 bootstrap，最终在迁移时写入 `console_users`。
- 已有部署升级后：原 bootstrap 管理员可继续用用户名+旧密码登录；若密码在旧流程已被改过，仍需改密标记保持一致。

### 为什么需要重新登录
- 会话升级为「账号 id + 角色」驱动，旧会话令牌结构不包含 `account_id`/`role`，RBAC 授权无法识别。
- 升级后已登录的管理员应退出并重新登录一次，否则会被要求再次认证；这是有意为之的安全边界。
- 系统设置改动仍需重启（`restart_required`）才会加载新配置；升级包应提示重启一次。

### 权限与角色
- 角色：`Admin` / `Operator` / `Viewer`；默认自助注册账号为 `Viewer`（只读）。
- 账号管理/删除/角色调整仅在 `Admin` 权限下可用；`Viewer` 调用返回 403。
- 「是否开放注册」默认关闭；关闭时 `/api/v1/auth/register` 返回 409，前端注册页不可访问。
