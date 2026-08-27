# Comet Design Handoff

- Change: user-rbac
- Phase: design
- Mode: compact
- Context hash: fed72a357e10603ed77fc5909ad287f5787c9308b9bd3054faeb68a2e61e0d5f

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/user-rbac/proposal.md

- Source: docs/openspec/changes/user-rbac/proposal.md
- Lines: 1-34
- SHA256: 5ad080093340c2daa4a24c6abf63b8e7e4de3ad62b469e7d4c7244d34c17f4bf

```md
## Why

Pixoma 后台目前只有一个 bootstrap 单管理员账号（默认 `admin`），登录后所有管理接口一律放行，没有角色与权限区分，也不能创建/管理其他系统账号；首次初始化向导改密后直接就进入数据库配置，缺少管理员资料的采集。随着多运维协作与内网共享部署，需要一个面向「登录后台的控制台账号」的多用户体系与 RBAC，并允许按需开放自助注册。

## What Changes

- 新增控制台系统账号模型：账号名、邮箱、昵称、头像、启用状态、角色；作为管理后台登录与鉴权的主体，与已有的渠道终端用户目录（`users` 资源页）明确区分。
- 新增 RBAC：`Admin` / `Operator` / `Viewer` 三档角色，按角色映射权限点，统一授权门禁对受限管理接口按当前登录账号的权限做 403 校验。
- 首次初始化向导在「更新密码」之后新增一个「如何称呼您」步骤，采集初始管理员的昵称、邮箱、头像，并写入该 admin 系统账号。
- 系统设置新增「用户管理」：管理员可新增账号（账号名、邮箱、昵称、密码）、禁用/启用、删除、重置密码。
- 系统设置新增「是否开放注册」开关（默认关闭）；开启后允许自助注册控制台账号。
- 把 bootstrap 初始 `admin` 迁移进新的系统账号模型，保留凭据并补齐资料。
- 将渠道终端用户内部表由 `users` 更名为 `channel_users`（`user_external_identities` → `channel_user_external_identities`），通过 `db.RenameLegacy` 幂等迁移保留既有数据。
- 不改变：渠道终端用户目录页与 API 行为、各业务功能（Case/Channel/Edge 等）自身行为；仅本 change 不含用户自助改密与个人资料自助管理（改密仅由管理员重置）；对渠道终端用户仅做内部表更名（API/展示不变），对管理接口施加 RBAC 授权。

## Capabilities

### New Capabilities
- `console-user-accounts`: 控制台登录账号模型（账号/邮箱/昵称/头像/启用状态/角色/认证），区别于渠道终端用户；bootstrap 初始 admin 迁移为系统账号。
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
- 兼容：渠道终端用户 API/页面与既有业务 API 路径不变；内部表更名经 `db.RenameLegacy` 幂等迁移，保留既有渠道用户数据；登录会话升级为标识系统账号 id 与角色。
- 依赖：业务库 AutoMigrate 新增系统账号表；无需新增第三方依赖。

```

## docs/openspec/changes/user-rbac/design.md

- Source: docs/openspec/changes/user-rbac/design.md
- Lines: 1-73
- SHA256: 86764f6cdee911400a17ac50a8bf7124a29e64768a05e1c3e4e849684318bf39

