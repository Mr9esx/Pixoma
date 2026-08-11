# Verification Report: edge-agent-topic-routing

- **日期：** 2026-08-11
- **分支：** `feature/20260810/edge-agent-topic-routing`
- **verify_mode：** full
- **base-ref：** `21021f77eb0e128f7417ca9d83a9080a6611a5b1`
- **Design Doc：** `docs/superpowers/specs/2026-08-10-edge-agent-dual-mode-design.md`

## Summary Scorecard

| 维度 | 结果 |
|---|---|
| Completeness | PASS |
| Correctness | PASS |
| Coherence | PASS（已记录 Implementation Divergence） |
| **总体** | **PASS** |

## Completeness

- tasks.md：17/17 `[x]`（含 4.4 明确移出本期范围）
- Superpowers plan：任务 1–8 已勾选；Topic 延后不实施
- 变更规模：相对 base-ref 约 57 files（双模式 + job_ref + Edge + Redis/S3）

## Correctness

### 需求 ↔ 实现映射（本期）

| 能力 / 要求 | 证据 |
|---|---|
| `runtime_mode` + 驱动校验 | `internal/platform/botconfig` + 测试 |
| S3 blob / Redis Streams | `internal/platform/blob/s3`、`queue/redis` + 测试 |
| 方案 A `job_ref` + prep | `actuator.PrepareJob`、`DispatchCommand.JobRef`、Worker `graphFromJob` |
| allinone 同进程 | bot main：Memory+localfs+Prep+订 dispatch |
| split Edge | `apps/edge-agent`；bot 不订生产 dispatch；`edgeonline` + `ListEnabled` |
| 无在线不投递 | `orch.Online` + `online_test` |
| 坏 job_ref → failed | `TestWorker_BadJobRefPublishesFailed` |
| Mock 主路径 | `test/integration` + `comfy_mock` 约定 |

### Spec 场景

- **已覆盖：** job 包、双驱动、Edge/allinone、Online 选路、PEL 重试、mock。
- **明确不验收：** Topic Admin / 投放表达式（delta MUST NOT；用户确认不做 Topic）。

### 验证命令（本轮）

```text
go test ./internal/platform/botconfig/ ./internal/platform/blob/... \
  ./internal/platform/queue/... ./internal/platform/edgeonline/ \
  ./internal/platform/instance/... ./internal/runtime/... \
  ./internal/sharedkernel/ ./test/integration/ ./internal/packaging/botapp/ -count=1
→ 全绿

go build ./apps/bot/cmd/comfyui-bot/ ./apps/edge-agent/cmd/edge-agent/
→ exit 0
```

### 安全速查

- 无硬编码生产密钥；S3/Redis 凭证走环境变量。
- Redis AUTH/TLS、S3 启动探测列为后续加固（非 CRITICAL）。

## Coherence

- 与 Design Doc 决策一致：双模式、方案 A、Topic 延后、Edge 心跳作存在信号。
- Open `design.md` / `proposal.md` 仍含早期 Topic 目标 → **用户选择 A**：已在 Open `design.md` 追加 **Implementation Divergence**（2026-08-11）。
- Build 期 review Critical（云 Comfy 探活拦 split、Redis PEL、fail 吞错误）已修复并有测试。

## Issues

无 CRITICAL / IMPORTANT 未关闭项。

WARNING（接受）：Redis AUTH/TLS、S3 HeadBucket、`OutputPrefix` 未驱动路径、allinone 无 `job_ref` 可回落 Workflows — 见 tasks.md Review notes。

## Verdict

**PASS** — 可进入 Archive（须用户确认归档）。
