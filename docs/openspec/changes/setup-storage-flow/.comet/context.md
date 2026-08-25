# Comet Design Handoff

- Change: setup-storage-flow
- Phase: design
- Mode: compact
- Context hash: ca95f1f03418d6be10d00aa58cd6146f27d1ee6393afdd1b09366bc84a268d7d

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/setup-storage-flow/proposal.md

- Source: docs/openspec/changes/setup-storage-flow/proposal.md
- Lines: 1-33
- SHA256: 1afd0e574ec629311404611392b11122acdd0b014e539ab7b0578f660acacc7d

```md
## Why

Setup 向导目前先问「出图机器在哪」（本机/远程），再进对象存储配置。对多数用户这是多余的一步：部署位置本质上由对象存储选型决定（本机目录=本机，S3/TOS=远程），且对象存储配置目前没有任何连通性校验，填错 endpoint/密钥/bucket 要到保存后才暴露。向导应直接进入对象存储配置，并像数据库一样提供连通性测试。

## What Changes

- **去掉「出图机器在哪」步骤**：Setup 步骤变为 密码 → 数据库 → 对象存储；`placement` 由对象存储驱动推断（`localfs`→本机，`s3`/`tos`→远程），后端 `settings.Validate` 的「远程禁止 localfs」约束继续生效。
- **对象存储连通性测试**：对象存储步骤新增「连通性测试 + 继续」按钮（复用数据库步骤模式）；`localfs` 校验目录可写，`s3`/`tos` 用 HeadBucket 校验 endpoint/region/bucket/密钥可达；通过显示 success Alert，失败显示 destructive Alert（中文友好文案 + 实际详情）。
- **后端**：新增对象存储连通性检查能力（`blob` 包各驱动 `Check`、工厂级带显式凭据的检查）；Setup 新增 `POST /api/v1/setup/blob-test` 端点。
- **规格与文档**：更新 `setup-wizard`（移除本机/远程选择、对象存储直接配置与连通性测试）、`shared-blob-store`（连通性测试需求）；README 同步说明。

## Non-Goals

- 不新增 OSS / Azure Blob 等新对象存储驱动（本期只对现有 `localfs`/`s3`/`tos` 做连通性测试；OSS 等作为后续驱动扩展）。
- 不做自动创建 bucket（连通性测试只校验可达，bucket 不存在时报错，不自动创建）。
- 不改数据库步骤行为与设置页对象存储配置。
- 不在向导内重做 Edge 部署指引（部署说明保留在 README/文档）。

## Capabilities

### New Capabilities

（无新能力，均为既有能力的行为变更。）

### Modified Capabilities
- `setup-wizard`: 移除「出图机器在哪」步骤，直接进入对象存储配置；对象存储步骤支持连通性测试（同数据库模式）。
- `shared-blob-store`: 对象存储驱动支持连通性检查（localfs 目录可写、S3/TOS HeadBucket 校验）。

## Impact

- 前端：`web/admin` Setup 向导（步骤序列、placement 推断、对象存储表单 + 连通性测试/继续按钮）、合同测试与步骤测试。
- 后端：`internal/platform/blob`（localfs/s3/tos 新增 `Check`）、`internal/platform/blob/factory`（带显式凭据的检查入口）、`internal/httpapi/setup`（`blob-test` 端点）、设置校验兼容。
- 文档：`setup-wizard`、`shared-blob-store` 规格、README。

```

## docs/openspec/changes/setup-storage-flow/design.md

- Source: docs/openspec/changes/setup-storage-flow/design.md
- Lines: 1-64
- SHA256: 93a6e273b4c50bea3174818ee3889ee2a258f7b4c79e58b758cba3bdb6552837

