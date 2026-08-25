## ADDED Requirements

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

## MODIFIED Requirements

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
