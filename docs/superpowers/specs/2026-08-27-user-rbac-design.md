---
comet_change: user-rbac
role: technical-design
canonical_spec: openspec
---

# 用户 + RBAC 深度技术设计

## 1. 概述

在现有 bootstrap 单管理员 + 统一登录门禁之上，引入面向后台控制台的系统账号体系与 RBAC，并支持：首启向导「如何称呼您」资料步骤、系统设置「用户管理」、系统设置「是否开放注册」开关与自助注册。

相关 OpenSpec：`console-user-accounts`、`user-rbac`、`admin-user-management`、`self-registration`（新增），`setup-wizard`、`platform-bootstrap`、`admin-api-host`（修改）。

## 2. 架构总览

```
┌────────────────────────── admin web (web/admin) ──────────────────────────┐
│  首启向导(改密→如何称呼您→…→storage)   系统设置[用户管理][开放注册]   注册页  │
└───────────────┬──────────────────────────────────────────────────────────┘
                │ /api/v1
┌───────────────▼──────────────────────────────────────────────────────────┐
│  adminhost.Gate                                                        │
│  ① 引导态门闩(未初始化仅放行 setup/login)                                 │
│  ② 会话校验(account_id + role)                                           │
│  ③ RBAC 权限点校验 → 403                                                │
└──────┬─────────────────┬─────────────────┬────────────────────┬─────────┘
  /api/v1/setup   /api/v1/adminusers  子资源路由(edges/cases/…)   /api/v1/auth
  (login/password/ (用户管理, 仅admin)  (声明所需权限点)            (注册/登出)
   profile/wizard)
      │
      ▼
┌──────────────┬───────────────────────────────────────────────┐
│ bootstrap.db │ 业务库 AutoMigrate                             │
│ 引导态门闩    │  console_users(系统账号)  channel_users(消息平台用户) │
│  向导进度     │  channel_user_external_identities               │
└──────────────┴───────────────────────────────────────────────┘
```

## 3. 数据模型

### 3.1 `console_users`（新增，业务库）
| 列 | 类型 | 说明 |
|---|---|---|
| `id` | uuid PK | 系统账号 |
| `username` | string 唯一 | 账号名（登录名）|
| `email` | string 唯一可空 | 邮箱，校验格式 |
| `nickname` | string | 昵称 |
| `avatar_url` | string | 头像 blob key（复用 `blob.Store`）|
| `role` | string | `admin` / `operator` / `viewer` |
| `enabled` | bool 默认 true | 禁用后不可登录 |
| `must_change_password` | bool 默认 false | 管理员重置密码后置 true，强制下次改密 |
| `password_hash` | string | bcrypt |
| `last_login_at` / `created_at` / `updated_at` | timestamps | |

- GORM `ConsoleUser` Domain + `internal/identity`（或新 `internal/consoleuser`）持久化；AutoMigrate 注册。
- 约束：账号名唯一；邮箱唯一可空；密码 ≥8 位 bcrypt；列表接口不返回 `password_hash`。

### 3.2 消息平台终端用户表更名（数据保留）
- `users` → `channel_users`；`user_external_identities` → `channel_user_external_identities`。
- 仅改 GORM `TableName()`（代码无裸 SQL 依赖）；在 `internal/platform/db/rename_legacy.go` 追加幂等迁移：`if mig.HasTable(old) && !mig.HasTable(new) { mig.RenameTable(...) }`，于 AutoMigrate 前执行。

## 4. 认证与会话

- 登录：以 `console_users` 校验（`enabled` 且 bcrypt 匹配）；引导态（未初始化）仍走 bootstrap 登录。
- 会话：`setup.Sessions.session` 由 `Username` 升级为 `{AccountID, Role, Username, Expires, Remember}`；签发与 Lookup 相应扩展。
- 迁移：部署时 bootstrap `admin` 幂等迁入 `console_users` 首条（role=admin，保留凭据与 `must_change_password`）。
- 兼容：既有 remember-me 会话（旧 schema）升级后失效，需重新登录；升级说明明示。

## 5. RBAC

- 权限点常量：`account.manage`、`account.delete`、`case.manage`、`case.read`、`channel.manage`、…（按需声明）。
- 角色映射：`Admin`=全量；`Operator`=业务运维（不含 `account.manage`/`account.delete`）；`Viewer`=仅只读。
- 授权：`adminhost.Gate` 认证后，按会话 `role` 解析权限集合；受保护路由声明所需权限点（`RequirePermission` 中间件），不足返回 `403`，不执行副作用。
- 拦截规则：未登录 `401`；已登录无权限 `403`；重复声明或未声明权限的默认策略：新增敏感接口一律显式声明。

## 6. 用户管理 & 注册 & 设置

- 用户管理 API（新包 `internal/httpapi/adminusers`，仅 `account.manage`）：`POST /`（新增）、`PATCH /{id}`（禁用/启用/改角色/重置密码）、`DELETE /{id}`、`GET /`（列表/搜索）。
- 保护：拒绝删除/禁用唯一 `Admin`；重置密码置 `must_change_password=true`；`DELETE` 前基本级确认（前端）。
- 自助注册 API（`/api/v1/auth/register`，受 `settings.allow_self_registration` 门控，默认 `false`；关闭返回 `409`）：创建角色 **`Viewer`** 账号并签发会话。
- 系统设置：`settings` 新增 `allow_self_registration`（`settings.go` + 读写接口 + 前端开关）。

## 7. 前端

- `web/admin/src/features/setup`：`setup-steps.ts` 在 `password` 后插入 `profile` 步骤（标题「如何称呼您」；昵称必填、邮箱/头像可选、可跳过）；`initialSetupStep`/进度同步 `bootstrap.WizardStep`。
- 系统设置（`routes/_app/settings/`）新增「用户管理」页（列表/新增/禁用/启用/删除/重置密码）与「是否开放注册」开关。
- 新增注册页；`auth-gates` 按注册开关放行。
- 复用既有 `*.contract.test.ts` 模式补充契约测试。

## 8. 错误处理

- 登录失败：`401 invalid credentials`；禁用账号：`403 account disabled`（或 `401`，避免枚举）。
- 越权：`403 forbidden`。
- 注册关闭：`409 registration disabled`；冲突：`409 username/email taken`。
- 校验失败：`400` 中文可诊断错误。

## 9. 风险 / 缓解

- [开放注册扩大攻击面] → 默认关闭；注册默认只读 `Viewer`，受 RBAC 约束，管理员可禁用/删除。
- [会话/表更名迁移] → `RenameLegacy` 幂等保留数据；remember-me 旧会话失效重新登录，迁移说明明示。
- [bootstrap 双源] → 引导态后一律以系统账号为准，禁用账号即刻失效。
- [RBAC 粒度] → 权限点驱动、角色为预置组合，后续可扩展。

## 10. 测试策略

- 后端单测：会话升级、RBAC 矩阵（401/无权限403/允许/拒绝删唯一admin/禁用账号登录）、`users→channel_users` 表更名保留数据、bootstrap 迁移幂等、注册门控与默认 `Viewer`、邮箱/密码校验。
- 前端 contract：用户管理页交互、注册开关、向导「如何称呼您」步骤（可跳过/非法邮箱）。
- 集成：初始化流程（改密→资料→db→storage→finalize）、既有部署升级路径。

## 11. Out of Scope

- 消息平台终端用户目录页/API 行为不变。
- 用户自助改密与个人资料自助管理（本 change 不含，仅管理员重置）。
- 密码找回、二次验证、登录失败锁定、细粒度自定义角色。