```md
## Context

现有后台只有 bootstrap 单管理员：`bootstrap.db` 存一个 `admin` 账号的 bcrypt 哈希与 `must_change_password`，`setup.Sessions` 按用户名签发会话，`adminhost.Gate` 统一门禁只区分「已登录/未登录」。首次向导步骤为 `password → database → storage`，`WizardStep` 记录进度。渠道终端用户目录（`internal/httpapi/users`）是只读的 TG 用户视图，与登录后台的系统账号无关。系统设置持久化在业务库 `settings.Store`。

目标是引入控制台多用户账号体系 + RBAC：首启向导改密后采集初始 admin 资料（昵称/邮箱/头像），系统设置提供用户管理（新增/禁用/删除/改密）与「是否开放注册」开关。约束：不新增第三方依赖；复用既有 blob、settings、统一鉴权 gate；除非必要不改业务接口路径。

## Goals / Non-Goals

**Goals:**
- 建立控制台系统账号模型（账号/邮箱/昵称/头像/启用状态/角色），作为登录与鉴权主体。
- RBAC 授权：`Admin`/`Operator`/`Viewer` 三档角色，统一 gate 按权限点拒绝未授权请求（403）。
- 系统设置「用户管理」：新增、禁用/启用、删除、重置密码。
- 系统设置「是否开放注册」开关 + 自助注册流程。
- 首启向导改密后新增「如何称呼您」步骤，写初始 admin 资料。

**Non-Goals:**
- 不改造渠道终端用户目录（`users` 资源页）及其 API。
- 不做角色自定义/细粒度自定义权限组合（以权限点驱动、角色为预置组合）。
- 不做公开公网用户体系；注册对象为后台控制台账号。
- 不新增密码找回、二次验证、登录失败锁等账号安全旁路能力（可作为后续 change）。

## Decisions

### 1. 系统账号独立于渠道终端用户
新建 `console_users` 表（Domain `ConsoleUser`）承载控制台账号，不复用 `internal/identity` 的渠道用户聚合。现有 `users` 资源页语义（渠道终端用户）保持不变。为避免语义混淆，渠道终端用户的内部表由 `users` 更名为 `channel_users`（配套外身份映射表 `user_external_identities` → `channel_user_external_identities`）。
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
在 `adminhost.Gate` 认证通过后叠加 RBAC：账号管理/角色/注册相关接口要求 `Admin`（或对应权限点）；`setup` 登录与向导步骤按现状放行引导态。用户管理 API 在 `internal/httpapi/users` 之外新增 `internal/httpapi/accountauth`（或 `adminusers`）路由，避免与渠道 `users` 汉语语义混淆。

### 8. 渠道终端用户表更名迁移
渠道终端用户内部表 `users`/`user_external_identities` 更名为 `channel_users`/`channel_user_external_identities`，仅改 GORM `TableName()`（无裸 SQL 依赖）。在 `db.RenameLegacy` 中追加幂等重命名（`HasTable(old) && !HasTable(new)` 时 `RenameTable`），于 AutoMigrate 前执行，保留既有数据；新装库幂等跳过。
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

```

## docs/openspec/changes/user-rbac/tasks.md

- Source: docs/openspec/changes/user-rbac/tasks.md
- Lines: 1-37
- SHA256: e6abdf8e128a4c8cc6c814bdf2435c2598715eeba37a60d5dccd4f894024e556

