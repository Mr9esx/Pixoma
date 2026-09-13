## ADDED Requirements

### Requirement: 飞书消息平台创建与凭证
管理员 MUST 能选择平台类型 `feishu` 并提交 App ID 与 App Secret（不得只填单一 TG Bot Token）。凭证 MUST 安全存储、不在列表明文回显。启用后探测 MUST 验证飞书应用身份，MUST NOT 调用 Telegram getMe。

#### Scenario: 后台创建飞书
- **WHEN** 管理员在创建表单选择飞书并填写 App ID 与 App Secret
- **THEN** 系统保存该消息平台，列表显示平台为飞书，详情不回显 Secret 明文

#### Scenario: 飞书探测不打 Telegram
- **WHEN** 系统对已启用飞书消息平台做可达性探测
- **THEN** 探测请求发往飞书开放平台身份接口，不访问 api.telegram.org

### Requirement: 飞书启停热生效
飞书消息平台启用 MUST 在数秒内挂上长连接；停用或删凭证 MUST 断开该连接。凭证轮换 MUST 先停旧连接再连新连接。

#### Scenario: 启用后长连接在线
- **WHEN** 管理员启用凭证完整的飞书消息平台
- **THEN** 无需重启进程，数秒内可收到飞书事件

#### Scenario: 停用后不再收事件
- **WHEN** 管理员停用该飞书消息平台
- **THEN** 长连接断开，后续飞书事件不再被本进程处理
