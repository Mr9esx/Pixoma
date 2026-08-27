# admin-user-management Specification

## Purpose
TBD - created by archiving change user-rbac. Update Purpose after archive.
## Requirements
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
