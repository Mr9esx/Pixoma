# user-admin-api Specification

## Purpose
为管理后台提供用户资源的查询 HTTP，支撑运维查看 TG 关联用户。
## Requirements
### Requirement: 用户列表与详情
系统 MUST 通过 admin-api 提供用户列表与按内部 ID 详情查询。

#### Scenario: 列出用户
- **WHEN** 客户端请求用户列表
- **THEN** 返回用户集合（含可观察标识字段，如内部 id、tg 用户相关字段）

#### Scenario: 按过滤条件列出用户
- **WHEN** 客户端请求用户列表并携带 `q`、时间范围或 `tg_user_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的用户，并支持 `limit`/`offset` 分页

#### Scenario: 获取用户详情
- **WHEN** 客户端请求已存在用户的详情
- **THEN** 返回该用户记录

#### Scenario: 用户不存在
- **WHEN** 客户端请求不存在的用户 ID
- **THEN** 返回未找到错误

### Requirement: 不替代 TG upsert 主路径
admin-api MUST NOT 成为用户创建的主业务入口；用户仍主要由 bot/TG 路径 upsert。本期 MUST NOT 提供用户写接口。

#### Scenario: 管理面本期只读
- **WHEN** 运维使用本期用户管理 API
- **THEN** 可完成列表与详情，且不存在创建/更新/删除用户的管理写入口

