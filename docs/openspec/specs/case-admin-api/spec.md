# case-admin-api Specification

## Purpose
为管理后台提供 Case（工作流目录）资源的 HTTP 管理能力，含列表、详情与写操作。
## Requirements
### Requirement: Case 列表与详情
系统 MUST 通过 admin-api 提供 Case 列表与按 ID 详情查询。

#### Scenario: 列出 Case
- **WHEN** 客户端请求 Case 列表（可带过滤参数）
- **THEN** 返回匹配的 Case 集合及启用状态等关键字段

#### Scenario: 按过滤条件列出 Case
- **WHEN** 客户端请求 Case 列表并携带 `q`、时间范围或 `enabled`/`category`/`tag` 等已支持过滤参数
- **THEN** 仅返回匹配条件的 Case，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Case 详情
- **WHEN** 客户端请求已存在 Case 的详情
- **THEN** 返回可用于管理编辑的 Case 表示（含协议/输入相关必要字段）

### Requirement: Case 创建与更新
系统 MUST 允许通过 admin-api 创建与更新 Case，并持久化；非法协议 MUST 被拒绝。

#### Scenario: 创建合法 Case
- **WHEN** 客户端提交通过校验的 Case 载荷
- **THEN** Case 被持久化且可在列表/详情中查到

#### Scenario: 更新 Case
- **WHEN** 客户端提交已存在 Case 的合法更新
- **THEN** 持久化反映更新内容

#### Scenario: 非法 Case 被拒绝
- **WHEN** 客户端提交未通过校验的 Case 载荷
- **THEN** 返回明确的校验错误且不写入无效数据

### Requirement: Case 禁用与重新启用
系统 MUST 支持禁用/下架 Case，使 bot 菜单主路径不再将其作为可用入口；亦 MUST 支持重新启用（与 catalog 启用语义对齐）。

#### Scenario: 禁用后列表可观察
- **WHEN** 运维禁用某 Case
- **THEN** 后续列表/详情反映禁用状态

#### Scenario: 重新启用后可观察
- **WHEN** 运维对已禁用 Case 发起重新启用
- **THEN** 后续列表/详情反映启用状态，且可作为可用入口（在启用语义下）

### Requirement: Case 路由投放配置读写
Case 创建/更新 API MUST 接受可选的路由投放配置字段，并按 `topic-routing-config` 校验；非法配置 MUST 被拒绝并返回可定位错误。Case 详情接口 MUST 返回路由配置。

#### Scenario: 保存合法路由配置
- **WHEN** 更新 Case 携带引用已启用 Topic 的路由规则
- **THEN** 更新成功且详情返回该路由配置

#### Scenario: 非法路由配置被拒绝
- **WHEN** 更新 Case 携带引用不存在 Topic 的路由规则
- **THEN** 更新被拒绝，返回含 Topic 相关信息的错误

