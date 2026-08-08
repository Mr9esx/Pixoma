## Purpose

为运维排障提供 Session 只读管理查询，避免通过 admin 破坏对话采集态机。

## ADDED Requirements

### Requirement: Session 列表与详情
系统 MUST 通过 admin-api 提供 Session 列表与详情查询。

#### Scenario: 列出 Session
- **WHEN** 客户端请求 Session 列表
- **THEN** 返回 Session 集合及状态等关键字段

#### Scenario: 按过滤条件列出 Session
- **WHEN** 客户端请求 Session 列表并携带 `q`、时间范围或 `user_id`/`chat_id`/`status` 等已支持过滤参数
- **THEN** 仅返回匹配条件的 Session，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Session 详情
- **WHEN** 客户端请求已存在 Session 的详情
- **THEN** 返回含草稿/进度等排障所需字段的表示

### Requirement: 不以完整 CRUD 破坏态机
本期 Session 管理 MUST NOT 提供任意改写对话态机关键字段的通用 Update/Delete（除非显式定义的安全运维动作）；默认以只读查询交付。

#### Scenario: 默认无通用写接口
- **WHEN** 客户端尝试对 Session 做未定义的通用更新
- **THEN** 系统不提供该通用写能力或返回方法不允许
