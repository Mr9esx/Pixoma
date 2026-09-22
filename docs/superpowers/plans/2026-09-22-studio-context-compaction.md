# Studio Agent Context Compaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Pixoma Studio Agent 增加可配置模型预算、跨轮历史上下文和分层上下文压缩，避免在未知模型限制下盲目调用模型。

**Architecture:** 模型配置保存上下文窗口、最大输入 Token、最大输出 Token；Agent 启动前用这些限制、System Prompt 和 Tool Schema 计算会话预算。上下文管理器按工具结果微压缩、LLM 摘要、最近轮次、硬截断逐级降级，并通过 Session 上的摘要边界跨 Run 复用摘要。Eino 使用 `Runner.Run` 接收编译后的历史消息，并通过 `BeforeChatModel` 在同一 ReAct Run 的每次模型调用前再次检查预算。

**Tech Stack:** Go 1.25、Eino ADK 0.9.20、GORM、SQLite/MySQL/PostgreSQL、React/TypeScript、现有 Studio model API。

## Global Constraints

- 保留当前工作树已有用户修改，不使用破坏性 Git 操作。
- 后台前端优先使用已有 shadcn/ui 组件；当前仓库未找到 `.agents/skills/pixoma-design-system-skill` 和 shadcn skill，因此沿用现有 `studio-settings.tsx` 组件风格，不新增裸色值。
- Agent 未配置模型限制时必须给出明确错误，不根据模型名猜测上下文窗口。
- Token 估算必须保守，并把模型输出预算、System Prompt、Tool Schema 从输入上下文预算中扣除。
- 压缩不能删除当前用户消息，不能留下孤立的 Tool 消息，也不能把 Tool Schema 当成普通历史文本持久化。

---

### Task 1: 模型限制契约、持久化和管理端表单

**Files:**
- Modify: `internal/studio/domain/model_config.go`
- Modify: `internal/studio/application/model_config.go`
- Modify: `internal/studio/infrastructure/persistence/model_config.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: `internal/studio/application/model_config_test.go`
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/features/studio/studio-settings.tsx`
- Modify: `web/admin/src/lib/api/studio.test.ts`

**Interfaces:**
- Produces `domain.ModelLimits{ContextWindowTokens, MaxInputTokens, MaxOutputTokens}` and exposes it as `limits` in model create/update/view JSON.
- `ModelLimits.Validate()` requires all three values to be positive and each input/output limit not to exceed the context window.
- `ModelConfigService.Resolve` rejects Agent-enabled models whose limits are missing or invalid; connection tests remain usable before limits are configured.

- [ ] **Step 1: Write failing domain/service tests** for invalid zero/oversized limits, JSON round-trip fields, and Agent resolve failure when legacy rows have zero limits.
- [ ] **Step 2: Run the focused Go tests and verify they fail** because `ModelLimits` and `limits` fields do not exist.
- [ ] **Step 3: Add the domain, application input/view, persistence row, and conversion fields** using numeric columns with `default:0` so existing database rows migrate safely.
- [ ] **Step 4: Add `limits` to create/update/test payloads and the React model type/input**, with required numeric fields for context window, max input, and max output.
- [ ] **Step 5: Run backend and frontend focused tests** and verify existing model CRUD behavior still passes.

### Task 2: Provider output limits and usage contract

**Files:**
- Modify: `internal/studio/infrastructure/modelprovider/openai_compatible.go`
- Modify: `internal/studio/infrastructure/modelprovider/openai_compatible_test.go`
- Modify: `internal/studio/infrastructure/modelprovider/eino_chat_model_test.go`

**Interfaces:**
- Provider requests use configured `MaxOutputTokens` for OpenAI Chat, OpenAI Responses, and Anthropic-compatible calls.
- A zero value remains accepted only by low-level connection-test fixtures and falls back to the existing 4096 default; Agent resolution prevents zero-valued production configs.
- Existing provider usage is preserved in `schema.ResponseMeta.Usage`.

- [ ] **Step 1: Add failing request-body tests** asserting configured output limits are sent for all supported protocols.
- [ ] **Step 2: Run provider tests and verify the bodies still contain hardcoded/default values.**
- [ ] **Step 3: Implement one output-limit helper and wire protocol-specific request fields.**
- [ ] **Step 4: Run all model-provider tests.**

### Task 3: Pure context budget and compaction engine

**Files:**
- Create: `internal/studio/application/contextcompaction/manager.go`
- Create: `internal/studio/application/contextcompaction/manager_test.go`

