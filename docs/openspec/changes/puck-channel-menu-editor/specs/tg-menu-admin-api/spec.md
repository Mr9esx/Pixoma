## MODIFIED Requirements

### Requirement: Menu 树管理 HTTP 接口
admin-api MUST 暴露消息平台菜单管理接口（路径约定 `/api/v1/channels/{id}/menu`）：支持按消息平台获取完整嵌套树，以及整棵树写回。未知动作类型 MUST 被拒绝。接口鉴权策略与现有 admin-api 一致（仅内网约定）。

#### Scenario: 获取 Menu 树
- **WHEN** 客户端 GET `/api/v1/channels/{id}/menu` 且服务可用
- **THEN** 返回 200 与当前嵌套树 JSON（含键盘、内嵌卡片与动作）

#### Scenario: 更新 Menu 树
- **WHEN** 客户端提交合法菜单树进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填、未知动作类型、非法动作参数、无效工作流/Case id、空发媒体或空按钮文案的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交发媒体动作且媒体列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

### Requirement: Case 菜单挂载查询接口
admin-api MUST 暴露按 Case id 查询菜单挂载的接口（路径约定 `/api/v1/cases/{id}/menu-placements`），返回该 Case 出现的消息平台与从根键盘起的路径列表。

#### Scenario: 查询已挂载 Case
- **WHEN** 客户端 GET 某已挂到菜单的 Case 的 placements
- **THEN** 返回 200 与至少一条可读路径（含消息平台标识）

#### Scenario: 未挂载 Case
- **WHEN** 客户端 GET 未出现在任何菜单动作中的 Case 的 placements
- **THEN** 返回 200 与空列表

### Requirement: 仅经 admin-api 管理
菜单的管理写路径 MUST 仅通过 admin-api；Bot 进程 MUST NOT 再依赖改源码作为常规配置手段（默认种子除外）。

#### Scenario: 控制台经 admin-api 读写
- **WHEN** 管理前端在消息平台详情中保存菜单或加载 Case 挂载
- **THEN** 请求指向 admin-api 的消息平台菜单路径，而非 bot 管理残留路径

## ADDED Requirements

### Requirement: 不再提供独立卡片集合接口
admin-api MUST NOT 再把卡片当作渠道下可独立 CRUD 的资源。卡片只作为菜单树节点随 `/menu` 整份读写。原 `/api/v1/channels/{id}/cards` 及其引用查询 MUST 停止作为管理契约（返回 404 或明确 410，且 MUST NOT 再写入 `channel_cards`）。

#### Scenario: 旧卡片列表接口不可用
- **WHEN** 客户端请求原卡片列表或创建卡片路径
- **THEN** 系统不创建或更新独立卡片资源，调用方无法再按旧契约管理卡片