```md
## Context

现状（动机见 `proposal.md`）：

- Setup 向导步骤为 密码 → 数据库 → 出图机器在哪（placement）→ 对象存储；placement 决定 blob 驱动合法组合（远程禁止 localfs）。
- 对象存储步骤没有任何连通性校验，填错 endpoint/密钥/bucket 要到 draft 保存后才暴露。
- `settings.Settings.Validate` 已按 placement 校验 blob 合法组合；`blob` 各驱动（localfs/s3/tos）只有 Put/Get，无连通性检查。
- 数据库步骤已建立「连通性测试 + 继续」双按钮与 success/destructive Alert 模式，可直接复用到对象存储步骤。

## Goals / Non-Goals

**Goals:**

- Setup 去掉「出图机器在哪」步骤，数据库配置后直接进入对象存储配置；`placement` 由 blob 驱动推断，远程禁止 localfs 约束在 API 层继续生效。
- 对象存储步骤支持连通性测试：localfs 校验目录可写；S3/TOS 用 HeadBucket 校验 endpoint/region/bucket/密钥；成功 success Alert、失败 destructive Alert。

**Non-Goals:**

- 不新增 OSS/Azure 等对象存储驱动（本期只对现有 localfs/s3/tos 做连通性测试）。
- 不自动创建 bucket（连通性测试只校验，不创建）。
- 不改数据库步骤行为、不改设置页对象存储配置、不在向导内重做 Edge 部署指引。

## Decisions

### D1：移除 placement 步骤，placement 由 blob 驱动推断

前端 `SETUP_STEPS` 变为 密码 → 数据库 → 对象存储；`draft()` 不再携带用户选择的 placement，而是按 `blob_driver` 推断：`localfs`→`local`，`s3`/`tos`→`remote`。后端 `settings.Validate` 不变，继续拒绝 `remote + localfs`（API 层保护）；前端不提供该组合输入。

### D2：对象存储连通性检查（后端）

- `blob/localfs`、`blob/s3`、`blob/tos` 各新增 `Check(ctx) error`：
  - localfs：`os.MkdirAll` + 探测文件写入/删除。
  - s3：`HeadBucket`。
  - tos：TOS SDK `ClientV2.HeadBucket`。
- `blob/factory` 新增 `Check(ctx, CheckOptions)`，`CheckOptions` 带显式凭据（endpoint/region/bucket/accessKey/secretKey/localRoot），不依赖环境变量回退，供向导测试用。
- Setup 新增 `POST /api/v1/setup/blob-test`：请求体为 blob 配置（driver/root/endpoint/region/bucket/keys），先做合法性校验，再调用 `factory.Check`；成功返回 ok，失败返回可诊断错误。
- 自动建 bucket：`s3`/`tos` 各新增 `EnsureBucket(ctx)`（`HeadBucket` 探测 → 不存在时 `CreateBucket`/`CreateBucketV2` → 再 `HeadBucket` 确认）；`blob` 包新增哨兵错误 `ErrBucketNotFound`。`blob-test` 检查到 bucket 不存在且未确认创建时返回 `{ok:false, code:"bucket_not_found", bucket}`，前端提示「是否帮你创建」，用户确认后带 `auto_create_bucket:true` 重试，后端创建后返回 ok。

### D3：前端对象存储步骤复用数据库步骤交互

- 对象存储步骤增加「连通性测试」按钮（调 `blob-test`）+「继续」按钮；通过显示 success Alert（`CircleCheck` + 连接正常），失败显示 destructive Alert。
- 新增 `blob-error.ts`：常见对象存储错误（InvalidAccessKeyId、SignatureDoesNotMatch、NoSuchBucket、AccessDenied、connection refused、timeout、空配置等）映射中文友好标题 + 实际详情，复用 `setupErrorCopy` 的 Alert 形态。
- localfs 选择仍展示目录输入；S3/TOS 展示 endpoint/region/bucket/access key/secret key。
- bucket 不存在时：测试返回 `bucket_not_found` → 展示提示「bucket `xxx` 不存在，帮你创建？」（创建/取消），确认后带 `auto_create_bucket` 重试，创建成功显示连接正常。

### D4：步骤序列与测试

- `setup-steps.ts`：删除 `placement` 步骤与 copy；`initialSetupStep`/`previousSetupStep`/`setupStepIndex` 相应更新。
- 更新 `setup-steps.test.ts` 与合同测试：3 步流程、无「出图机器在哪」、对象存储测试按钮与 Alert 断言。

## Risks / Trade-offs

- [创建 bucket 需要账号权限] → 用户确认创建后若权限不足，报可诊断错误；README 说明需 `CreateBucket`/`PutBucket` 权限。
- [OSS 等新驱动缺失] → 本期非目标；`blob-test` 端点按 driver 分发，后续加驱动只需补 `Check`。
- [placement 推断改变既有保存语义] → 前端 draft 发送推断值，后端 Validate 行为不变；既有已初始化部署不受影响（putSettings 仍从引导态读 DB 连接）。

## Migration Plan

- 无破坏性 schema 变更；仅向导流程与新增端点。
- 新部署按新步骤完成向导；既有部署不受影响。

## Open Questions

- 是否需要支持 MinIO 等 S3 兼容自建（当前 s3 驱动已支持 endpoint + path-style）——连通性测试直接复用 HeadBucket，无需额外决策。

```

## docs/openspec/changes/setup-storage-flow/tasks.md

