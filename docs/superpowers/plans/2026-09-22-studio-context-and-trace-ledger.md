# Studio 上下文与 Trace 账本重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Studio 的对话上下文和模型运行轨迹重构为同一追加事件底座上的两类独立投影：前者保证可继续对话，后者准确解释每一次模型/工具调用。

**Architecture:** `studio_events` 保持追加写入，但事件明确区分 model-visible surface、上下文压缩和诊断轨迹。上下文编译器只读取可见消息投影与已提交的压缩边界；Trace 只读取 run/step/attempt 的诊断投影，绝不回流进模型。模型网关在序列化请求、收到首个有效流 chunk、完成/失败处写入可关联的事件，工具执行器以相同 `step_id` 参与轨迹。

**Tech Stack:** Go、Eino ADK、GORM、OpenAI/Responses/Anthropic-compatible adapters、React、Vitest。

## Global Constraints

- 保留当前工作树中既有的模型限制、压缩和 transcript 修改；不回退或覆盖用户改动。
- 所有新增行为遵循 TDD：先写失败测试并实际观察失败，再写最小实现。
- 完整 prompt、工具参数和工具结果按敏感字段脱敏；原始大载荷通过 Blob 引用保存，不写入普通 API 响应。
- Trace 事件不参与 `historyBeforeMessage` / 上下文编译，也不能作为 fake user/system message 注入模型。
- 压缩只有在摘要和边界持久化成功后才对后续 Run 生效；失败必须保留原历史并记录诊断事件。
- 没有真实流式首 token 的调用必须将 TTFT 标为 unavailable，禁止用本地切片流伪造数据。

---

### Task 1: 模型可见回放投影与可审计压缩记录

**Files:**
- Create: `internal/studio/application/contextprojection.go`
- Create: `internal/studio/application/contextprojection_test.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Test: `internal/studio/application/transcript_test.go`
- Test: `internal/studio/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:**
- Produces `ProjectModelHistory(messages, runs, events, beforeMessageID, compaction)` returning exact `schema.Message` values, including assistant tool calls and tool results paired by `tool_call_id`.
- Produces append-only `CONTEXT_COMPACTION_STARTED`, `CONTEXT_COMPACTION_COMMITTED`, `CONTEXT_COMPACTION_FAILED` events. A committed event records `source_through_message_id`, summary, retained boundary, estimated before/after tokens, and source event IDs.
- `Session` stores only the latest committed compaction pointer; the pointer is advanced in the same transaction as the committed event.

- [ ] **Step 1: Write failing context-projection tests.** Cover a historical assistant tool call followed by its tool result, a normal multi-run text history, and an interrupted/unmatched tool call. Assert a paired tool call/result becomes model-visible and an unmatched group is excluded instead of emitting invalid history.

- [ ] **Step 2: Run the focused tests.**

  Run: `go test ./internal/studio/application -run 'ContextProjection|HistoryBeforeMessage' -count=1`

  Expected: FAIL because the model-history projector and tool-aware replay do not exist.

- [ ] **Step 3: Implement the pure projector and adapt executor history loading.** Preserve ordinary user/assistant messages, reconstruct `schema.ToolCall` and `schema.Tool` from `TOOL_CALL_*` events, and preserve stable chronological order. Do not use `TranscriptMessage` as an intermediate model contract.

- [ ] **Step 4: Write failing persistence tests for compaction commit atomicity.** Simulate a repository failure while recording the compaction event and assert the session pointer remains unchanged; assert a successful commit returns an immutable compaction record.

- [ ] **Step 5: Implement the compaction record/pointer repository operation and wire the context compactor to use it.** Remove best-effort summary persistence; a persistence failure leaves the current request's safe mechanical reduction in memory but does not advance durable compaction state.

- [ ] **Step 6: Run focused application and persistence tests.**

  Run: `go test ./internal/studio/application ./internal/studio/infrastructure/persistence -run 'Context|Transcript|SessionTranscript' -count=1`

  Expected: PASS.

### Task 2: Trace domain contract and event writer

**Files:**
- Create: `internal/studio/application/trace.go`
- Create: `internal/studio/application/trace_test.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/application/transcript.go`

**Interfaces:**
- Produces `TraceAttempt` and `TraceStep` projections keyed by `run_id`, `step_id`, and `attempt_id`.
- Defines `MODEL_REQUEST_STARTED`, `MODEL_FIRST_TOKEN`, `MODEL_REQUEST_FINISHED`, `MODEL_REQUEST_FAILED`, `MODEL_RETRY`, `TOOL_CALL_STARTED`, `TOOL_CALL_FINISHED`, and compaction Trace event payloads.
- `MODEL_REQUEST_STARTED` contains a redacted compiled message snapshot, protocol-normalized request metadata, provider/model, model limits, token estimate, selected tool schema digest, and prompt blob reference/hash when the full snapshot is retained.
- Finished/failed events contain duration, TTFT availability/value, actual usage, provider request ID, HTTP status, finish reason, and redacted error identity.

- [ ] **Step 1: Write failing trace-projection tests.** Feed one run containing two model attempts, a retry, a streamed first token, and a tool call/result. Assert the projection exposes exact prompt metadata, first-token latency, total duration, usage, retry association, and tool duration.

- [ ] **Step 2: Run the focused tests.**

  Run: `go test ./internal/studio/application -run Trace -count=1`

  Expected: FAIL because attempt-level trace types and projection do not exist.

- [ ] **Step 3: Implement versioned payload types, redaction helpers, and the pure Trace projection.** Make unknown future event types safely visible as opaque diagnostic rows. Keep Trace events out of `SessionTranscript.Messages`.

