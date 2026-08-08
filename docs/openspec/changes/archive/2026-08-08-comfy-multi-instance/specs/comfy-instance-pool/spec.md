## Purpose

在同一 bot 进程内将多台远程 ComfyUI 的实例元数据持久化到数据库，提供增删改查，并结合健康探测与进程内 HTTP 客户端池供调度选路；观测面分接口查询 Comfy system/queue，以及按实例关联的本系统 Task。

## ADDED Requirements

### Requirement: 实例元数据持久化
系统 MUST 将 Comfy 实例记录持久化到数据库。每条记录 MUST 至少包含：稳定 `id`、`base_url`、是否启用（enabled）。可选字段可包含能力标签（capabilities）。删除或禁用后 MUST 不再作为默认可调度目标（进行中的已派发任务按既有失败/对账路径处理）。

#### Scenario: 重启后实例仍在
- **WHEN** 管理员已创建实例 `gpu-1` 并成功写入数据库后进程重启
- **THEN** 系统仍能按 id 读出该实例及其 base_url

### Requirement: 实例增删改查 API
系统 MUST 在 bot 进程的 HTTP 服务上提供实例的创建、查询（单个与列表）、更新、删除（或等价禁用）接口。写操作成功后 MUST 反映到后续列表查询与调度所用的注册视图（允许短暂传播延迟，但 MUST 在同一进程内可预期地刷新客户端池）。未认证的公网暴露风险本期可用本地绑定/文档警示；完整鉴权可后续增强，但 API 形状 MUST 稳定可测。

#### Scenario: 创建后可列表查出
- **WHEN** 调用创建接口写入 id=`gpu-2`、合法 base_url
- **THEN** 列表接口返回包含该实例的记录

#### Scenario: 更新 base_url
- **WHEN** 对已存在实例更新 base_url
- **THEN** 后续健康探测与对该 InstanceID 的 Comfy 调用使用新地址

#### Scenario: 删除或禁用后不再被调度
- **WHEN** 实例被删除或标记为 disabled
- **THEN** 调度的健康可选集合不再包含该实例

### Requirement: 实例系统概况只读查询
系统 MUST 提供 `GET`（或等价）按实例 id 查询系统概况的接口。数据 MUST 来自对该实例 Comfy 的 `GET /system_stats`（或文档约定的等价端点）。响应 MUST 能表达连通性，并在可达时包含系统/设备资源摘要（如 RAM/VRAM、版本信息等 Comfy 返回的关键字段）。不可达时 MUST 明确失败或 `reachable=false`，MUST NOT 伪造成功空概况。本期 MUST NOT 经此接口执行写操作。

#### Scenario: 可达时返回 system 摘要
- **WHEN** 实例可达
- **THEN** 接口返回可达标志及源自 system_stats 的摘要字段

#### Scenario: 不可达
- **WHEN** 目标实例网络失败或超时
- **THEN** 接口明确表示不可达（或非 2xx 业务错误）

### Requirement: 实例队列只读查询
系统 MUST 提供独立于系统概况的按实例 id 队列只读接口。数据 MUST 来自对该实例 Comfy 的 `GET /queue`（或文档约定的等价只读端点），并 MUST 暴露 running 与 pending 概况。MUST NOT 用 Comfy `/history` 充当本系统任务历史，MUST NOT 把业务 Task 列表混进该响应作为唯一任务来源。不可达时规则与系统概况接口相同。本期 MUST NOT 经此接口清队列或 interrupt。

#### Scenario: 可达时返回队列概况
- **WHEN** 实例可达且队列中有 running 或 pending
- **THEN** 接口返回 running/pending 概况

#### Scenario: 队列接口与 system 分离
- **WHEN** 调用方只请求队列接口
- **THEN** 不强制同时返回完整 system_stats 载荷

### Requirement: 按实例查询本系统任务
系统 MUST 提供按实例 id 列出本系统 Task 的只读接口。查询 MUST 以数据库中 Task 的 `instance_id` 关联实例，MUST NOT 依赖拉取 Comfy `/history` 作为该列表数据源。接口 SHOULD 支持分页与按任务状态过滤，并返回足以运维对照的字段（如 task id、status、prompt_id、时间戳、错误摘要）。尚未派发、因而没有 `instance_id` 的 pending 任务 MUST NOT 被错误归入某实例的该列表。

#### Scenario: 已派发任务可按实例查出
- **WHEN** 任务已调度到实例 `gpu-1`（Task.InstanceID 已写入）
- **THEN** 对该实例的任务列表接口能返回该任务

#### Scenario: 不混入未关联实例的 pending
- **WHEN** 存在尚未选定实例的 pending 任务
- **THEN** 这些任务不出现在任意实例的按实例任务列表中

### Requirement: 配置种子与 Mock 兼容
系统 MUST 允许启动时从配置（如可选 `comfy_instances` 或单一 `comfyui_base_url`）**upsert** 种子实例到数据库。`comfy_mock` 为真时 MUST 仍提供可用的进程内 Mock 客户端路径；system/queue 查询对 Mock MUST 返回稳定可测占位（标明 mock），不得因缺远程而崩溃。

#### Scenario: 空库用单 URL 种子
- **WHEN** 数据库中尚无实例，且配置了 `comfyui_base_url`，`comfy_mock` 为假
- **THEN** 启动后存在默认实例记录，指向该 URL

#### Scenario: Mock 模式查询 system/queue
- **WHEN** `comfy_mock` 为真并查询默认实例的 system 或 queue
- **THEN** 返回标明 mock 的占位，进程不崩溃

### Requirement: 健康检查过滤不可用实例
系统 MUST 对**已启用**的真实 HTTP 实例做周期性健康探测（例如请求 Comfy 的系统状态端点）。探测失败或超时的实例 MUST 不出现在「健康可选」集合中。探测间隔与超时 MUST 可配置并设有安全默认值。

#### Scenario: 一台宕机被剔除
- **WHEN** 启用实例 `a` 探测失败，启用实例 `b` 探测成功
- **THEN** 健康可选集合仅包含 `b`（及仍健康的其他启用实例）

#### Scenario: 恢复后重新可选
- **WHEN** 曾不健康的启用实例随后探测成功
- **THEN** 该实例重新进入健康可选集合

### Requirement: 进程内客户端按实例选用
系统 MUST 在同一进程内按实例 id 获取对应 Comfy 客户端（随 CRUD/种子变化刷新）。执行面 MUST 使用 dispatch 携带的 `InstanceID` 选择客户端。

#### Scenario: 派发到指定实例后调用其 API
- **WHEN** dispatch 的 InstanceID 为 `gpu-2`
- **THEN** Upload/Submit/Wait 均针对 `gpu-2` 当前 base_url 发出
