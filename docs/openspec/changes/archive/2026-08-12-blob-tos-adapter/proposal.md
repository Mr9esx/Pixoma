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
