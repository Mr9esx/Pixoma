---
change: edge-agent-topic-routing
design-doc: docs/superpowers/specs/2026-08-10-edge-agent-dual-mode-design.md
base-ref: 21021f77eb0e128f7417ca9d83a9080a6611a5b1
---

# Edge-Agent 双模式 + 方案 A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落地 allinone（单进程 Memory+localfs）与 split（Edge+Redis Streams+S3）双模式，统一方案 A（`job_ref` 任务包）；本期不做 Topic 表达式分流。

**Architecture:** 配置选择 runtime_mode 与 queue/blob 驱动并做启动校验；云侧 prep 写 job 到 Blob 后 dispatch 带 `job_ref`；执行面（同进程或 Edge）只认 job + 本机 Comfy；产物回 Blob 再 `task.status`。

**Tech Stack:** Go、现有 `queue`/`blob`/`actuator`/`orchestrator`、Redis Streams、S3 兼容 SDK、可选 testcontainers/miniredis 类替身。

## Global Constraints

- 产物语言：zh-CN（文档）；代码标识符保持仓库英文风格
- `comfy_mock` / `COMFY_MOCK` 必须继续端到端可通
- 本期不做 Topic Admin / 投放表达式 / 会员×分类分流；dispatch 仍 `dispatch.<instance_id>`
- Edge/执行面成功主路径不连 Task/Case DB
- 架构变更须同步 `docs/architecture/`（runtime/overview）

## 文件结构

| 文件 | 职责 |
|---|---|
| `internal/platform/botconfig`（及 edge 配置） | `runtime_mode`、queue/blob 驱动与校验 |
| `internal/platform/blob/s3` | S3 兼容 `blob.Store` |
| `internal/platform/queue/redis` | Redis Streams `queue.Bus` |
| `internal/sharedkernel/events.go` | `DispatchCommand.JobRef` |
| `internal/runtime/.../actuator` | job 类型、prep、按 job 执行 |
| `internal/runtime/.../orchestrator` | prep 后带 `job_ref` 投递；split 在线判断 |
| `apps/bot/cmd/comfyui-bot` | allinone / 云侧 split 接线 |
| `apps/edge-agent/...` | Edge 入口 |
| `docs/architecture/runtime.md` 等 | 双模式文档 |

---

## 任务 1：模式配置与驱动骨架

**Files:**
- Modify: `internal/platform/botconfig/config.go`（及测试）
- Create: 配置校验辅助（可同包）

- [x] 1.1 写失败测试：`split`+`memory` / 非法组合被拒绝
- [x] 1.2 实现 `runtime_mode`、`queue.driver`、`blob.driver` 字段与默认（allinone→memory+localfs）
- [x] 1.3 启动校验通过合法组合；测试通过

## 任务 2：S3 兼容 Blob

**Files:**
- Create: `internal/platform/blob/s3/*.go`
- Test: 同目录或带 minio/fake 的测试

- [x] 2.1 写失败测试：Put 后 Get 同一 key 内容一致（可用 fake/minio）
- [x] 2.2 实现 S3 `Store`（endpoint/bucket/凭证配置）
- [x] 2.3 与 localfs 选型工厂或 bot 接线辅助；测试通过

## 任务 3：Redis Streams Queue

**Files:**
- Create: `internal/platform/queue/redis/*.go`
- Test: 可用 miniredis 或 testcontainer

- [x] 3.1 写失败测试：Publish 后 Subscribe handler 收到等价 payload；Ack 行为可测
- [x] 3.2 实现 Bus：topic→stream 映射、消费组、Ack/最小重试
- [x] 3.3 测试通过；文档注释说明与 Memory 语义差异

## 任务 4：job 包与 DispatchCommand

**Files:**
- Modify: `internal/sharedkernel/events.go`
- Create: `internal/runtime/infrastructure/actuator/job.go`（或等价）
- Test: 序列化 / 解析测试

- [x] 4.1 扩展 `DispatchCommand` 增加 `job_ref`；兼容字段策略写清
- [x] 4.2 定义 job JSON 结构（workflow、images[]、output_prefix）
- [x] 4.3 单测：marshal/unmarshal round-trip

## 任务 5：prep + 执行面改读 job

**Files:**
- Modify: `internal/runtime/infrastructure/actuator/snapshot.go`（拆 prep）
- Modify: `internal/runtime/infrastructure/actuator/worker.go`
- Modify: `internal/runtime/application/orchestrator/service.go`
- Test: 现有 snapshot/worker/orchestrator 测试更新

- [x] 5.1 写失败测试：prep 产出 job 且不含 UploadImage；缺必填输入不 Publish
- [x] 5.2 实现 prep：非图片注入 + images 清单 → Blob Put `jobs/<task_id>/job.json`
- [x] 5.3 Orchestrator：Claim 前/时 prep，dispatch 带 `job_ref`
- [x] 5.4 Worker：Get job → 拉图 Upload → Submit/Wait → outputs；坏 job_ref → failed status
- [x] 5.5 相关单测全绿

## 任务 6：allinone 接线

**Files:**
- Modify: `apps/bot/cmd/comfyui-bot/main.go`
- Modify: configs 示例 yaml（若有）

- [x] 6.1 allinone 默认：Memory + localfs + 同进程执行面 + prep
- [x] 6.2 `comfy_mock=true` 冒烟/集成：确认生成路径仍通（已有 smoke 则更新）
- [x] 6.3 提交前跑相关包测试

## 任务 7：Edge 进程与 split 行为

**Files:**
- Create: `apps/edge-agent/cmd/...`
- Modify: bot 在 split 下不订阅生产 dispatch
- Create/Modify: 最小在线信号（Redis key TTL 或等价）
- Test: 无在线 Edge 保持 pending

- [x] 7.1 Edge 入口：配置 Redis/S3/Comfy/instance_id/mock；订阅 dispatch topic；跑 Worker
- [x] 7.2 心跳/在线最小实现；Orchestrator split 选路尊重在线
- [x] 7.3 测试：无消费者时不假装 queued
- [x] 7.4 文档化 split 冒烟步骤（Redis+S3+Edge+mock）

## 任务 8：架构文档与回归

**Files:**
- Modify: `docs/architecture/runtime.md`、`overview.md`（按需）
- tasks.md 勾选同步

- [x] 8.1 更新架构文档：双模式、job_ref、Edge、驱动表
- [x] 8.2 全量相关 `go test` 回归；勾选 tasks.md 1–4.3（4.4 保持延后）
- [x] 8.3 自检：allinone+mock 主路径说明写入验收笔记或测试

## 延后（不实施）

- Topic Admin、投放表达式、会员×分类分流（tasks 4.4）
