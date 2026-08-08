## Purpose

在独立 admin-api 上提供 Comfy 实例的管理 CRUD 与 system/queue/tasks 观测 HTTP，语义与迁出前对齐。

## ADDED Requirements

### Requirement: 实例列表与详情
系统 MUST 通过 admin-api 提供 Comfy 实例的列表与按 ID 查询能力。

#### Scenario: 列出实例
- **WHEN** 客户端请求实例列表
- **THEN** 返回当前持久化中的实例集合（含启用状态等关键字段）

#### Scenario: 获取单个实例
- **WHEN** 客户端请求已存在实例的详情
- **THEN** 返回该实例记录

#### Scenario: 实例不存在
- **WHEN** 客户端请求不存在的实例 ID
- **THEN** 返回表示未找到的错误响应

### Requirement: 实例创建与更新
系统 MUST 允许通过 admin-api 创建与更新实例（含 base URL、启用状态、能力等字段），变更 MUST 持久化。

#### Scenario: 创建实例
- **WHEN** 客户端提交合法的新实例载荷
- **THEN** 实例被持久化并可在后续列表/详情中查到

#### Scenario: 更新实例
- **WHEN** 客户端提交已存在实例的合法更新
- **THEN** 持久化记录反映更新后的字段

### Requirement: 实例删除或停用
系统 MUST 支持通过 admin-api 删除或停用实例，使该实例不再作为可用管理目标（停用语义须在响应/列表中可观察）。

#### Scenario: 停用或删除后不可再作为健康调度目标
- **WHEN** 运维通过 admin-api 停用或删除某实例
- **THEN** 后续列表/详情反映该变更，且不得再将该实例表现为可用启用实例（除非再次启用）

### Requirement: 实例观测 system/queue/tasks
系统 MUST 在 admin-api 上按实例提供 system、queue 与本系统 tasks 观测端点，行为与迁出前 bot 挂载时对齐（含 mock 模式下的可观察标记，若启用 mock）。

#### Scenario: 查询 system
- **WHEN** 客户端请求某已存在实例的 system 观测
- **THEN** 返回该实例的 system 观测载荷或明确的错误

#### Scenario: 查询 queue
- **WHEN** 客户端请求某已存在实例的 queue 观测
- **THEN** 返回该实例的 queue 观测载荷或明确的错误

#### Scenario: 按实例列出本系统 tasks
- **WHEN** 客户端请求某已存在实例关联的本系统 tasks
- **THEN** 返回与该实例关联的任务列表（可为空）

### Requirement: 管理端变更对 bot 调度池最终可见
当 admin-api 与 bot 共用同一持久化时，经 admin-api 持久化的实例变更 MUST 在 bot 的约定刷新周期内对 bot 调度用实例视图可见。系统 MUST NOT 要求该可见性为瞬时；刷新周期与 bot 健康探活间隔对齐（可配置）。

#### Scenario: 一个探活周期内 bot 可见新名单
- **WHEN** 运维通过 admin-api 创建或更新某启用实例并写入持久化
- **AND** bot 与 admin-api 使用同一数据库
- **AND** 已等待至多一个 bot 健康探活间隔
- **THEN** bot 的调度用实例视图 MUST 能观察到该变更（例如新启用实例可进入健康候选，或停用实例不再表现为可用启用实例）
