## ADDED Requirements

### Requirement: Telegram 外部用户是操作者
在 Telegram 上，`external_user_id` MUST 为消息或回调查询的操作者 user id。群 chat id MUST NOT 作为 `external_user_id`。能力上下文仍携带投递用的聊天地址（群或私聊），以便结果发回正确会话。

#### Scenario: 群成员账户按人区分
- **WHEN** 同一群内两个成员分别触发能力
- **THEN** 两条能力调用的 external_user_id 不同，内部用户记录不合并成「这个群」

#### Scenario: 投递地址仍是群
- **WHEN** 群成员触发的任务需要发终态图
- **THEN** 投递目标是该群的聊天地址，账户仍是该成员
