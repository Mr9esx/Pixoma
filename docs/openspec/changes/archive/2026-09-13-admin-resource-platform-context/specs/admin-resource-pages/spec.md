## MODIFIED Requirements

### Requirement: User / Session / Task 运维页
控制台 MUST 提供 User、Session、Task 的列表与详情页；Task 页 MUST 仅对可取消状态（pending / queued）提供取消动作。三类详情 Modal MUST 使用与计算节点新建 Modal 一致的宽度与内边距。Task 列表 MUST 展示消息平台、关联用户和关联 Session；Session 列表 MUST 展示消息平台；User 列表 MUST 展示消息平台和统一的「用户信息」列，且 MUST NOT 展示 Telegram 专用 ID 列或专用筛选字段。

#### Scenario: 查看用户与会话详情
- **WHEN** 运维从列表进入某 User 或 Session 详情
- **THEN** 页面展示 admin-api 返回的关键字段、消息平台来源和用户信息

#### Scenario: 识别任务上下文
- **WHEN** 运维在 Task 列表查看某条任务
- **THEN** 页面展示该任务的消息平台、关联用户和关联 Session，且这些字段可读且可直接用于排障

#### Scenario: 识别会话来源
- **WHEN** 运维在 Session 列表查看某条会话
- **THEN** 页面展示该会话的消息平台来源

#### Scenario: 用户列表不绑定单一消息平台
- **WHEN** 运维在 User 列表查看或搜索用户
- **THEN** 页面展示消息平台和统一的用户信息列，单一搜索框覆盖内部 ID、通用外部用户标识和用户资料，不出现 Telegram 专用 ID 字段

#### Scenario: 从 UI 取消任务
- **WHEN** 运维在 Task 详情对 pending 或 queued 任务执行取消
- **THEN** UI 反映取消结果或明确错误

#### Scenario: 统一详情容器
- **WHEN** 运维打开 Task、Session 或 User 详情
- **THEN** Modal 宽度与内边距沿用同一套资源详情规则，并与计算节点新建 Modal 对齐