```md
## 1. 系统账号数据模型与后端基础

- [ ] 1.1 新建 `ConsoleUser` Domain 与业务库 `console_users` 表（账号/邮箱/昵称/头像/启用/角色/bcrypt 密码哈希/时间戳），AutoMigrate 注册
- [ ] 1.2 渠道终端用户表更名迁移：`users`→`channel_users`、`user_external_identities`→`channel_user_external_identities`（改 `TableName()` + `db.RenameLegacy` 幂等迁移保留数据，含测试）
- [ ] 1.3 定义角色与权限常量（`Admin`/`Operator`/`Viewer` 及权限点集合、`Admin` 独占权限点）
- [ ] 1.4 实现 bootstrap 初始 `admin` 幂等迁移为系统账号（保留用户名与凭据）
- [ ] 1.5 `settings` 新增 `allow_self_registration` 开关（默认 false）及其读写接口

## 2. 登录/认证与会话升级

- [ ] 2.1 登录改为以系统账号认证；会话签发升级为「系统账号 id + 角色」
- [ ] 2.2 重构改密：管理员改密走系统账号；支持「强制下次改密」标记
- [ ] 2.3 向导「如何称呼您」后端接口：保存初始 admin 昵称/邮箱/头像并推进 `WizardStep`

## 3. RBAC 授权门禁

- [ ] 3.1 在 `adminhost.Gate` 认证后叠加 RBAC：从会话加载角色权限，按接口所需权限点校验
- [ ] 3.2 账号管理/角色管理/删除类接口仅 `Admin`；未授权返回 403
- [ ] 3.3 单测：未认证 401 / 无权限 403 / 有权限通过 / 拒绝删除唯一 Admin

## 4. 系统设置用户管理界面 + API

- [ ] 4.1 用户管理 API：新增（账号/邮箱/昵称/密码）、禁用/启用、删除、重置密码、列表与搜索
- [ ] 4.2 系统设置新增「用户管理」页：列表/新增/禁用/启用/删除/重置密码交互
- [ ] 4.3 系统设置新增「是否开放注册」开关 UI 并绑定接口
- [ ] 4.4 用户管理与设置契约测试（前端 contract + 后端 handler test）

## 5. 前端向导步骤与自助注册

- [ ] 5.1 setup 向导在改密后插入「如何称呼您」步骤（昵称/邮箱/头像，标题「如何称呼您」，支持跳过）
- [ ] 5.2 自助注册页（仅注册开关开启时可访问）与注册接口前端接入（注册账号默认角色 `Viewer`）
- [ ] 5.3 向导与注册页 contract 测试

## 6. 文档与归档

- [ ] 6.1 归档本 change delta specs 到 `docs/openspec/specs/`（新增 console-user-accounts/user-rbac/admin-user-management/self-registration，修改 setup-wizard/platform-bootstrap/admin-api-host）
- [ ] 6.2 升级/迁移说明：既有部署如何迁移 bootstrap admin、为何需重新登录

```

## docs/openspec/changes/user-rbac/specs/admin-api-host/spec.md

- Source: docs/openspec/changes/user-rbac/specs/admin-api-host/spec.md
- Lines: 1-16
- SHA256: 8a8f805de9e05d3ccaff5f707c2278d883c0ab31d85712a93515c6d55fbde690

```md
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

```

## docs/openspec/changes/user-rbac/specs/admin-user-management/spec.md

- Source: docs/openspec/changes/user-rbac/specs/admin-user-management/spec.md
- Lines: 1-31
- SHA256: b04ab536cd1dd446692aff1b22d27a84f6f9bf507cbe2d5582d03aefbe77e10a

```md
## ADDED Requirements

### Requirement: 系统设置用户管理
系统设置 MUST 提供「用户管理」界面，管理员（`Admin`）可管理系统账号：新增用户（账号名、邮箱、昵称、密码）、禁用/启用用户、删除用户、重置用户密码。删除与禁用操作 MUST 需要有明确确认，且不得删除当前会话所属的最后一个 `Admin` 账号。

#### Scenario: 新增用户
- **WHEN** 管理员在用户管理页填写账号名、邮箱、昵称、密码并提交
- **THEN** 系统创建该系统账号并出现在用户列表

#### Scenario: 禁用并阻止登录
- **WHEN** 管理员禁用某系统账号
- **THEN** 该账号立即无法登录或使用会话；再次启用后可恢复

#### Scenario: 删除用户
- **WHEN** 管理员删除某系统账号且该账号不是唯一 `Admin`
- **THEN** 系统删除该账号并使其会话失效

#### Scenario: 阻止删除最后一个 Admin
- **WHEN** 管理员尝试删除或禁用自己的账号且它是唯一的 `Admin` 账号
- **THEN** 系统拒绝该操作并返回可诊断错误

#### Scenario: 重置密码
- **WHEN** 管理员对某被重置用户执行重置密码
- **THEN** 该用户新密码生效，旧密码立即失效（可用强制下次改密标记）

### Requirement: 用户列表与查询
用户管理 MUST 提供系统账号的列表与搜索，展示账号名、邮箱、昵称、角色、启用状态与最近登录/更新时间；列表接口 MUST 不返回密码哈希。

#### Scenario: 搜索并查看列表
- **WHEN** 管理员在用户管理页搜索账号名或邮箱
- **THEN** 列表按关键字过滤展示匹配系统账号，且不含密码哈希字段

```

