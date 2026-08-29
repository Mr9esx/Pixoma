# tg-menu-admin-api Specification

## Purpose
为管理端提供 Telegram 主键盘（树形 Menu）配置的 HTTP API，并支持按 Case 反查菜单挂载路径。
## Requirements
### Requirement: Menu 树管理 HTTP 接口
admin-api MUST 暴露消息平台菜单管理接口（路径约定 `/api/v1/channels/{id}/menu`）：支持按消息平台获取完整树形配置，以及整棵树写回。菜单载荷使用能力入口结构（capability_id + params + 展示字段）；未知 capability_id MUST 被拒绝。接口鉴权策略与现有 admin-api 一致（仅内网约定）。

#### Scenario: 获取 Menu 树
- **WHEN** 客户端 GET `/api/v1/channels/{id}/menu` 且服务可用
- **THEN** 返回 200 与当前树形菜单 JSON（含能力入口与展示字段）

#### Scenario: 更新 Menu 树
- **WHEN** 客户端提交合法菜单树进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填、未知 capability_id、非法能力参数、无效 `case_id`、空 reply_media 或违反同层 label 唯一等规则的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交回复媒体展示字段且图片列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

### Requirement: Case 菜单挂载查询接口
admin-api MUST 暴露按 Case id 查询菜单挂载的接口（路径约定 `/api/v1/cases/{id}/menu-placements`），返回该 Case 出现的消息平台与菜单路径列表。

#### Scenario: 查询已挂载 Case
- **WHEN** 客户端 GET 某已挂到菜单的 Case 的 placements
- **THEN** 返回 200 与至少一条可读路径（含消息平台标识）

#### Scenario: 未挂载 Case
- **WHEN** 客户端 GET 未出现在任何菜单项关联中的 Case 的 placements
- **THEN** 返回 200 与空列表

### Requirement: 仅经 admin-api 管理
菜单的管理写路径 MUST 仅通过 admin-api；Bot 进程 MUST NOT 再依赖改源码作为常规配置手段（默认种子除外）。

#### Scenario: 控制台经 admin-api 读写
- **WHEN** 管理前端在消息平台详情中保存菜单或加载 Case 挂载
- **THEN** 请求指向 admin-api 的消息平台菜单路径，而非 bot 管理残留路径

