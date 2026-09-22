# Pixoma 错误文案表（最终稿）

> 本文是**面向用户的最终错误文案**，回答「改成什么」。
> 现状审计见 [`docs/api-error-audit.md`](./api-error-audit.md)（回答「现在错在哪」）。
> 出稿依据 [`docs/voice-profile.md`](./voice-profile.md)：事实句 + 动作句，短句、动作收尾、不卖萌。

## 1. 设计原则

每条文案必须同时满足三条：

| 要求 | 实现方式 |
| --- | --- |
| **可读性** | 中文短句，技术词不解释（DSN / blob / token / MCP 直接用），中英之间留空格 |
| **足够信息** | 说清**哪个实体**、**什么操作**、**什么原因**；不写「操作失败」这类空话 |
| **指导性** | 第二句必须是**界面上的可执行动作**：重新保存、刷新页面、等待后重试、去某个页面配置 |

### 1.1 三段式结构

```json
{
  "message": "用例被菜单或卡片引用。勾选「同时清理引用」后删除。",
  "code": 4090604,
  "data": null,
  "error_detail": "case is referenced by menu entries: menu-a, menu-b"
}
```

| 段 | 承载 | 产出方 |
| --- | --- | --- |
| 事实句 | 发生了什么、涉及哪个实体 | 后端（前端 i18n 可按 code 覆盖） |
| 动作句 | 用户下一步做什么 | 同上 |
| `error_detail` | 原始技术原因（脱敏后），前端直接展开显示 | 后端自动产出 |

### 1.2 message 与 error_detail 的分工

| 字段 | 读者 | 写什么 |
| --- | --- | --- |
| `message` | 界面上的用户 | 发生了什么，以及界面上能做的下一步动作 |
| `error_detail` | 同一个用户，前端直接展开显示 | 原始技术原因（脱敏后） |

`message` 只写交互动作：重新保存、刷新页面、等待 1 分钟后重试、去某个页面配置。
技术原因由 `error_detail` 承载，前端直接展示，`message` 里不写「查看日志」「定位原因」这类阅读方法。

### 1.3 voice profile 合规

- 禁用词已清零：`请` / `您` / `进行` / `完成` / `实施` / `执行` / `麻烦` / `前往` / `首先`。
- 实体名按域区分：`/api/v1/users` 是 bot 用户（profile 例外允许「用户」作实体名），`/api/v1/adminusers` 是控制台**管理员**。
- 无感叹号、无 emoji；中英之间留空格。

## 2. 通用文案模板

| 场景 | 模板 |
| --- | --- |
| 内部错误（5xx + 原始 error） | `{动作}{实体}失败。看 error_detail 定位原因；持续失败查后台日志。` |
| 资源不存在 | `{实体}不存在。可能已被删除，刷新列表后重试。` |
| 请求体格式 | `请求体不是合法 JSON。检查字段格式后重试。` |
| 未认证 | `登录已失效。重新登录。`（边缘 Agent 为 `凭证无效。检查实例的 agent token。`） |
| 无权限 | `没有权限。这个操作需要管理员。` |
| 服务未配置 | `{服务}未配置。检查 {配置项} 配置后重启 pixoma。` |
| 依赖不可达 | `{服务}不可用。检查 {配置项} 配置后重启 pixoma。` |
| 参数必填 | `{字段}不能为空。填{字段}后重试。` |
| 参数格式 | `{字段}不合法。{正确格式}。` |
| 唯一冲突 | `{实体}已存在。改个名称，或先删掉旧的。` |

## 3. 逐域文案表

共 **204** 条词条，覆盖 **438** 个错误返回点。

### 通用 / 权限（业务域 `00`，4 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4010007` | 401 | RequirePermission | `unauthorized` | **登录已失效。重新登录。** | 禁止（安全） |
| `4030008` | 403 | RequirePermission | `forbidden` | **没有权限。这个操作需要管理员。** | 禁止（安全） |
| `4130002` | 413 | RequestBodyLimit | `request body too large` | **请求内容超过大小上限。缩小后重新提交。** | — |
| `5000005` | 500 | Fail（兜底） | `internal error` | **服务暂时不可用。重启 `pixoma` 后重试。** | **必须** |

