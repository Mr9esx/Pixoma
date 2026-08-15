# shared-blob-store Specification

## Purpose
TBD - created by archiving change edge-agent-topic-routing. Update Purpose after archive.
## Requirements
### Requirement: Blob 双驱动（localfs 与 S3 兼容）
系统 MUST 提供可配置的 `blob.Store` 后端：至少支持 localfs、S3 兼容实现，以及火山引擎 TOS。本机部署默认使用 localfs（控制面与 Edge 共用目录）；远程跨机主路径 MUST 使用 `s3` 或 `tos`，使得控制面 Put 的对象可被 Edge Get，Edge Put 的产物可被控制面通知路径 Get。逻辑 key 前缀（`inputs/`、`jobs/`、`outputs/`）MUST 一致。MUST NOT 再以 `runtime_mode=allinone|split` 作为 blob 选型文案前提。

#### Scenario: allinone 使用 localfs
- **WHEN** 部署位置为本机且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: split 云写入 Edge 可读
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: split 云写入 Edge 可读（TOS）
- **WHEN** 控制面将对象 Put 到 TOS 的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

#### Scenario: 本机使用 localfs
- **WHEN** 部署位置为本机且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: 远程云写入 Edge 可读
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: 远程云写入 Edge 可读（TOS）
- **WHEN** 控制面将对象 Put 到 TOS 的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

### Requirement: 引用仍使用 BlobRef
跨进程传递文件 MUST 继续使用 `BlobRef`，MUST NOT 在 MQ 消息中携带文件二进制作为成功主路径。

#### Scenario: status 仅带引用
- **WHEN** 执行成功
- **THEN** `task.status` 的 outputs 为 BlobRef 列表而非文件字节

### Requirement: Blob 支持火山引擎 TOS 驱动
系统 MUST 提供 `blob.driver=tos` 适配器，基于火山引擎 TOS 官方 SDK 实现 `blob.Store`。Put/Get MUST 使用逻辑 key（含 `inputs/`、`jobs/`、`outputs/` 前缀约定），MUST 返回/接受 `BlobRef`，MUST NOT 在 MQ 消息中携带文件二进制作为成功主路径。

#### Scenario: split 使用 TOS 云写入 Edge 可读
- **WHEN** runtime_mode 为 split、blob 驱动为 tos，控制面将对象 Put 到 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读（TOS）
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`（blob 驱动为 tos）
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

### Requirement: 模式与驱动一致性校验（blob 维）
系统 MUST 在保存设置或启动时校验部署位置与 blob 驱动组合合法；非法组合 MUST 失败并给出可诊断错误。远程下 blob 驱动 MUST 为 `s3` 或 `tos`。MUST NOT 要求与 `queue.driver=redis` 搭配作为合法前提。

#### Scenario: split 允许 TOS + Redis
- **WHEN** 部署位置为远程、blob 驱动为 tos
- **THEN** 校验通过（在其它必填项满足时），且 MUST NOT 再要求搭配 Redis 队列

#### Scenario: split 允许 S3 + Redis
- **WHEN** 部署位置为远程、blob 驱动为 s3
- **THEN** 校验通过（在其它必填项满足时），且 MUST NOT 再要求搭配 Redis 队列

#### Scenario: split 拒绝 localfs
- **WHEN** 部署位置为远程且 blob 驱动为 localfs
- **THEN** 校验失败并说明原因

#### Scenario: 远程允许 TOS
- **WHEN** 部署位置为远程、blob 驱动为 tos
- **THEN** 校验通过（在其它必填项满足时）

#### Scenario: 远程允许 S3
- **WHEN** 部署位置为远程、blob 驱动为 s3
- **THEN** 校验通过（在其它必填项满足时）

#### Scenario: 远程拒绝 localfs
- **WHEN** 部署位置为远程且 blob 驱动为 localfs
- **THEN** 校验失败并说明原因