## docs/openspec/changes/user-rbac/specs/console-user-accounts/spec.md

- Source: docs/openspec/changes/user-rbac/specs/console-user-accounts/spec.md
- Lines: 1-41
- SHA256: 9674b5d6e202054dcf25abde434309946d190b1911efb8ec9163550a691d8b64

```md
## ADDED Requirements

### Requirement: 控制台系统账号模型
系统 MUST 提供面向后台控制台登录的系统账号模型，与渠道终端用户目录相区分；每个系统账号 MUST 包含：账号名（唯一）、邮箱（可选唯一）、昵称、头像、启用状态、角色；密码 MUST 以 bcrypt 哈希存储，不得明文或可逆明文落库。

#### Scenario: 新建系统账号
- **WHEN** 管理员通过用户管理新建一个系统账号并填写账号名、邮箱、昵称、密码
- **THEN** 系统创建该账号并分配默认角色，密码以 bcrypt 哈希持久化

#### Scenario: 账号名唯一冲突
- **WHEN** 新建或改名的账号名与既有系统账号冲突
- **THEN** 系统拒绝并返回可诊断错误，不写入该账号

### Requirement: bootstrap 初始 admin 迁移为系统账号
系统初始化时 MUST 将引导态初始 `admin` 账号迁移为系统账号（角色 `Admin`），保留其用户名与凭据；后续管理后台登录与鉴权 MUST 以系统账号为准，而非 bootstrap 单账号。

#### Scenario: 升级后首次登录
- **WHEN** 平台升级后管理员用既有 `admin` 账号与密码登录
- **THEN** 登录通过且该账号在系统账号表中为 `Admin` 角色

#### Scenario: 迁移幂等
- **WHEN** 已迁移过初始 admin 后再次启动迁移过程
- **THEN** 不重复创建账号，迁移幂等

### Requirement: 系统账号与渠道终端用户隔离
系统账号 MUST NOT 出现在渠道终端用户目录（`users` 资源页）中，二者数据与语义分离。

#### Scenario: 目录互不包含
- **WHEN** 管理员查看渠道终端用户目录或系统账号列表
- **THEN** 两类账号互不混淆，各按其来源与语义展示

### Requirement: 渠道终端用户表更名保留数据
渠道终端用户内部表 MUST 由 `users`/`user_external_identities` 更名为 `channel_users`/`channel_user_external_identities`，更名 MUST 幂等且保留既有数据；系统账号表 MUST 为 `console_users`，三表语义不混。

#### Scenario: 既有库更名保留数据
- **WHEN** 含既有渠道用户数据的 `users` 表被迁移至 `channel_users`
- **THEN** 旧表被重命名（非清空/重建），既有渠道用户记录完整保留

#### Scenario: 新装库幂等
- **WHEN** 在全新数据库上执行初始化/迁移
- **THEN** 直接按新表名建表，更名步骤为无害幂等跳过

```

## docs/openspec/changes/user-rbac/specs/platform-bootstrap/spec.md

- Source: docs/openspec/changes/user-rbac/specs/platform-bootstrap/spec.md
- Lines: 1-12
- SHA256: 548d9b9db01a8ac42c7f834dfe4f56dd14a7d7a362bbead4fdc04b25801368bd

```md
## ADDED Requirements

### Requirement: 系统账号是登录主体
平台初始化完成后，管理后台登录、改密、会话与授权 MUST 以系统账号（`console_users`）为准；引导态 bootstrap 仅承担未初始化阶段的门闩与向导时序，MUST NOT 承担初始化后的登录认证主体。

#### Scenario: 初始化后登录走系统账号
- **WHEN** 平台已完成初始化，管理员登录后台
- **THEN** 认证校验系统账号表，成功则签发标识系统账号 id 与角色的会话

#### Scenario: 引导态仍可引导登录
- **WHEN** 平台尚未初始化或处于向导阶段
- **THEN** 仍走引导态登录与向导接口，完成初始化后再以系统账号登录

```

## docs/openspec/changes/user-rbac/specs/self-registration/spec.md

