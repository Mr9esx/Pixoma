# Verification Report: topic-routing

- **日期**：2026-08-20
- **verify_mode**：full（31 任务 / 12 delta specs / 111 变更文件，均超轻量阈值）
- **范围**：`3f2cc676...HEAD`（base-ref 来自计划 frontmatter；当前 HEAD `f6ca566`）
- **语言**：zh-CN
- **工作区**：`feat-init`（isolation=current，用户确认）；`configs/cases/*.json` 等未提交改动为**用户既有工作**（本次 change 开始前已存在），不属于本 change 验证输入，本报告不将其计入。

## Summary Scorecard

| 维度 | 状态 |
|---|---|
| Completeness | 31/31 tasks 完成；12 个 delta capability 均有实现证据 |
| Correctness | 12/12 capability 需求映射到实现与测试；并发/重试/回退关键场景有契约测试 |
| Coherence | 设计决策全部落地（Topic 调度、条件协议、DB CAS、有界重试、默认 Topic、presence 首写）；handoff hash 因 tasks.md 勾选而变化（设计阶段产物快照，verify 流程按 hash 不一致全量读取，无跳过） |

## 验证证据（fresh）

| 检查 | 命令 | 结果 |
|---|---|---|
| 构建 | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| 全量测试 | `go test ./... -count=1` | 68 包 ok，0 FAIL |
| 并发竞态 | `go test ./internal/runtime/domain/ ./internal/runtime/application/orchestrator/ ./test/integration/ -race -count=1` | 通过 |
| 任务勾选 | `rg -c '^- \[x\]' tasks.md` | 31/31 |
| 构建证据 | `comet state record-check ... build` | 已记录 `go build ./...` 与 `go test ./...` |

## Completeness

- tasks.md 31/31 已勾选（数据模型 5、条件协议 5、Topic/Admin API 5、调度 3、原子领取 4、失败回收 4、端到端 5）。
- 12 个 delta capability 实现证据：
  - `condition-protocol` → `internal/runtime/domain/condition/`（rule/provider/evaluate + 扩展性测试）
  - `topic-routing-config` → `internal/catalog/domain/document.go` RoutingConfig + validation
  - `topic-admin` → `internal/httpapi/topics/` + `internal/platform/topic/`（含 default 保护）
  - `dispatch-topic-routing` → `internal/runtime/application/routing/router.go`（首个命中/回退/错误传播）
  - `edge-agent` → `apps/edge-agent`（`EDGE_SUBSCRIBE_TOPICS` 解析 + presence 载荷）
  - `agent-pull-dispatch` → `ClaimNextWithLease(topics)` 原子 CAS + `RequeueAfterFailure` + `RequeueExpiredLeases`
  - `task-orchestrator` → `service.go` resolveTopic/dispatchTask/topicHasOnlineConsumer/applyStatus
  - `workflow-protocol` → CaseDocument.Routing（json roundtrip 测试）
  - `case-admin-api` → `cases.Handler` + `validation.ValidateRouting`（存在/启用校验）
  - `task-persistence` → `TaskRow` dispatch_topic/attempts/requeue_at + 迁移
  - `comfy-instance-pool` → `edge.Record.SubscribeTopics` + `EffectiveTopics` + PATCH 绑定
  - `platform-bootstrap` → `EnsureDefaultTopic` 幂等种子 + 启动迁移

## Correctness（需求 ↔ 实现）

| 核心需求 | 实现 | 测试证据 |
|---|---|---|
| 同一任务只被一台消费 | gorm/memory `ClaimNextWithLease` 事务内条件 UPDATE | `TestMemory_ConcurrentClaimEachTaskOnce`、`TestAgent_ConcurrentClaimTopicSingleTask` |
| 首个命中即投 / 无命中回退 default | `routing.Resolve` | router_test 4 例 + 集成 `dispatch_topic=default` |
| 求值错误保持 pending 不误投 | `dispatchTask` 记录 `routing:` reason | `TestDispatch_EvalErrorKeepsPending` |
| 目标 Topic 无在线消费者保持 pending | `topicHasOnlineConsumer` | `TestDispatch_NoOnlineConsumerKeepsPending` |
| 失败有界重试 5s/15s/45s、超限 failed | `RequeueAfterFailure` + applyStatus | `TestRequeueAfterFailure_BoundedRetries`、`TestApplyStatus_*` |
| 租约过期回收不烧 attempts | `RequeueExpiredLeases`（清 edge、置 requeue_at=now） | `TestMemory_LeaseExpiryRequeueKeepsAttempts` |
| 默认 Topic 种子/不可删/不可禁用 | `EnsureDefaultTopic` + handler 保护 | seed_test、`TestTopics_DeleteProtections`、`TestTopics_CannotDisableDefault` |
| 节点订阅 presence 首写 + 管理端覆盖 | agent presence + `UpdateSubscribeTopics` | `TestAgent_PresenceWritesSubscribeTopicsOnce`、PATCH 测试 |
| 新增条件不改引擎 | provider 注册 + schema | `TestEvaluate_NewProviderNoEngineChange`、attributes API |
| 旧数据兼容（空 topic=default、迁移清旧持有者） | `MigrateLegacyTasks` + 查询兜底 | `TestMigrateLegacyTasks`、claim 测试 legacy 行 |

## Coherence

- Design Doc 决策逐项落地：D1 数据落点 ✓、D2 条件协议（provider + schema、引擎零改动）✓、D3 先路由后抢占 ✓、D4 DB 条件更新原子领取 ✓、D5 失败重回/超时回收 + StormGuard ✓、D6 API 形状 ✓。
- `PrepareForJob` 去 edge 化与 Design Doc §5.1 一致（job 包 edge 无关，claim 时写 edge_id）。
- 未发现 delta spec 与 design doc 矛盾；verify 期间未修改实现/spec/Design Doc。

## Issues

### CRITICAL

无。

### WARNING

无。

### SUGGESTION

- `ClaimNextWithLease` 的 `(status, dispatch_topic, created_at)` 复合索引未建（现状为 status、dispatch_topic 各自索引）：当前数据量可接受，规模增长后补。
- 任务流程编辑器（`task-flow-editor` change）尚未实现，`/api/v1/routing/attributes` 契约已就绪，等待该 change 消费。

## 审查去重说明

`review_mode: standard` 的 build 阶段最终审查已完成（见 `docs/openspec/changes/topic-routing/review-notes.md`）：子代理派发通道不可用（连续多次未收到任务载荷），经用户确认后由主会话对完整 diff 执行同等审查；发现两项（LIKE 通配符注入、default 可禁用）已在 build 内修复并补测试，随后才进入 verify。verify 阶段无新增实现改动，故不重复评审。

## Final Assessment

**All checks passed. Ready for archive.** 无 CRITICAL / WARNING；2 条 SUGGESTION 已记录接受理由与影响范围。
