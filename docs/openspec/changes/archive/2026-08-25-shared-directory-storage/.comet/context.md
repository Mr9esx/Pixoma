# Comet Design Handoff

- Change: shared-directory-storage
- Phase: design
- Mode: compact
- Context hash: d0c0f398c149180d8c8266aa7b9481b4ccada8a091044cf9df5c7c241c511a31

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/shared-directory-storage/proposal.md

- Source: docs/openspec/changes/shared-directory-storage/proposal.md
- Lines: 1-32
- SHA256: ebbfd754c47ea506d2f61cbb81f3b9c835f68159330e242f7e3158d7ea8a9c7a

```md
## Why

文件存储目前只有三类：`localfs`（本机目录，明确警告只适合同机）、`s3`/`tos`（跨网络对象存储）。同一机房多台设备共享文件的场景夹在中间：不需要 S3/TOS 的开销，但本机目录语义又不成立。SMB / NFS 这类操作系统级共享目录正好覆盖这个空档——两边挂载同一个目录即可互通，应用层只需要把它作为一等选项并提供挂载指引与可写校验。

## What Changes

- **新增 `sharedfs` 共享目录驱动**：复用 `localfs` 实现（挂载点即路径），作为独立驱动值 `blob.driver=sharedfs`；配置字段仍是目录路径。
- **校验与 placement 语义**：`sharedfs` 允许本机或远程部署（跨机场景），API 层「远程禁止 localfs」继续只针对本机目录；`sharedfs` 必须有非空路径。
- **向导「文件存储配置」**：驱动下拉新增「共享目录（SMB / NFS）」选项；选中时展示挂载路径输入、info 提示（需先在所有机器上挂载同一目录，如 `mount -t nfs` / `mount -t cifs`）、连通性测试（目录可写）与继续按钮；本机目录仍保留「仅同机」warn。
- **局域网 S3 提示**：选择 S3 时在 Endpoint 下加提示「局域网可用 MinIO 等 S3 兼容服务，如 `http://192.168.x.x:9000`」。
- **文档**：README 与规格补充 SMB/NFS 挂载指引、`sharedfs` 说明。

## Non-Goals

- 不做 WebDAV / SFTP 驱动（应用内直连协议，另行评估）。
- 应用不负责挂载 SMB/NFS（操作系统层运维职责，仅校验挂载点可写并给指引）。
- 不改 `s3`/`tos` 驱动实现；不做自动挂载脚本。

## Capabilities

### New Capabilities
- `shared-directory-storage`: 文件存储支持共享目录（SMB/NFS 挂载）驱动 `sharedfs`，覆盖同机房多设备共享文件场景。

### Modified Capabilities
- `shared-blob-store`: 新增 `sharedfs` 驱动（复用 localfs 实现），连通性检查与合法组合校验相应扩展。
- `setup-wizard`: 「文件存储配置」新增「共享目录（SMB / NFS）」选项与挂载指引、局域网 S3 提示。

## Impact

- 后端：`internal/platform/botconfig`（驱动常量）、`internal/platform/settings`（校验）、`internal/platform/blob/factory`（`sharedfs` 分发）、`internal/httpapi/setup`（blob-test 与 draft 兼容）。
- 前端：`web/admin` Setup 向导（驱动选项、共享目录表单与提示、S3 局域网提示）、合同测试。
- 文档：`shared-blob-store`、`setup-wizard` 规格、README。

```

## docs/openspec/changes/shared-directory-storage/design.md

- Source: docs/openspec/changes/shared-directory-storage/design.md
- Lines: 1-53
- SHA256: 988bc82797916ddb8881f0cb15e2699e47ee75052ce13ff061d8dc6cbbdeb65a

```md
## Context

现状（动机见 `proposal.md`）：

- `blob.Store` 已有 localfs/s3/tos 三驱动；`factory.Check`/`EnsureBucket`、向导连通性测试与 `blob-test` 端点刚落地。
- `settings.Validate` 按 placement 校验：远程禁止 localfs；向导按 blob 驱动推断 placement（localfs→本机，s3/tos→远程）。
- localfs 本质是「目录路径即存储」，SMB/NFS 挂载点对应用层就是本地路径——复用 localfs 实现即可获得跨机共享，缺的是独立驱动标识、校验语义与向导引导。

## Goals / Non-Goals

**Goals:**

- 新增 `blob.driver=sharedfs`（复用 localfs 实现），覆盖同机房多设备通过 SMB/NFS 挂载共享目录。
- 校验与 placement：远程允许 sharedfs、仍拒绝 localfs；sharedfs 要求非空路径。
- 向导「文件存储配置」新增「共享目录（SMB / NFS）」选项：挂载路径输入、info 挂载指引、连通性测试（目录可写）、placement 推断远程；S3 选项加局域网（MinIO）提示。

**Non-Goals:**

- 不做 WebDAV/SFTP 驱动；应用不负责挂载（OS 层运维）。
- 不改 s3/tos 实现；不做自动挂载脚本。

