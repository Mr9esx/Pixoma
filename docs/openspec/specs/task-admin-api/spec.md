# task-admin-api Specification

## Purpose
为运营侧提供 Task 列表、详情与取消等管理 HTTP，不在 admin 替代用户 ConfirmRun。
## Requirements
### Requirement: Task 列表与详情
系统 MUST 通过 admin-api 提供 Task 列表与按 ID 详情查询。

#### Scenario: 列出 Task
- **WHEN** 客户端请求 Task 列表
- **THEN** 返回任务集合及状态、关联 session/instance 等关键字段

#### Scenario: 按过滤条件列出 Task
- **WHEN** 客户端请求 Task 列表并携带 `q`、时间范围或 `status`/`instance_id`/`chat_id`/`session_id`/`case_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的任务，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Task 详情
- **WHEN** 客户端请求已存在 Task 的详情
- **THEN** 返回该任务记录

### Requirement: Task 取消
系统 MUST 支持运营侧通过 `POST /api/v1/tasks/{id}/cancel`（或等价动作路径）取消仍可取消的任务，并反映取消后状态；取消语义 MUST 复用 runtime 领域取消规则。

#### Scenario: 取消可取消任务
- **WHEN** 运维对可取消状态的任务发起取消
- **THEN** 任务进入取消相关终态或约定的取消中状态，且详情可观察

#### Scenario: 不可取消时明确失败
- **WHEN** 运维对不可取消状态的任务发起取消
- **THEN** 返回明确错误且不伪造成功

### Requirement: 不在 admin 发起代用户 ConfirmRun
admin-api MUST NOT 提供“代替用户确认生成并创建任务”的管理入口作为本期能力。通过 MCP 调用工作流 MUST 走独立入口，MUST NOT 实现为 admin-api 上的 ConfirmRun。

#### Scenario: 无 ConfirmRun 管理入口
- **WHEN** 客户端查找代用户 ConfirmRun 的管理 API
- **THEN** 本期不提供该能力

#### Scenario: MCP 不是 admin ConfirmRun
- **WHEN** MCP 客户端成功调用工作流
- **THEN** 该调用不经过 admin-api 的 ConfirmRun 管理入口，管理 API 契约保持无代跑

