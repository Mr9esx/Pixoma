## 1. 系统账号数据模型与后端基础

- [x] 1.1 新建 `ConsoleUser` Domain 与业务库 `console_users` 表（账号/邮箱/昵称/头像/启用/角色/bcrypt 密码哈希/时间戳），AutoMigrate 注册
- [x] 1.2 消息平台终端用户表更名迁移：`users`→`channel_users`、`user_external_identities`→`channel_user_external_identities`（改 `TableName()` + `db.RenameLegacy` 幂等迁移保留数据，含测试）
- [x] 1.3 定义角色与权限常量（`Admin`/`Operator`/`Viewer` 及权限点集合、`Admin` 独占权限点）
- [x] 1.4 实现 bootstrap 初始 `admin` 幂等迁移为系统账号（保留用户名与凭据）
- [x] 1.5 `settings` 新增 `allow_self_registration` 开关（默认 false）及其读写接口

## 2. 登录/认证与会话升级

- [x] 2.1 登录改为以系统账号认证；会话签发升级为「系统账号 id + 角色」
- [x] 2.2 重构改密：管理员改密走系统账号；支持「强制下次改密」标记
- [x] 2.3 向导「如何称呼您」后端接口：保存初始 admin 昵称/邮箱/头像并推进 `WizardStep`

## 3. RBAC 授权门禁

- [x] 3.1 在 `adminhost.Gate` 认证后叠加 RBAC：从会话加载角色权限，按接口所需权限点校验
- [x] 3.2 账号管理/角色管理/删除类接口仅 `Admin`；未授权返回 403
- [x] 3.3 单测：未认证 401 / 无权限 403 / 有权限通过 / 拒绝删除唯一 Admin

## 4. 系统设置用户管理界面 + API

- [x] 4.1 用户管理 API：新增（账号/邮箱/昵称/密码）、禁用/启用、删除、重置密码、列表与搜索
- [x] 4.2 系统设置新增「用户管理」页：列表/新增/禁用/启用/删除/重置密码交互
- [x] 4.3 系统设置新增「是否开放注册」开关 UI 并绑定接口
- [x] 4.4 用户管理与设置契约测试（前端 contract + 后端 handler test）

## 5. 前端向导步骤与自助注册

- [x] 5.1 setup 向导在改密后插入「如何称呼您」步骤（昵称/邮箱/头像，标题「如何称呼您」，支持跳过）
- [x] 5.2 自助注册页（仅注册开关开启时可访问）与注册接口前端接入（注册账号默认角色 `Viewer`）
- [x] 5.3 向导与注册页 contract 测试

## 6. 文档与归档

- [x] 6.1 归档本 change delta specs 到 `docs/openspec/specs/`（新增 console-user-accounts/user-rbac/admin-user-management/self-registration，修改 setup-wizard/platform-bootstrap/admin-api-host）
- [x] 6.2 升级/迁移说明：既有部署如何迁移 bootstrap admin、为何需重新登录