## Decisions

### D1：新增驱动值 `sharedfs`，复用 localfs

- `botconfig` 新增 `BlobDriverSharedFS = "sharedfs"`；`blob/factory` 将 `sharedfs` 分发到 `localfs.New`（同一实现，语义不同）。
- `settings.Validate`：`sharedfs` 与 `localfs` 一样要求非空 `BlobRoot`；placement 允许 local/remote（远程+sharedfs 通过，远程+localfs 仍拒绝）。
- 向导/设置页按驱动值区分选项与文案；`blob-test` 的 placement 推断 `sharedfs`→remote。

### D2：向导「共享目录（SMB / NFS）」选项

- 驱动下拉新增 `sharedfs`（文案「共享目录（SMB / NFS）」）。
- 选中时：目录路径输入（复用 localfs 表单）+ info Alert 挂载指引（先在所有机器挂载同一目录，示例 `mount -t nfs` / `mount -t cifs`）+ 连通性测试/继续（复用现有模式）。
- localfs 保留「仅同机」warn；S3 选中时 Endpoint 下加局域网提示（MinIO 等，示例 `http://192.168.x.x:9000`）。

### D3：错误映射与文档

- `blob-error.ts` 增加 sharedfs 相关文案（目录不可写等走现有 localfs 映射路径）。
- README 增加 SMB/NFS 挂载指引与 `sharedfs` 说明；规格同步。

## Risks / Trade-offs

- [挂载是 OS 层职责] → 应用只校验可写并给指引；README 明确「先挂载再配向导」。
- [Windows 与 Linux 挂载命令差异] → 指引同时给出 NFS（Linux）与 SMB/CIFS（Windows/Linux）示例。
- [sharedfs 与 localfs 同实现易混] → 用独立驱动值区分，校验与文案不共用「仅本机」语义。

## Migration Plan

- 无破坏性 schema 变更；新增驱动值与向导选项，既有部署不受影响。

## Open Questions

- 是否需要在设置页也展示 sharedfs（本期仅向导，设置页保持只读展示驱动名即可）。

```

## docs/openspec/changes/shared-directory-storage/tasks.md

- Source: docs/openspec/changes/shared-directory-storage/tasks.md
- Lines: 1-16
- SHA256: c6f2e97ce4dd3e0d86761dd0a53b7f37cfe1d1698e1b0d8186e20a2176ef4110

```md
## 1. 后端 sharedfs 驱动与校验

- [ ] 1.1 `botconfig` 新增 `BlobDriverSharedFS = "sharedfs"`；`blob/factory` 将 `sharedfs` 分发到 `localfs.New`，含单测
- [ ] 1.2 `settings.Validate`：`sharedfs` 要求非空 `BlobRoot`，允许 local/remote（远程+sharedfs 通过、远程+localfs 仍拒绝），含单测
- [ ] 1.3 `blob-test` 的 placement 推断与校验支持 `sharedfs`（含单测）

## 2. 前端向导「共享目录」选项

- [ ] 2.1 向导驱动下拉新增「共享目录（SMB / NFS）」`sharedfs` 选项：目录路径输入 + info 挂载指引（NFS/CIFS 示例）+ 连通性测试/继续；local fs 保留仅同机 warn
- [ ] 2.2 选择 S3 时 Endpoint 下加局域网提示（MinIO 等 S3 兼容服务示例）
- [ ] 2.3 合同测试：sharedfs 选项、挂载指引、S3 局域网提示、placement 推断

## 3. 文档与验证

- [ ] 3.1 README 增加 SMB/NFS 挂载指引与 `sharedfs` 说明
- [ ] 3.2 `go build ./...` + `go test ./...`；`pnpm tsc -b` + `pnpm vitest run`（web/admin）

```

## docs/openspec/changes/shared-directory-storage/specs/setup-wizard/spec.md

- Source: docs/openspec/changes/shared-directory-storage/specs/setup-wizard/spec.md
- Lines: 1-80
- SHA256: 6f1008d4593b7504a0f2ad15eeaf23536ac0ff12244499d0e2c7a655a627ff9c

```md
## MODIFIED Requirements

### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；业务数据库步骤 MUST 支持 `sqlite`、`mysql`、`postgres` 三种驱动；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误；对象存储步骤 MUST 提供连通性测试，MUST 支持共享目录（SMB/NFS 挂载）选项。

#### Scenario: 本机路径完成向导
- **WHEN** 用户配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置），`placement` 推断为本机

#### Scenario: 远程禁止 localfs
- **WHEN** 用户选择远程部署并尝试将对象存储选为 localfs
- **THEN** 向导拒绝该组合并说明原因

#### Scenario: 使用 MySQL 完成向导
- **WHEN** 用户在数据库步骤选择 MySQL、填写可达 DSN 并测连通
- **THEN** 向导继续后续步骤并可完成初始化，重启后控制面使用该 MySQL 库