- Source: docs/openspec/changes/setup-storage-flow/tasks.md
- Lines: 1-24
- SHA256: 11d206288328c6219dc5d272576656c5c16c385016ac4172a48d800ae30dc947

```md
## 1. 向导移除「出图机器在哪」步骤

- [ ] 1.1 `setup-steps.ts` 删除 `placement` 步骤与 copy，步骤序列变为 密码 → 数据库 → 对象存储；`initialSetupStep`/`previousSetupStep`/`setupStepIndex` 更新
- [ ] 1.2 `setup-wizard.tsx` 删除 placement RadioGroup；`draft()` 的 `placement` 改为按 `blob_driver` 推断（localfs→local，s3/tos→remote）
- [ ] 1.3 更新 `setup-steps.test.ts` 与合同测试：3 步流程、无「出图机器在哪」文案

## 2. 对象存储连通性测试（后端）

- [ ] 2.1 `blob/localfs`、`blob/s3`、`blob/tos` 新增 `Check(ctx)`：目录可写探测 / `HeadBucket`（S3 与 TOS），含单元测试
- [ ] 2.2 `blob/factory` 新增 `Check(ctx, CheckOptions)`（显式凭据，不依赖环境变量回退）
- [ ] 2.3 `s3`/`tos` 新增 `EnsureBucket(ctx)`（HeadBucket 探测 → CreateBucket/CreateBucketV2 → 复检），`blob` 包新增 `ErrBucketNotFound`，含单元测试
- [ ] 2.4 `internal/httpapi/setup` 新增 `POST /api/v1/setup/blob-test`：校验 blob 配置 → `factory.Check` → ok / `bucket_not_found` / 可诊断错误；带 `auto_create_bucket` 时先 `EnsureBucket`，含单测

## 3. 前端对象存储步骤连通性测试

- [ ] 3.1 `setup-wizard.tsx` 对象存储步骤增加「连通性测试 + 继续」按钮与 success/destructive Alert（复用数据库步骤模式）；S3/TOS 显示 endpoint/region/bucket/密钥字段
- [ ] 3.2 新增 `blob-error.ts`：常见对象存储错误映射中文友好标题 + 实际详情，含单元测试
- [ ] 3.3 bucket 不存在时前端提示「是否帮你创建」并支持确认后自动创建（带 `auto_create_bucket` 重试），含合同测试
- [ ] 3.4 合同测试：对象存储测试按钮、Alert 展示、无「出图机器在哪」

## 4. 文档与验证

- [ ] 4.1 README 更新：向导步骤（密码/数据库/对象存储）、对象存储连通性测试、bucket 不存在可确认自动创建（需建桶权限）
- [ ] 4.2 `go build ./...` + `go test ./...`；`pnpm tsc -b` + `pnpm vitest run`（web/admin）

```

## docs/openspec/changes/setup-storage-flow/specs/setup-wizard/spec.md

- Source: docs/openspec/changes/setup-storage-flow/specs/setup-wizard/spec.md
- Lines: 1-71
- SHA256: 920913dfb028df21ed072f7f2f3ace26884cf77710f314faba944e858f6d26d2

```md
## MODIFIED Requirements

### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；业务数据库步骤 MUST 支持 `sqlite`、`mysql`、`postgres` 三种驱动；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误；对象存储步骤 MUST 提供连通性测试。

#### Scenario: 本机路径完成向导
- **WHEN** 用户配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置），`placement` 推断为本机

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
- **THEN** 向导直接展示对象存储配置，不再出现「出图机器在哪」步骤；保存时 `placement` 由对象存储驱动推断（`localfs`→本机，`s3`/`tos`→远程）

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

### Requirement: 本机与远程部署说明
系统 MUST 在向导完成页或项目文档中说明本机/远程两种部署与 `pixoma-edge-agent` 的部署要点（控制面地址、instance_id、鉴权、blob）；向导流程 MUST NOT 要求用户先选择部署位置，部署位置由对象存储选型决定。

#### Scenario: 完成向导后可获得部署指引
- **WHEN** 用户完成初始化向导
- **THEN** 完成页/README 提供 Edge 部署要点（控制面地址、节点 token、blob 说明），无需在向导中先选本机/远程

```

## docs/openspec/changes/setup-storage-flow/specs/shared-blob-store/spec.md

- Source: docs/openspec/changes/setup-storage-flow/specs/shared-blob-store/spec.md
- Lines: 1-35
- SHA256: cf8b66c120869943750665862f1c1ed648acc38b6d21d6343691dd25a36ac18c

```md
## ADDED Requirements

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

```
