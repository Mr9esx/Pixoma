## Purpose

为管理端提供 Telegram 主菜单配置的 HTTP API，使运维可在 admin-api 上查询与更新 Menu，无需改代码发版。

## ADDED Requirements

### Requirement: Menu 管理 HTTP 接口
admin-api MUST 暴露 Menu 管理接口（路径约定在设计中固定，例如 `/api/v1/tg-menu`）：至少支持获取当前完整配置，以及用完整配置或等价更新语义写回。接口 MUST 无鉴权策略与现有 admin-api 一期一致（仅内网使用约定不变）。

#### Scenario: 获取 Menu
- **WHEN** 客户端 GET Menu 配置且服务可用
- **THEN** 返回 200 与当前菜单项列表 JSON

#### Scenario: 更新 Menu
- **WHEN** 客户端提交合法 Menu 配置进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填字段、非法动作或无效 `case_id` 的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交回复媒体项且图片列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

### Requirement: 仅经 admin-api 管理
Menu 的管理写路径 MUST 仅通过 admin-api 提供；Bot 进程 MUST NOT 再依赖改源码作为常规配置手段（默认种子除外）。

#### Scenario: 控制台经 admin-api 读写
- **WHEN** 管理前端保存 Menu
- **THEN** 请求指向 admin-api 的 Menu 接口，而非 bot `:8080` 管理残留路径