### 认证 / 初始化（业务域 `01`，41 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000102` | 400 | database | `dsn required` | **DSN 不能为空。填写数据库连接串后重试。** | — |
| `4000110` | 400 | finalize | `change default password first` | **仍在使用默认密码。先修改默认管理员密码。** | — |
| `4000111` | 400 | draft | `err.Error(` | **保存草稿失败。检查请求参数后重试。** | — |
| `4000112` | 400 | putSettings | `err.Error(` | **保存设置失败。检查请求参数后重试。** | — |
| `4000113` | 400 | putSettings | `finalize setup first` | **初始化尚未结束。先运行完初始化向导。** | — |
| `4000114` | 400 | finalize | `err.Error(` | **初始化收尾失败。检查请求参数后重试。** | — |
| `4000115` | 400 | password | `password already set` | **密码已经设置。需要修改密码时使用「修改密码」。** | — |
| `4000116` | 400 | password, register | `password must be at least 8 ch` | **密码至少 8 位。增加长度后重试。** | — |
| `4000117` | 400 | blobTest | `blob check failed: "+err.Error` | **对象存储连通失败。检查 Endpoint、Bucket 以及密钥后重新测试。** | 需要 |
| `4000118` | 400 | database | `database ping failed: "+err.Er` | **数据库 ping 失败。检查 DSN 和网络后重新测试。** | 需要 |
| `4000119` | 400 | draft, finalize, putSettings | `configure database first` | **数据库尚未配置。先在向导里填写 DSN。** | 需要 |
| `4000120` | 400 | database, draft | `database unreachable: "+err.Er` | **无法连接数据库。检查 DSN 和网络后重新测试。** | — |
| `4000122` | 400 | blobTest | `err.Error(` | **测试对象存储失败。检查请求参数后重试。** | — |
| `4000123` | 400 | database | `err.Error(` | **测试数据库连接失败。检查请求参数后重试。** | — |
| `4000125` | 400 | finalize, putSettings | `save settings first` | **设置尚未保存。先保存设置后继续。** | — |
| `4000126` | 400 | blobTest, database, draft 等8处 | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4000127` | 400 | getSettings | `err.Error(` | **读取设置失败。检查请求参数后重试。** | — |
| `4000128` | 400 | register | `账号名不能为空` | **账号名不能为空。填写账号名后重试。** | — |
| `4000129` | 400 | profile, register | `invalid email` | **邮箱格式不正确。检查后重试。** | — |
| `4010107` | 401 | Middleware, me, password 等4处 | `err.Error( / unauthorized` | **登录已失效。重新登录。** | 禁止（安全） |
| `4010140` | 401 | login | `invalid credentials` | **账号或密码不正确。检查后重试。** | 禁止（安全） |
| `4030108` | 403 | Middleware | `msg` | **平台尚未初始化。先完成初始化设置。** | 禁止（安全） |
| `4030141` | 403 | Middleware | `forbidden` | **没有权限。这个操作需要管理员。** | 禁止（安全） |
| `4030142` | 403 | Middleware | `settings saved; restart pixoma` | **设置已保存。重启 `pixoma` 后生效。** | 禁止（安全） |
| `4030143` | 403 | login | `account disabled` | **这个账号已停用。联系管理员启用。** | 禁止（安全） |
| `4040101` | 404 | blobTest | `bucket_not_found` | **bucket 不存在。勾选「自动创建」后重新测试。** | — |
| `4090103` | 409 | register | `registration disabled` | **自助注册已关闭。需要管理员在后台开启。** | — |
| `4090130` | 409 | register | `username or email already take` | **账号名或邮箱已被占用。更换后重试。** | — |
| `4290121` | 429 | register | `too many registrations` | **注册请求过多。等待 1 分钟后重试。** | — |
| `4290124` | 429 | login | `too many login attempts` | **登录尝试过多。等待 1 分钟后重试。** | — |
| `5000105` | 500 | register | `unavailable` | **账号服务不可用。重启 `pixoma` 后重试。** | **必须** |
| `5000106` | 500 | draft | `err.Error(` | **保存草稿失败。重新保存一次。** | **必须** |
| `5000131` | 500 | putSettings | `err.Error(` | **保存设置失败。重新保存一次。** | **必须** |
| `5000132` | 500 | profile | `err.Error(` | **保存资料失败。重新保存一次。** | **必须** |
| `5000133` | 500 | password | `err.Error(` | **修改密码失败。重新提交一次。** | **必须** |
| `5000134` | 500 | register | `create failed` | **创建账号失败。重新提交一次。** | **必须** |
| `5000135` | 500 | finalize | `err.Error(` | **初始化收尾失败。重新提交一次。** | **必须** |
| `5000136` | 500 | register | `failed to hash password` | **密码加密失败。重新提交一次。** | **必须** |
| `5000137` | 500 | register | `err.Error(` | **注册失败。重新提交一次。** | **必须** |
| `5000138` | 500 | database | `err.Error(` | **测试数据库连接失败。检查 DSN 和网络后重新测试。** | **必须** |
| `5000139` | 500 | login | `err.Error(` | **登录失败。等待 1 分钟后重试。** | **必须** |

### 系统账号（业务域 `02`，11 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000202` | 400 | updateAccess | `invalid access` | **访问权限不合法。检查后重试。** | — |
| `4000211` | 400 | updateAccess | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4000212` | 400 | list | `err.Error(` | **读取用户失败。检查请求参数后重试。** | — |
| `4030208` | 403 | RotateMCPToken | `forbidden` | **没有权限。这个操作需要管理员。** | 禁止（安全） |
| `4040201` | 404 | loadMCPToken | `mcp token not found` | **MCP token 不存在。先在用户详情里生成一个。** | — |
| `4040210` | 404 | get, loadMCPToken, updateAccess | `user not found` | **用户不存在。可能已被删除，刷新列表后重试。** | — |
| `5000206` | 500 | loadMCPToken | `mcp token not configured` | **MCP token 还没生成。先在用户详情里生成。** | **必须** |
| `5000213` | 500 | updateAccess | `err.Error(` | **更新访问权限失败。重新保存一次。** | **必须** |
| `5000214` | 500 | GetMCPToken, loadMCPToken | `err.Error(` | **读取 MCP token 失败。刷新页面后重试。** | **必须** |
| `5000215` | 500 | get, list | `err.Error(` | **读取用户失败。刷新页面后重试。** | **必须** |
| `5000216` | 500 | RotateMCPToken | `err.Error(` | **轮换 MCP token 失败。重新生成一次。** | **必须** |

### 管理员账号（业务域 `03`，17 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000302` | 400 | list | `invalid limit` | **`limit` 不合法。按照文档的取值范围填写。** | — |
| `4000310` | 400 | create | `password must be at least 8 ch` | **密码至少 8 位。增加长度后重试。** | — |
| `4000311` | 400 | patch | `invalid role` | **角色不合法。只能填写 `admin` / `operator` / `viewer`。** | — |
| `4000312` | 400 | create, patch | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4000313` | 400 | create | `账号名不能为空` | **账号名不能为空。填写账号名后重试。** | — |
| `4000314` | 400 | create, patch | `invalid email` | **邮箱格式不正确。检查后重试。** | — |
| `4040301` | 404 | writeUserErr | `user not found` | **管理员不存在。可能已被删除，刷新列表后重试。** | — |
| `4090303` | 409 | patch | `cannot remove the last admin` | **只剩这一个管理员。先添加管理员后再修改。** | — |
| `4090315` | 409 | delete | `cannot delete the last admin` | **只剩这一个管理员。先添加管理员后再删除。** | — |
| `4090316` | 409 | writeCreateError, writeUpdateError | `username or email already take` | **账号名或邮箱已被占用。更换后重试。** | — |
| `5000306` | 500 | writeCreateError | `create failed` | **创建管理员失败。重新提交一次。** | **必须** |
| `5000317` | 500 | delete | `delete failed` | **删除管理员失败。重试一次。** | **必须** |
| `5000318` | 500 | create, patch | `failed to hash password` | **密码加密失败。重新提交一次。** | **必须** |
| `5000319` | 500 | writeUpdateError | `update failed` | **更新管理员失败。重新保存一次。** | **必须** |
| `5000320` | 500 | delete, patch | `count admins failed` | **统计管理员数量失败。刷新页面后重试。** | **必须** |
| `5000321` | 500 | list | `list failed` | **读取管理员列表失败。刷新页面后重试。** | **必须** |
| `5000322` | 500 | writeUserErr | `load failed` | **读取管理员失败。刷新页面后重试。** | **必须** |

### 会话（业务域 `04`，3 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000402` | 400 | list | `err.Error(` | **读取会话失败。检查请求参数后重试。** | — |
| `4040401` | 404 | get | `session not found` | **会话不存在。可能已被删除，刷新列表后重试。** | — |
| `5000406` | 500 | get, list | `err.Error(` | **读取会话失败。刷新页面后重试。** | **必须** |

### 任务（业务域 `05`，6 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000502` | 400 | list | `err.Error(` | **读取任务失败。检查请求参数后重试。** | — |
| `4040501` | 404 | cancel, get | `task not found` | **任务不存在。可能已被删除，刷新列表后重试。** | — |
| `4090503` | 409 | cancel | `cancel not allowed` | **当前状态不能取消。等待任务运行结束后重试。** | — |
| `5000505` | 500 | cancel | `cancel not configured` | **任务取消服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5000506` | 500 | cancel | `err.Error(` | **取消任务失败。重新提交一次。** | **必须** |
| `5000510` | 500 | get, list | `err.Error(` | **读取任务失败。刷新页面后重试。** | **必须** |

### 用例（业务域 `06`，15 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000602` | 400 | create | `err.Error(` | **创建用例失败。检查请求参数后重试。** | — |
| `4000610` | 400 | patch | `err.Error(` | **更新用例失败。检查请求参数后重试。** | — |
| `4000611` | 400 | delete, disable, enable 等5处 | `invalid case id` | **用例 ID 必须是数字。检查 URL 后重试。** | — |
| `4000612` | 400 | create, patch | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4000613` | 400 | list | `err.Error(` | **读取用例失败。检查请求参数后重试。** | — |
| `4040601` | 404 | delete, disable, enable 等5处 | `case not found` | **用例不存在。可能已被删除，刷新列表后重试。** | — |
| `4090603` | 409 | create | `case already exists` | **用例已存在。修改名称，或者先删除原有的。** | — |
| `4090604` | 409 | delete | `case is referenced by menu or ` | **用例被菜单或卡片引用。勾选「同时清理引用」后删除。** | — |
| `5000605` | 500 | delete | `delete cleanup not configured` | **删除清理服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5000606` | 500 | disable | `err.Error(` | **停用用例失败。重试一次。** | **必须** |
| `5000614` | 500 | create | `err.Error(` | **创建用例失败。重新提交一次。** | **必须** |
| `5000615` | 500 | delete | `err.Error(` | **删除用例失败。重试一次。** | **必须** |
| `5000616` | 500 | enable | `err.Error(` | **启用用例失败。重试一次。** | **必须** |
| `5000617` | 500 | patch | `err.Error(` | **更新用例失败。重新保存一次。** | **必须** |
| `5000618` | 500 | get, list | `err.Error(` | **读取用例失败。刷新页面后重试。** | **必须** |

### 实例（业务域 `07`，22 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000702` | 400 | listTasks | `invalid limit` | **`limit` 不合法。按照文档的取值范围填写。** | — |
| `4000710` | 400 | listTasks | `invalid offset` | **`offset` 不合法。填写非负整数。** | — |
| `4000711` | 400 | create, patch | `name required` | **名称不能为空。填写名称后重试。** | — |
| `4000712` | 400 | patch | `unknown or disabled topic: "+k` | **话题不存在或已停用。刷新话题列表后重试。** | — |
| `4000713` | 400 | create, patch | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4000714` | 400 | metrics | `err.Error(` | **读取实例指标失败。检查请求参数后重试。** | — |
| `4040701` | 404 | delete, get, listTasks 等7处 | `instance not found` | **实例不存在。可能已被删除，刷新列表后重试。** | — |
| `4090703` | 409 | create | `instance already exists` | **实例已存在。修改名称，或者先删除原有的。** | — |
| `4090704` | 409 | delete | `edge has running tasks; confir` | **实例还有运行中的任务。勾选「同时清理引用」后删除，任务会被标记为失败。** | — |
| `5000705` | 500 | delete | `delete cleanup not configured` | **删除清理服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5000706` | 500 | writeInstance | `err.Error(` | **写入实例失败。重试一次。** | **必须** |
| `5000715` | 500 | metrics | `metrics not configured` | **指标服务未配置。检查 `metrics` 配置后重启 `pixoma`。** | **必须** |
| `5000716` | 500 | patch | `topics repository not configur` | **话题存储未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5000717` | 500 | create | `err.Error(` | **创建实例失败。重新提交一次。** | **必须** |
| `5000718` | 500 | delete | `err.Error(` | **删除实例失败。重试一次。** | **必须** |
| `5000719` | 500 | patch | `err.Error(` | **更新实例失败。重新保存一次。** | **必须** |
| `5000720` | 500 | listTasks | `err.Error(` | **读取任务列表失败。刷新页面后重试。** | **必须** |
| `5000721` | 500 | listPresence | `err.Error(` | **读取在线实例失败。刷新页面后重试。** | **必须** |
| `5000722` | 500 | get, list | `err.Error(` | **读取实例失败。刷新页面后重试。** | **必须** |
| `5000723` | 500 | metrics | `err.Error(` | **读取实例指标失败。刷新页面后重试。** | **必须** |
| `5000724` | 500 | stats | `err.Error(` | **读取统计失败。刷新页面后重试。** | **必须** |
| `5000725` | 500 | rotateToken | `err.Error(` | **轮换实例 token 失败。重新生成一次。** | **必须** |

### 渠道（业务域 `08`，20 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000802` | 400 | Create | `err.Error(` | **创建渠道失败。检查请求参数后重试。** | — |
| `4000811` | 400 | CreateMCPUser | `name required` | **名称不能为空。填写名称后重试。** | — |
| `4000812` | 400 | putMenu, save | `err.Error( / templates required` | **处理渠道失败。检查请求参数后重试。** | — |
| `4000813` | 400 | CreateMCPUser, DeleteMCPUser, ListMCPUsers | `channel is not mcp` | **渠道类型不支持。更换为 MCP 类型的渠道后重试。** | — |
| `4000814` | 400 | Create, CreateMCPUser, Update 等6处 | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4040801` | 404 | CreateMCPUser, Delete, DeleteMCPUser 等8处 | `channel not found / user not found` | **渠道不存在。可能已被删除，刷新列表后重试。** | — |
| `4100810` | 410 | goneCards | `cards are nested in the menu t` | **渠道不存在。可能已被删除，刷新列表后重试。** | — |
| `5000805` | 500 | CreateMCPUser, DeleteMCPUser, ListMCPUsers | `mcp users not configured` | **MCP 用户服务未配置。检查 `channels` 配置后重启 `pixoma`。** | **必须** |
| `5000806` | 500 | Disable | `err.Error(` | **停用渠道失败。重试一次。** | **必须** |
| `5000816` | 500 | Create, KickProbe, List | `channel service not configured` | **渠道服务未配置。检查 `channels` 配置后重启 `pixoma`。** | **必须** |
| `5000817` | 500 | Create | `err.Error(` | **创建渠道失败。重新提交一次。** | **必须** |
| `5000818` | 500 | DeleteMCPUser | `err.Error(` | **删除 MCP 用户失败。重试一次。** | **必须** |
| `5000819` | 500 | Delete | `err.Error(` | **删除渠道失败。重试一次。** | **必须** |
| `5000820` | 500 | Enable | `err.Error(` | **启用渠道失败。重试一次。** | **必须** |
| `5000821` | 500 | ListWorkflowPlacements, getMenu, putMenu 等5处 | `err.Error(` | **处理渠道失败。重试一次。** | **必须** |
| `5000822` | 500 | Update | `err.Error(` | **更新渠道失败。重新保存一次。** | **必须** |
| `5000823` | 500 | CreateMCPUser | `err.Error(` | **添加 MCP 用户失败。重新提交一次。** | **必须** |
| `5000824` | 500 | ListMCPUsers | `err.Error(` | **读取 MCP 用户失败。刷新页面后重试。** | **必须** |
| `5000825` | 500 | Get, List, list | `err.Error(` | **读取渠道失败。刷新页面后重试。** | **必须** |
| `5030815` | 503 | reset, save | `text templates unavailable` | **模板存储未配置。检查 `channels` 配置后重启 `pixoma`。** | **必须** |

### 话题（业务域 `09`，14 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4000902` | 400 | create, update | `name required` | **名称不能为空。填写名称后重试。** | — |
| `4000910` | 400 | create | `invalid topic key (lowercase l` | **话题 key 不合法。仅支持小写字母、数字和连字符。** | — |
| `4000911` | 400 | create, update | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4040901` | 404 | delete, get, stats 等4处 | `topic not found` | **话题不存在。可能已被删除，刷新列表后重试。** | — |
| `4090903` | 409 | create | `topic key already exists` | **话题 key 已存在。更换一个 key。** | — |
| `4090904` | 409 | delete | `topic is referenced by cases o` | **话题被用例或实例引用。勾选「同时清理引用」后删除。** | — |
| `4090912` | 409 | update | `default topic cannot be disabl` | **默认话题不能停用。更换其他话题后重试。** | — |
| `4090913` | 409 | delete | `default topic cannot be delete` | **默认话题不能删除。只能删除自建话题。** | — |
| `5000905` | 500 | delete | `delete cleanup not configured` | **删除清理服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5000906` | 500 | create | `err.Error(` | **创建话题失败。重新提交一次。** | **必须** |
| `5000914` | 500 | delete | `err.Error(` | **删除话题失败。重试一次。** | **必须** |
| `5000915` | 500 | update | `err.Error(` | **更新话题失败。重新保存一次。** | **必须** |
| `5000916` | 500 | stats | `err.Error(` | **读取统计失败。刷新页面后重试。** | **必须** |
| `5000917` | 500 | get, list | `err.Error(` | **读取话题失败。刷新页面后重试。** | **必须** |

### 路由（业务域 `10`，1 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `5001005` | 500 | attributes | `registry not configured` | **路由注册表未配置。检查配置后重启 `pixoma`。** | **必须** |

### 统计（业务域 `11`，11 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4001102` | 400 | errors | `invalid limit: must be 1..100` | **`limit` 超出范围。填写 1–100。** | — |
| `4001110` | 400 | casesTop | `invalid limit: must be 1..20` | **`limit` 超出范围。填写 1–20。** | — |
| `4001111` | 400 | parseRange | `from must not be after to` | **开始日期晚于结束日期。调整后重试。** | — |
| `4001112` | 400 | parseRange | `invalid from/to: expected YYYY` | **日期格式不正确。使用 YYYY-MM-DD 后重试。** | — |
| `4001113` | 400 | parseRange | `range exceeds 365 days` | **查询范围超过 365 天。缩小范围后重试。** | — |
| `5001105` | 500 | fleet | `metrics not configured` | **指标服务未配置。检查 `metrics` 配置后重启 `pixoma`。** | **必须** |
| `5001106` | 500 | edges | `err.Error(` | **读取实例统计失败。刷新页面后重试。** | **必须** |
| `5001114` | 500 | daily | `err.Error(` | **读取每日统计失败。刷新页面后重试。** | **必须** |
| `5001115` | 500 | casesTop | `err.Error(` | **读取用例排行失败。刷新页面后重试。** | **必须** |
| `5001116` | 500 | errors | `err.Error(` | **读取错误统计失败。刷新页面后重试。** | **必须** |
| `5001117` | 500 | fleet | `err.Error(` | **读取集群统计失败。刷新页面后重试。** | **必须** |

### 工作台（业务域 `12`，16 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4001202` | 400 | streamAGUI | `会话、运行和用户消息不能为空` | **会话、运行、消息都不能为空。补充完整后重试。** | — |
| `4001212` | 400 | uploadAsset | `请选择上传文件` | **尚未选择文件。选择文件后重新上传。** | — |
| `4001213` | 400 | createConnector, createFlowEdge, createFlowNode 等19处 | `请求内容格式不正确` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4001214` | 400 | writeError（分类路由器） | `domain.ErrInvalid / ErrInvalid` | **请求内容不合法。检查后重新提交。** | **必须** |
| `4001215` | 400 | uploadAsset | `上传文件读取失败` | **读取上传文件失败。重新选择文件后上传。** | — |
| `4011207` | 401 | accountID | `登录状态已失效` | **登录已失效。重新登录。** | 禁止（安全） |
| `4041201` | 404 | writeError（分类路由器） | `domain.ErrNotFound` | **内容不存在或无权访问。刷新列表后重试。** | 省略（防探测） |
| `4041210` | 404 | assetContent | `资产内容不存在` | **资产内容不存在。刷新素材库后重试。** | — |
| `4041211` | 404 | assetContent | `资产版本不存在` | **这个资产版本不存在。刷新后选择其他版本。** | — |
| `4091203` | 409 | writeError（分类路由器） | `domain.ErrAlreadyExists` | **内容已存在。修改名称，或者先删除原有的。** | 省略 |
| `5001205` | 500 | writeError（分类路由器） | `默认分支（服务暂时不可用）` | **工作台服务不可用。重启 `pixoma` 后重试。** | **必须** |
| `5001206` | 500 | streamAGUI | `当前连接不支持流式响应` | **当前连接不支持流式响应。更换为支持 SSE 的环境。** | **必须** |
| `5021219` | 502 | writeError（分类路由器） | `studioapp.ErrModelConnectionTe` | **模型连接测试失败。检查 Base URL 和 API Key 后重新测试。** | **必须** |
| `5031216` | 503 | createModel, testModelConfig, testModelConnection 等4处 | `模型配置服务不可用` | **模型配置服务不可用。检查模型配置后重启 `pixoma`。** | **必须** |
| `5031217` | 503 | createConnector, createSkill, listAgentWorkflows 等9处 | `能力配置服务不可用` | **能力配置服务不可用。检查能力配置后重启 `pixoma`。** | **必须** |
| `5031218` | 503 | createTextAsset, updateTextAsset, uploadAsset | `资产存储服务不可用` | **资产存储不可用。检查对象存储配置后重启 `pixoma`。** | **必须** |

### 媒体（业务域 `13`，7 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4001302` | 400 | Upload | `缺少上传文件` | **尚未选择文件。选择文件后重新上传。** | — |
| `4001310` | 400 | Upload | `不支持的文件类型，仅支持图片（png/jpeg/webp/g` | **文件类型不支持。仅支持 png / jpeg / webp / gif 图片以及 mp4 / webm 视频。** | — |
| `4001312` | 400 | Upload | `请求格式错误：需 multipart/form-data` | **请求格式不支持。使用 multipart/form-data 重新上传。** | — |
| `4001313` | 400 | Upload | `读取上传文件失败` | **读取上传文件失败。重新选择文件后上传。** | — |
| `4041301` | 404 | ServePreview | `媒体不存在` | **媒体不存在。可能已被删除，刷新后重试。** | — |
| `4131311` | 413 | Upload | `文件超过大小上限` | **文件超过大小上限。压缩后重新上传。** | — |
| `5001306` | 500 | Upload | `媒体保存失败` | **保存媒体失败。重新上传一次。** | **必须** |

### 链路健康（业务域 `14`，2 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `5001405` | 500 | Get | `link health not configured` | **链路健康服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5001406` | 500 | Get | `err.Error(` | **读取链路健康失败。刷新页面后重试。** | **必须** |

### 边缘 Agent（业务域 `15`，14 条）

| code | HTTP | 触发点 | 现有文案 | **最终 message** | `error_detail` |
| --- | --- | --- | --- | --- | --- |
| `4001502` | 400 | claim, heartbeat, presence 等4处 | `edge_id required` | **`edge_id` 不能为空。携带实例 ID 后重试。** | — |
| `4001510` | 400 | presence | `invalid json` | **请求体格式不正确。检查上报字段的格式。** | — |
| `4001511` | 400 | heartbeat, status | `invalid json` | **请求体格式不正确。检查字段格式后重试。** | — |
| `4011507` | 401 | claim, heartbeat, presence 等4处 | `unauthorized` | **凭证无效。检查实例的 `agent token`。** | 禁止（安全） |
| `4081512` | 408 | claim | `canceled` | **请求被取消。重新领取任务。** | — |
| `4091503` | 409 | status | `err.Error(` | **上报任务状态失败。刷新后重试。** | — |
| `4091513` | 409 | status | `task not found` | **任务不存在。可能已被回收，重新领取任务。** | — |
| `4091514` | 409 | status | `stale holder` | **任务持有者已过期。重新领取任务后上报。** | — |
| `4091515` | 409 | heartbeat | `heartbeat rejected` | **心跳被拒绝。任务可能已被回收，重新领取任务。** | — |
| `5001505` | 500 | status | `status not configured` | **任务状态服务未配置。检查配置后重启 `pixoma`。** | **必须** |
| `5001506` | 500 | presence | `err.Error(` | **上报在线状态失败。稍后重试。** | **必须** |
| `5001516` | 500 | presence | `presence not configured` | **在线状态服务未配置。检查 `presence` 配置后重启 `pixoma`。** | **必须** |
| `5001517` | 500 | heartbeat | `err.Error(` | **上报心跳失败。稍后重试。** | **必须** |
| `5001518` | 500 | claim | `err.Error(` | **领取任务失败。稍后重试。** | **必须** |

## 4. i18n key 规范

每个 code 一个词条：`apiError.<code>`。

例：`apiError.4090604` → `"用例被菜单或卡片引用。勾选「同时清理引用」后删除。"`

- 后端 `apierr` 注册表 `I18nKey` 填该 key。
- 前端 `web/admin/src/lib/api/error-messages.ts` 建 `code → key` 映射。
- 词条写入 `web/admin/src/lib/i18n/locales/zh.json` 与 `en.json`。
- `apierr` 测试强制校验：每个 code 的 key 在 zh/en 两份词条中都存在。

## 5. 覆盖率

| 指标 | 数量 |
| --- | --- |
| 错误返回点（真实调用点） | 438 |
| **最终 i18n 词条（域 + HTTP + 文案）** | **204** |
| 业务域 | 16 |
| 明细码最大值 | 43（上限 99，余量充足） |

## 6. 复核重点

1. **动作句可执行性**：`重试一次` / `刷新页面后重试` 是通用兜底；有更具体的界面动作时优先替换。
2. **配置项键名**：`channels` / `metrics` / `presence` 需与 `configs/` 实际一致。
3. **`error_detail` 禁止项**：401 / 403 共 11 条，确认不含资源存在性信息。
4. **英文词条**：`en.json` 按同一事实+动作结构翻译，不做逐字直译。
