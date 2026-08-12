# Comet Design Handoff

- Change: blob-tos-adapter
- Phase: design
- Mode: compact
- Context hash: cf6bf8f6d40abb873f868ae942c7f9f52c5c59846f27a765138a848553f3e308

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/blob-tos-adapter/proposal.md

- Source: docs/openspec/changes/blob-tos-adapter/proposal.md
- Lines: 1-29
- SHA256: b1710454ccf30be135635fd5f7bf84e11eabbc5ac5a9a9649bb4d56dffc73aa3

```md
## Why

当前 `split` 跨机文件只支持 S3 兼容存储。部署若落在火山引擎，需要直接用 TOS 官方能力存取输入/任务包/产物，同时保留 localfs/s3，通过 `blob.driver` 切换即可。

## What Changes

- 新增 `blob.driver=tos`，基于**火山引擎 TOS 官方 SDK**实现 `blob.Store`（Put / Get）
- 扩展 `botconfig`：驱动常量、TOS 连接配置（endpoint、region、bucket、AK/SK 等）、启动校验放宽为 `split` 下 `blob ∈ {s3, tos}`
- `split` 下 `blob ∈ {s3, tos}`，与 `queue=redis` 搭配（`queue-nats-adapter` 已取消，本期不含 nats）
- botconfig / edge-agent 按驱动装配；补充测试与配置样例；**真 TOS Put→Get 为验收硬门禁**
- **非本期**：改 BlobRef 契约、改 key 前缀、删除 s3/localfs、在 MQ 中传文件二进制、NATS

## Capabilities

### New Capabilities

- （无）本期扩展既有共享对象存储能力，不新增限界能力名

### Modified Capabilities

- `shared-blob-store`：从「localfs + S3 兼容」扩展为还可选火山 TOS；`split` 下 blob 驱动校验改为 `s3 | tos`，与 redis 搭配

## Impact

- 代码：`internal/platform/blob/tos`、`internal/platform/botconfig`、bot / edge-agent 装配
- 依赖：火山引擎 TOS 官方 Go SDK
- 配置：YAML/env 增加 tos 字段与样例
- 规格：`docs/openspec/specs/shared-blob-store/spec.md`
- 校验矩阵本期：`split` → `queue=redis` ∧ `blob∈{s3,tos}`

```

## docs/openspec/changes/blob-tos-adapter/design.md

- Source: docs/openspec/changes/blob-tos-adapter/design.md
- Lines: 1-58
- SHA256: bbd1b96f69e190cc4719a4bc7afa28db73d54b41f9a3c55506281b257b803821

```md
## Context

共享对象存储已有 `localfs` 与 S3 兼容（`blob/s3`，AWS SDK v2）实现，接口为 `blob.Store`。`split` 当前强制 `blob.driver=s3`。本期新增火山引擎 TOS 官方 SDK 适配器，使火山部署可直接选 `tos`，并与 queue 驱动解耦（可与 redis/nats 任意搭配）。姊妹 change `queue-nats-adapter` 负责 NATS。

## Goals / Non-Goals

**Goals:**

- 实现 `internal/platform/blob/tos`，满足 Put / Get，契约对齐 `blob.Store`
- 配置与校验：`split` 允许 `blob ∈ {s3, tos}`，并支持与合法 queue 驱动任意搭配
- bot / edge-agent 装配；测试覆盖 Put→Get 与校验矩阵

**Non-Goals:**

- 用 S3 兼容端点「假装」TOS 作为本期正式驱动（用户已选官方 SDK）
- 改 BlobRef、key 前缀、在消息中传文件字节
- 删除 localfs/s3
- 实现预签名 URL / CDN（除非现有 s3 路径已有且必须对等；默认不对等扩展）

## Decisions

1. **官方 TOS SDK，而非复用 s3 包**  
   - 理由：用户明确要求 `driver: tos` + 火山官方 SDK；凭证/地域/endpoint 语义更贴火山。  
   - 备选：仅文档化 S3 兼容 → 否决为本期正式方案（可作为运维旁路，但不替代 `tos` 驱动）。

2. **包边界**  
   - 新包 `internal/platform/blob/tos`，实现与 `localfs`/`s3` 相同的 `blob.Store`。  
   - 不把火山类型泄漏到领域层；装配层按 driver 构造。

3. **配置字段（高层）**  
   - `blob.driver: tos`  
   - endpoint、region、bucket、access key、secret key（及 SDK 必需的其它字段）  
   - 具体 YAML key 在 Build 与官方 SDK 示例对齐。

4. **校验矩阵**  
   - `allinone`：仍 `localfs`  
   - `split`：`queue=redis` 且 `blob ∈ {s3, tos}`（nats change 已取消）

5. **测试策略**  
   - 优先可注入的 fake/HTTP 测试服务器或 SDK 接口抽象测 Put/Get 契约；真网联调不作为 CI 硬依赖。

## Risks / Trade-offs

- [官方 SDK API/依赖版本变动] → 锁定模块版本；封装在 tos 包内  
- [与 queue-nats 同时改 ValidateRuntimeDrivers] → 最终矩阵写成独立函数或表驱动，减少合并冲突  
- [大文件内存：现有 s3 Put 全量 ReadAll] → tos 初期可对齐该行为；后续若需流式再单开优化（本期不对齐新能力）

## Migration Plan

- 默认 allinone 不变  
- 现有 split + s3 零迁移  
- 新部署改 `blob.driver=tos` 并填 TOS 凭证  
- 回滚：改回 `s3` 并重启

## Open Questions

- TOS SDK 具体模块路径与最低 Go 版本兼容性（Build 时确认 go.mod）  
- 是否需要 path-style / 自定义域名等高级选项（有部署需求再加）

```