- [ ] **Step 4: Extend the execution writer with a typed trace emitter.** Give every model call an `attempt_id`; associate tool events with the active `step_id`; persist event timestamps at the source rather than assigning them in the UI.

- [ ] **Step 5: Run focused tests.**

  Run: `go test ./internal/studio/application -run 'Trace|Transcript' -count=1`

  Expected: PASS.

### Task 3: Provider instrumentation and truthful streaming telemetry

**Files:**
- Modify: `internal/studio/infrastructure/modelprovider/openai_compatible.go`
- Modify: `internal/studio/infrastructure/modelprovider/eino_chat_model.go`
- Modify: `internal/studio/infrastructure/modelprovider/openai_compatible_test.go`
- Modify: `internal/studio/infrastructure/modelprovider/eino_chat_model_test.go`
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Modify: `internal/studio/infrastructure/einoagent/engine_test.go`

**Interfaces:**
- Adds request-scoped `TraceObserver` to the provider adapter without coupling `modelprovider` to Studio persistence.
- Observes request serialization, request start, response headers/status/request ID, first meaningful provider delta, final usage/finish, and failure.
- Reports `TTFTUnavailable` for non-streaming and locally synthesized stream paths; reports a measured duration only for genuine provider streaming.
- Captures OpenAI Chat streaming usage when supplied and forwards it into `schema.ResponseMeta` or the trace finish event.

- [ ] **Step 1: Write failing provider tests.** Use an `httptest` SSE endpoint with controlled chunk timing and request ID header. Assert the observer receives redacted request metadata, first token after the first non-empty text/reasoning/tool delta, terminal duration, provider request ID, and usage. Add a non-stream test asserting TTFT is unavailable.

- [ ] **Step 2: Run focused provider tests.**

  Run: `go test ./internal/studio/infrastructure/modelprovider -run 'Trace|Stream' -count=1`

  Expected: FAIL because no observer or timing contract exists.

- [ ] **Step 3: Implement the provider-neutral observer and instrument `postJSON`/`StreamChat`.** Redact auth headers and credentials before event construction. Never persist raw HTTP Authorization values. Parse usage chunks where the compatible protocol supplies them.

- [ ] **Step 4: Thread the observer from Eino Engine to the execution writer.** Emit one model trace lifecycle around every model attempt, including provider rejection retry and compaction-summary requests.

- [ ] **Step 5: Run model-provider and Eino tests.**

  Run: `go test ./internal/studio/infrastructure/modelprovider ./internal/studio/infrastructure/einoagent -count=1`

  Expected: PASS.

### Task 4: Trace API and UI as an independent projection

**Files:**
- Modify: `internal/httpapi/studio/handler.go`
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/lib/api/studio.test.ts`
- Modify: `web/admin/src/features/studio/studio-trace.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- Adds an account-scoped `GET /api/v1/studio/sessions/:sessionID/trace` read model or a typed `trace` field on session detail, returning paged run/step/attempt projections and prompt-detail authorization metadata.
- Keeps `transcript` strictly focused on chat restoration; Trace no longer derives cards from chat messages.
- UI displays one expandable model-attempt row with provider/model, prompt view, estimated/actual tokens, TTFT, total duration, usage, retry/error, and nested tool calls.

- [ ] **Step 1: Write failing handler tests.** Assert a session detail or trace endpoint returns an attempt with prompt snapshot metadata, TTFT, duration, usage, error/retry, and tool timings; assert another account cannot fetch it.

- [ ] **Step 2: Run the focused HTTP tests.**

  Run: `go test ./internal/httpapi/studio -run Trace -count=1`

  Expected: FAIL because the Trace API is currently transcript-event based.

- [ ] **Step 3: Implement the read model and endpoint.** Use the application Trace projector, not frontend aggregation over raw event payloads. Return redacted prompt by default and a blob/detail reference only when authorized.

- [ ] **Step 4: Write failing frontend contract tests.** Assert Trace maps attempts separately from transcript messages, renders unavailable TTFT explicitly, and preserves tool timing/error information.

- [ ] **Step 5: Implement the Trace UI using existing shadcn components and Studio semantic classes.** Do not add explanatory filler copy; disclose sensitive prompt content only through the authorized detail action.

- [ ] **Step 6: Run frontend tests.**

  Run: `pnpm --dir web/admin test --run src/lib/api/studio.test.ts src/features/studio/studio-workspace.contract.test.ts`

  Expected: PASS.

### Task 5: End-to-end regression suite and architecture documentation

**Files:**
- Modify: `docs/superpowers/specs/2026-09-20-ai-studio-technical-design.md`
- Modify: `docs/architecture/context-management-architecture.html`
- Modify: relevant tests from Tasks 1-4

- [ ] **Step 1: Add failing end-to-end regression tests.** Cover a prior turn with tool calls, a compaction boundary, a subsequent model call, a provider context-length retry, and the resulting Trace ordering/timings.

- [ ] **Step 2: Implement only the regressions revealed by those tests.** Ensure every next-run context is a model-history projection, and every Trace is diagnostic-only.

- [ ] **Step 3: Update technical design and architecture diagram.** The diagram must present chat and Trace as independent projections; Trace collection originates at model/provider/tool boundaries.

- [ ] **Step 4: Run all verification.**

  Run: `go test ./internal/studio/... ./internal/httpapi/studio/... ./apps/pixoma/internal/application`

  Run: `pnpm --dir web/admin test --run src/lib/api/studio.test.ts src/features/studio/studio-workspace.contract.test.ts`

  Run: `git diff --check`

  Expected: PASS.