- Source: docs/openspec/changes/user-rbac/specs/self-registration/spec.md
- Lines: 1-27
- SHA256: 2d3e5139b2671ecb76cd19d259b54bf015ff654e7562e9fb8901ef4f1fff3cf3

```md
## ADDED Requirements

### Requirement: 开放注册开关
系统设置 MUST 提供「是否开放注册」开关（默认关闭）。开启后系统提供自助注册流程；关闭后自助注册请求 MUST 被拒绝。

#### Scenario: 关闭时拒绝注册
- **WHEN** 系统设置「是否开放注册」为关闭状态，客户端提交注册请求
- **THEN** 系统拒绝注册（返回 409 或等效可诊断错误）且不创建账号

#### Scenario: 开启后可注册
- **WHEN** 系统设置「是否开放注册」为开启状态，用户提交合法注册
- **THEN** 系统创建系统账号并允许其登录

### Requirement: 自助注册
自助注册 MUST 创建控制台系统账号（账号名、邮箱、昵称、密码），并默认分配 **`Viewer`（只读）** 角色；注册成功后账号可登录但仅只读访问，不自动获得运维或管理权限；已注册账号名或邮箱冲突 MUST 被拒绝。

#### Scenario: 注册默认只读
- **WHEN** 用户自助注册成功并用该账号登录
- **THEN** 账号角色为 `Viewer`，只能只读访问，触达受限管理接口返回 403

#### Scenario: 注册并登录
- **WHEN** 用户在开启注册时填账号名、邮箱、昵称、密码并提交注册，随后用该账号登录
- **THEN** 账号创建成功且可登录，访问范围受其角色约束

#### Scenario: 账号名冲突
- **WHEN** 注册账号名与既有系统账号冲突
- **THEN** 系统拒绝并提示账号名已被占用，不创建账号

```

## docs/openspec/changes/user-rbac/specs/setup-wizard/spec.md

- Source: docs/openspec/changes/user-rbac/specs/setup-wizard/spec.md
- Lines: 1-23
- SHA256: ba6d9a022140db0283f3e7e8025cbb88e11fe6046d37ab7a72685bdc29f1cba4

```md
## ADDED Requirements

### Requirement: 初始化向导管理员资料步骤
初始化向导在「更新管理员密码」步骤之后 MUST 提供「如何称呼您」步骤，展示标题「如何称呼您」，采集初始管理员的昵称（必填）、邮箱（可选）、头像（可选）；提交后写入该初始 admin 系统账号资料并进入下一步。邮箱若填写，MUST 是合法邮箱格式。

#### Scenario: 改密后进入资料步骤
- **WHEN** 用户完成更新管理员密码步骤
- **THEN** 向导进入标题为「如何称呼您」的资料步骤，展示昵称、邮箱、头像字段

#### Scenario: 填写资料并继续
- **WHEN** 用户在「如何称呼您」步骤填写昵称（及可选邮箱/头像）并提交
- **THEN** 系统保存初始 admin 的昵称/邮箱/头像并推进到下一步骤

#### Scenario: 邮箱格式非法
- **WHEN** 用户在「如何称呼您」步骤填写非法的邮箱格式
- **THEN** 向导提示错误且不进入下一步骤

### Requirement: 管理员资料步骤可跳过
「如何称呼您」步骤（昵称/邮箱/头像）MUST 允许用户暂时跳过；跳过后关闭在任何步骤，且管理员后续可在用户管理中补全或修改个人资料。

#### Scenario: 跳过资料步骤
- **WHEN** 用户在「如何称呼您」步骤选择暂时跳过
- **THEN** 向导继续进入下一步骤，初始 admin 账号资料保持待补全状态

```

## docs/openspec/changes/user-rbac/specs/user-rbac/spec.md

- Source: docs/openspec/changes/user-rbac/specs/user-rbac/spec.md
- Lines: 1-23
- SHA256: 1d93e7269ac769960a563037cfb3a50c705535e59600d7ea8421546e23923cf1

```md
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

```