## docs/openspec/changes/blob-tos-adapter/tasks.md

- Source: docs/openspec/changes/blob-tos-adapter/tasks.md
- Lines: 1-23
- SHA256: ecaa19e7f277067ebf73ea019026ce775dfb72055be1ba07ce9872cfbe62f756

```md
## 1. 配置与校验

- [ ] 1.1 在 `botconfig` 增加 `BlobDriverTOS` 及 TOS 连接配置字段
- [ ] 1.2 放宽 `ValidateRuntimeDrivers`：`split` 允许 `blob ∈ {s3, tos}`，与 `queue=redis` 搭配
- [ ] 1.3 补充校验单测与 YAML/env 样例

## 2. TOS 适配器

- [ ] 2.1 引入火山引擎 TOS 官方 Go SDK 依赖
- [ ] 2.2 实现 `internal/platform/blob/tos`：New / Put / Get，key 清洗规则对齐 s3/localfs
- [ ] 2.3 单元测试覆盖 Put→Get、非法 key、空 bucket 等失败路径

## 3. 进程装配

- [ ] 3.1 bot 控制面按 `blob.driver` 装配 tos（保留 localfs/s3）
- [ ] 3.2 edge-agent 按 `blob.driver` 装配 tos，保证 inputs/jobs/outputs 读写主路径
- [ ] 3.3 确认 `split + localfs` 等非法组合启动失败信息可诊断

## 4. 文档与验收

- [ ] 4.1 若触及架构边界，更新架构/配置文档中的 blob 驱动列表
- [ ] 4.2 真 TOS 硬门禁：加载 `.env.tos.local` / `TOS_*` 对 `pixoma-test` Put→Get 成功；缺配置则失败
- [ ] 4.3 localfs/s3 回归通过；`split + tos + redis` 启动校验通过

```

## docs/openspec/changes/blob-tos-adapter/specs/shared-blob-store/spec.md

- Source: docs/openspec/changes/blob-tos-adapter/specs/shared-blob-store/spec.md
- Lines: 1-48
- SHA256: 331c00310902763a768023d74b38b950105659ebe218a6896e4ec2851c75414b

```md
## ADDED Requirements

### Requirement: Blob 支持火山引擎 TOS 驱动
系统 MUST 提供 `blob.driver=tos` 适配器，基于火山引擎 TOS 官方 SDK 实现 `blob.Store`。Put/Get MUST 使用逻辑 key（含 `inputs/`、`jobs/`、`outputs/` 前缀约定），MUST 返回/接受 `BlobRef`，MUST NOT 在 MQ 消息中携带文件二进制作为成功主路径。

#### Scenario: split 使用 TOS 云写入 Edge 可读
- **WHEN** runtime_mode 为 split、blob 驱动为 tos，控制面将对象 Put 到 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读（TOS）
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`（blob 驱动为 tos）
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

## MODIFIED Requirements

### Requirement: Blob 双驱动（localfs 与 S3 兼容）
系统 MUST 提供可配置的 `blob.Store` 后端：至少支持 localfs、S3 兼容实现，以及火山引擎 TOS。`allinone` 默认使用 localfs；`split` 跨机主路径 MUST 使用 `s3` 或 `tos`，使得云 Put 的对象可被 Edge Get，Edge Put 的产物可被云通知路径 Get。逻辑 key 前缀（`inputs/`、`jobs/`、`outputs/`）MUST 一致。

#### Scenario: allinone 使用 localfs
- **WHEN** runtime_mode 为 allinone 且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: split 云写入 Edge 可读（S3）
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: split 云写入 Edge 可读（TOS）
- **WHEN** 控制面将对象 Put 到 TOS 的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

### Requirement: 模式与驱动一致性校验（blob 维）
系统 MUST 在启动时校验 runtime_mode 与 blob 驱动组合合法；非法组合 MUST 失败并给出可诊断错误。`split` 下 blob 驱动 MUST 为 `s3` 或 `tos`，且 MUST 与 `queue.driver=redis` 搭配（本期不以 nats 为合法 queue 驱动）。

#### Scenario: split 允许 TOS + Redis
- **WHEN** runtime_mode 为 split、blob 驱动为 tos、queue 驱动为 redis
- **THEN** 启动校验通过

#### Scenario: split 允许 S3 + Redis
- **WHEN** runtime_mode 为 split、blob 驱动为 s3、queue 驱动为 redis
- **THEN** 启动校验通过

#### Scenario: split 拒绝 localfs
- **WHEN** runtime_mode 为 split 且 blob 驱动为 localfs
- **THEN** 进程拒绝就绪或退出，并说明原因

```
