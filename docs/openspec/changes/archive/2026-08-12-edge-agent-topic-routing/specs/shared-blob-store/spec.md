## ADDED Requirements

### Requirement: Blob 双驱动（localfs 与 S3 兼容）
系统 MUST 提供可配置的 `blob.Store` 后端：至少支持 localfs 与 S3 兼容实现。`allinone` 默认使用 localfs；`split` 跨机主路径 MUST 使用 S3 兼容（或等价共享对象存储），使得云 Put 的对象可被 Edge Get，Edge Put 的产物可被云通知路径 Get。逻辑 key 前缀（`inputs/`、`jobs/`、`outputs/`）MUST 一致。

#### Scenario: allinone 使用 localfs
- **WHEN** runtime_mode 为 allinone 且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: split 云写入 Edge 可读
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

### Requirement: 引用仍使用 BlobRef
跨进程传递文件 MUST 继续使用 `BlobRef`，MUST NOT 在 MQ 消息中携带文件二进制作为成功主路径。

#### Scenario: status 仅带引用
- **WHEN** 执行成功
- **THEN** `task.status` 的 outputs 为 BlobRef 列表而非文件字节
