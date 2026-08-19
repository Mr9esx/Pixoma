## MODIFIED Requirements

### Requirement: User 持久化与主键
系统 MUST 将用户记录持久化到数据库。每条记录 MUST 使用内部 UUID（或等价内部 id）作为主键，并 MUST 对**渠道外部身份**（渠道 + 渠道外部用户 id）建立唯一约束；Telegram 用户 id 作为 `telegram` 渠道的外部身份。系统 MUST NOT 仅用 `chat_id` 充当唯一用户主键。

#### Scenario: 同一 tg_user_id 不重复建档
- **WHEN** 同一渠道同一外部用户 id 再次触发 upsert
- **THEN** 仍对应同一条内部用户记录（id 不变）

### Requirement: TG 资料字段与 upsert
系统 MUST 在处理渠道消息或回调查询的路径上，根据平台来源（如 Telegram `From`）按（渠道 + 外部 id）upsert 用户资料。Telegram 渠道存储字段 MUST 覆盖 From 可获得的常用身份信息，至少包括：`username`、`first_name`、`last_name`、`language_code`、`is_bot`、`is_premium`（若 API 提供；不可得则可空）。系统 MUST 维护 `last_seen_at`（或等价），并在 upsert 时刷新可变资料字段。其他渠道 MAY 存储其平台提供的等价资料字段。

#### Scenario: 消息到达后可查到用户资料
- **WHEN** 用户（telegram 渠道）首次向 bot 发消息且 From 含 username 与 first_name
- **THEN** 数据库中存在对应渠道外部身份的用户行，且上述字段已写入

#### Scenario: 资料变更被刷新
- **WHEN** 同一用户稍后消息中 username 已变更
- **THEN** upsert 后该用户行的 username 为新值，且 last_seen_at 更新