#### Scenario: 使用 Postgres 完成向导
- **WHEN** 用户在数据库步骤选择 Postgres、填写可达 DSN 并测连通
- **THEN** 向导继续后续步骤并可完成初始化，重启后控制面使用该 Postgres 库

#### Scenario: 数据库不可达提示
- **WHEN** 用户在数据库步骤填写不可达或非法的 DSN
- **THEN** 向导显示可诊断错误且不进入下一步，引导态业务库连接不被修改

#### Scenario: 切换数据库驱动后 DSN 输入更新
- **WHEN** 用户在数据库步骤把驱动从 SQLite 切换为 MySQL/Postgres
- **THEN** 输入区切换为对应驱动的结构化字段（Host/端口/用户/密码/数据库等），按字段组装出的连接随之更新，不再沿用 SQLite 的路径值

#### Scenario: 结构化字段配置 MySQL/Postgres
- **WHEN** 用户选择 MySQL 或 Postgres，并填写 Host、端口、用户、密码、数据库等字段
- **THEN** 向导按字段组装连接字符串用于测连通；密码字段不回显明文；用户可在「附加参数」输入框直接追加任意 key=value 参数（MySQL 为 `&a=b`，Postgres 为空格分隔的 `key=value`），无需手拼完整 DSN

#### Scenario: 常见错误中文提示与详情
- **WHEN** 测连通或保存失败，且错误属于常见类型（拒绝连接、鉴权失败、库不存在、超时、未知驱动等）
- **THEN** 向导以 Alert 形式展示中文友好提示，并在其下方展示实际错误详情

#### Scenario: 测连通与继续分离
- **WHEN** 用户在数据库步骤点「连通性测试」且服务器可达、账号密码正确
- **THEN** 显示连接正常；「继续」始终可点，不要求先测连通，连接问题会在后续步骤以 Alert 报错

#### Scenario: 目标数据库不存在时自动创建
- **WHEN** 用户选择 MySQL/Postgres，服务器可达且账号密码正确，但目标数据库不存在
- **THEN** 向导自动创建目标数据库（MySQL `CREATE DATABASE IF NOT EXISTS`，Postgres 检测 `pg_database` 后创建）并继续；业务表在保存/启动时由 AutoMigrate 自动创建

#### Scenario: 向导不询问部署位置
- **WHEN** 用户完成数据库步骤后进入对象存储步骤
- **THEN** 向导直接展示对象存储配置，不再出现「出图机器在哪」步骤；保存时 `placement` 由对象存储驱动推断（`localfs`→本机，`s3`/`tos`/`sharedfs`→远程）

#### Scenario: 对象存储连通性测试通过
- **WHEN** 用户选择 S3 或 TOS，填写 endpoint/region/bucket/密钥并点「连通性测试」，配置正确且 bucket 可达
- **THEN** 向导以 success Alert 显示连接正常，可继续下一步

#### Scenario: 对象存储连通性测试失败
- **WHEN** 用户点「连通性测试」但 endpoint 不可达、密钥错误或 bucket 不存在
- **THEN** 向导以 destructive Alert 显示中文友好提示与实际错误详情，不标记连接正常

#### Scenario: bucket 不存在时提示自动创建
- **WHEN** 用户点「连通性测试」且 bucket 不存在
- **THEN** 向导提示「bucket `xxx` 不存在，是否帮你创建？」；用户确认后自动创建并复检，成功显示连接正常

#### Scenario: localfs 目录校验
- **WHEN** 用户选择 localfs 并点「连通性测试」
- **THEN** 目录可写时显示连接正常；目录不可写时显示可诊断错误

#### Scenario: 远程 localfs 组合在 API 层仍被拒绝
- **WHEN** 提交 `placement=remote` 且 `blob.driver=localfs` 的设置
- **THEN** 设置校验失败并说明原因（向导 UI 不提供该组合，API 层保留原约束）

#### Scenario: 选择共享目录（SMB/NFS）完成向导
- **WHEN** 用户选择「共享目录（SMB / NFS）」并填写已挂载目录，点「连通性测试」通过
- **THEN** 向导可完成初始化；`placement` 推断为远程，控制面与 Edge 挂载同一目录后按 key 互通读写

#### Scenario: 共享目录挂载指引
- **WHEN** 用户选择「共享目录（SMB / NFS）」
- **THEN** 向导展示 info 提示：需先在所有机器上挂载同一共享目录（如 `mount -t nfs` / `mount -t cifs`），连通性测试校验目录可写

#### Scenario: 局域网 S3 提示
- **WHEN** 用户选择 S3 驱动
- **THEN** Endpoint 输入下展示提示：局域网可用 MinIO 等 S3 兼容服务，如 `http://192.168.x.x:9000`

```

## docs/openspec/changes/shared-directory-storage/specs/shared-blob-store/spec.md

- Source: docs/openspec/changes/shared-directory-storage/specs/shared-blob-store/spec.md
- Lines: 1-53
- SHA256: c4108acc56452cf7186a4193b511f58e87eb0e699a053836bf58a3cb0944ca23

```md
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

```