**Interfaces:**
- `Budget{ContextWindowTokens, MaxInputTokens, MaxOutputTokens, ReservedTokens}` computes a conservative conversation budget as `min(MaxInputTokens, ContextWindowTokens-MaxOutputTokens) - ReservedTokens`, with a 10% safety margin.
- `Manage(ctx, messages, options)` returns retained messages, optional summary, active layer, retained start index, and whether hard truncation occurred.
- Layers are: clear old large read/search/list Tool results; summarize early rounds through an injected `Summarizer`; progressively keep recent rounds; hard truncate while retaining the latest user turn and valid Tool groups.
- `EstimateTokens` is deliberately conservative and replaceable; it does not claim to be provider-exact.

- [ ] **Step 1: Write failing tests** for budget math, current-message preservation, Tool group sanitization, micro compaction, summary fallback, sliding-window degradation, hard truncation, and summary trigger behavior at configured ratio.
- [ ] **Step 2: Run the package tests and verify they fail.**
- [ ] **Step 3: Implement the pure manager without database or HTTP dependencies.**
- [ ] **Step 4: Run the package tests and refactor only after green.**

### Task 4: Session history and persisted summary boundary

**Files:**
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/executor_test.go` or the existing executor test file
- Modify: `internal/studio/infrastructure/persistence/gorm_repository_test.go`
- Modify: `apps/pixoma/internal/application/migration_models_test.go`

**Interfaces:**
- Session stores `ContextSummary` and `ContextSummaryThroughMessageID`; these fields are not exposed in the normal session API view.
- `AgentRequest` receives historical messages and a callback that atomically updates the session summary boundary through the repository transaction.
- Before each Run, the executor loads session messages in chronological order, drops messages through the persisted boundary, and passes the remaining history to the Agent.

- [ ] **Step 1: Add failing repository tests** for session summary round-trip and loading only messages after a boundary.
- [ ] **Step 2: Run persistence tests and verify the new fields/methods are absent.**
- [ ] **Step 3: Add session fields, GORM columns, conversion, update method, and executor history loading.**
- [ ] **Step 4: Add an executor test proving a second Run receives prior user/assistant messages.**
- [ ] **Step 5: Run application and persistence tests.**

### Task 5: Eino Agent context compilation and runtime rewriter

**Files:**
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Modify: `internal/studio/infrastructure/einoagent/engine_test.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/executor_test.go` or existing mock executor tests

**Interfaces:**
- Engine resolves model limits, builds tools, estimates System Prompt and Tool Schema reservation, and invokes `Runner.Run` with compiled history plus the current message.
- Engine adds an Eino `BeforeChatModel` middleware that re-runs `Manage` before every model call so tool results generated inside the current ReAct loop are also bounded.
- AutoCompact calls the same configured model with tools disabled and persists the summary through the request callback; summary failures fall through to mechanical reduction.
- Summary text is inserted as a clearly delimited data block in the Agent instruction, not as a fake persisted user message.

- [ ] **Step 1: Add failing engine tests** proving historical messages reach the provider, model/tool reservation is included, missing limits fail before provider calls, and an oversized Tool result is reduced before the next model call.
- [ ] **Step 2: Run Eino engine tests and verify the current implementation only sends the current user message.**
- [ ] **Step 3: Implement compiled history, budget calculation, summary model call, middleware rewriter, and summary callback.**
- [ ] **Step 4: Run Eino engine tests and existing tool/stream tests.**

### Task 6: Verification, documentation, and operational guardrails

**Files:**
- Modify: `docs/superpowers/specs/2026-09-20-ai-studio-technical-design.md`
- Modify: `internal/studio/infrastructure/einoagent/engine_test.go`
- Modify: `internal/studio/application/model_config_test.go`
- Modify: `web/admin/src/features/studio/studio-settings.tsx` tests if field contracts require them

**Interfaces:**
- Document the actual budget equation, required model fields, fallback behavior, and the fact that local Token estimation is conservative rather than exact.
- Add regression coverage for missing limits, model output-limit request bodies, summary persistence failure, and provider prompt-too-long fallback.

- [ ] **Step 1: Add failing regression tests for the identified edge cases.**
- [ ] **Step 2: Implement the smallest fixes and run focused tests.**
- [ ] **Step 3: Run `go test ./internal/studio/... ./internal/httpapi/studio/... ./apps/pixoma/internal/application` and the Studio frontend tests.**
- [ ] **Step 4: Review `git diff` and ensure unrelated user changes are untouched.**
