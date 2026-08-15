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
