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
系统 MUST 在保存设置或启动时校验部署位置与 blob 驱动组合合法；非法组合 MUST 失败并给出可诊断错误。远程下 blob 驱动 MUST 为 `s3`、`tos` 或 `sharedfs`。MUST NOT 要求与 `queue.driver=redis` 搭配作为合法前提。

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

#### Scenario: 远程允许 sharedfs
- **WHEN** 部署位置为远程且 blob 驱动为 sharedfs
- **THEN** 校验通过（在其它必填项满足时）

### Requirement: 对象存储连通性检查
系统 MUST 提供对象存储连通性检查能力，供 Setup 向导在保存前校验配置：`localfs` MUST 校验目录存在且可写；`s3` 与 `tos` MUST 使用对应 SDK 校验 endpoint/region/bucket/密钥（HeadBucket），无需读写业务对象；检查失败 MUST 给出可诊断错误。

#### Scenario: S3/TOS 连通性检查通过
- **WHEN** 向导提交 S3 或 TOS 配置，endpoint/region/bucket/密钥正确且 bucket 可达
- **THEN** 连通性检查通过，向导可继续

#### Scenario: 密钥错误被识别
- **WHEN** 向导提交的 S3/TOS 密钥无效（如 InvalidAccessKeyId、SignatureDoesNotMatch）
- **THEN** 连通性检查失败，错误信息可诊断

#### Scenario: bucket 不存在被识别
- **WHEN** 向导提交的 S3/TOS bucket 不存在（如 NoSuchBucket、404）
- **THEN** 连通性检查失败，错误信息可诊断

#### Scenario: localfs 目录校验
- **WHEN** 向导提交 localfs 配置
- **THEN** 目录存在且可写时检查通过；目录不可写或无法创建时检查失败并给出可诊断错误

### Requirement: bucket 不存在时可确认自动创建
`s3` 与 `tos` 的连通性检查发现 bucket 不存在时，系统 MUST 返回可识别的 `bucket_not_found` 结果并附带 bucket 名；向导 MUST 提示用户是否代为创建；用户确认后系统 MUST 创建该 bucket（S3 `CreateBucket`、TOS `CreateBucketV2`）并再次校验，创建成功后才标记连接正常；用户未确认时 MUST NOT 创建。

#### Scenario: bucket 不存在时提示创建
- **WHEN** 向导测试连通性且 bucket 不存在（404 / NoSuchBucket）
- **THEN** 向导提示「bucket 不存在，是否帮你创建」，未确认前不创建

#### Scenario: 确认后自动创建 bucket
- **WHEN** 用户确认创建 bucket 且账号具备建桶权限
- **THEN** 系统创建 bucket 并复检通过，向导显示连接正常

#### Scenario: 建桶权限不足
- **WHEN** 用户确认创建 bucket 但账号无建桶权限
- **THEN** 创建失败并给出可诊断错误，不标记连接正常

### Requirement: 共享目录驱动（SMB/NFS 挂载）
系统 MUST 提供 `blob.driver=sharedfs` 适配器，复用 `localfs` 实现（目录路径即挂载点），用于同机房多设备通过 SMB/NFS 挂载共享同一目录的场景。控制面与 Edge 挂载同一目录后 MUST 能按逻辑 key（`inputs/`、`jobs/`、`outputs/`）互通读写；`sharedfs` MUST 要求非空路径；远程部署 MUST 允许 `sharedfs`（区别于仅限本机的 `localfs`）。

#### Scenario: 共享目录写入 Edge 可读
- **WHEN** 控制面与 Edge 挂载同一 SMB/NFS 共享目录且 `blob.driver=sharedfs`，控制面将对象 Put 到 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: sharedfs 要求非空路径
- **WHEN** 提交 `blob.driver=sharedfs` 且路径为空
- **THEN** 校验失败并给出可诊断错误

#### Scenario: 远程允许 sharedfs
- **WHEN** 部署位置为远程且 `blob.driver=sharedfs`
- **THEN** 校验通过（在其它必填项满足时）

#### Scenario: sharedfs 连通性检查
- **WHEN** 向导提交 `sharedfs` 配置并点「连通性测试」
- **THEN** 目录存在且可写时检查通过；不可写时给出可诊断错误

