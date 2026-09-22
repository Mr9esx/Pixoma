# Studio Workflow Tool Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the Studio Eino Agent invoke explicitly enabled Pixoma Cases as dynamic tools, run them through the existing task scheduler, and adopt their outputs as immutable Session assets and Flow nodes.

**Architecture:** Studio owns the Agent-facing workflow catalog and the durable mapping from one Agent tool call to one Pixoma Task. A narrow application port starts a validated Case task without importing Bot Session semantics. A monitor reconciles terminal task state, creates asset versions referencing the existing output blobs, and appends Studio events. The Agent run may finish after task submission; its Session receives a durable workflow operation and generated assets when the task finishes.

**Tech Stack:** Go 1.25, Eino ADK, GORM, existing Case/Task domains, platform queue/blob stores, React Flow, existing Studio REST and AG-UI events.

## Global Constraints

- Reuse `CaseDocument.Name`/`Description`/`InputSchema`; do not add Agent-only descriptions, schemas, workflow JSON, pricing or timeout fields.
- Expose a workflow only when both the Case and its account-scoped Studio `AgentWorkflowSetting` are enabled.
- Never send Comfy workflow JSON, binding metadata, plaintext credentials or blob contents to the model or Trace.
- Preserve Studio ownership checks at every durable read and write; do not reuse `internal/sessions` or Bot user identity as a Studio identity model.
- Workflow submission is approval-controlled and idempotent per Studio run/tool call.
- Reuse existing output `BlobRef`s for Studio asset versions; never copy workflow output blobs.
- Use `apply_patch` for edits, test first, and commit as `李卓洲 <1138099359@qq.com>`.

---

### Task 1: Define the durable Studio-to-Task contract

**Files:**
- Create: `internal/studio/domain/workflow_execution.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository_test.go`

**Consumes:** `domain.Run`, `domain.FlowNode`, existing Studio ownership-scoped repository rules.

**Produces:** `WorkflowExecution` with one unique `(run_id, tool_call_id)` mapping, `task_id`, Case ID, operation node ID, and terminal adoption state. Repository methods `CreateWorkflowExecution`, `GetWorkflowExecutionByTask`, `ListPendingWorkflowExecutions`, and `MarkWorkflowExecutionTerminal`.

- [ ] **Step 1: Write failing persistence tests**

```go
func TestWorkflowExecutionIsIdempotentPerRunAndToolCall(t *testing.T) {
    execution := &domain.WorkflowExecution{ID: "wx-1", RunID: "run-1", ToolCallID: "call-1", TaskID: "task-1", AccountID: "account-a", SessionID: "session-a"}
    require.NoError(t, repo.CreateWorkflowExecution(ctx, execution))
    require.Error(t, repo.CreateWorkflowExecution(ctx, execution))
}

func TestWorkflowExecutionLookupIsAccountScoped(t *testing.T) {
    _, err := repo.GetWorkflowExecutionByTask(ctx, "account-b", "task-1")
    require.ErrorIs(t, err, domain.ErrNotFound)
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/studio/infrastructure/persistence -run 'TestWorkflowExecution' -count=1`

Expected: compile failure because `WorkflowExecution` and repository methods do not exist.

- [ ] **Step 3: Add domain type and GORM row**

```go
type WorkflowExecutionStatus string
const (
    WorkflowExecutionSubmitted WorkflowExecutionStatus = "submitted"
    WorkflowExecutionSucceeded WorkflowExecutionStatus = "succeeded"
    WorkflowExecutionFailed WorkflowExecutionStatus = "failed"
    WorkflowExecutionCancelled WorkflowExecutionStatus = "cancelled"
)
type WorkflowExecution struct {
    ID, AccountID, SessionID, RunID, ToolCallID, TaskID, WorkflowID, OperationNodeID string
    Status WorkflowExecutionStatus
    ErrorMessage string
    CreatedAt, UpdatedAt, CompletedAt time.Time
}
```

Make `(run_id, tool_call_id)` and `task_id` unique in the persisted row. Add the row to `Models()` and use `WHERE account_id = ?` on all fetch/update paths.

- [ ] **Step 4: Run persistence tests and package tests**

Run: `go test ./internal/studio/domain ./internal/studio/infrastructure/persistence -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/studio/domain/workflow_execution.go internal/studio/domain/repository.go internal/studio/infrastructure/persistence/gorm_repository.go internal/studio/infrastructure/persistence/gorm_repository_test.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): persist workflow task executions'
```

### Task 2: Build the Case catalog and task-start application port

**Files:**
- Create: `internal/studio/application/workflow.go`
- Create: `internal/studio/application/workflow_test.go`
- Modify: `internal/studio/application/capability_config.go`
- Modify: `apps/pixoma/internal/application/app.go`

**Consumes:** enabled `CaseDocument`, account-scoped workflow setting, task repository, blob store, task-created publisher.

**Produces:** `ResolvedWorkflow` with stable `studio_workflow_<case-id>` tool name, JSON schema copied from `CaseDocument.InputSchema`, and an application `WorkflowStarter` that validates values, stages inputs under `studio-workflow-inputs/<task-id>`, and publishes `TaskCreated`.

- [ ] **Step 1: Write failing application tests**

```go
func TestResolveWorkflowsIncludesOnlyEnabledAgentCases(t *testing.T) {
    workflows, err := service.ResolveWorkflows(ctx, "account-a")
    require.NoError(t, err)
    require.Equal(t, []string{"studio_workflow_12"}, names(workflows))
}

func TestStartWorkflowCreatesOneTaskForOneToolCall(t *testing.T) {
    first, err := starter.Start(ctx, WorkflowStartInput{RunID: "run-1", ToolCallID: "call-1", WorkflowID: "12", Inputs: map[string]any{"prompt": "rain"}})
    second, err := starter.Start(ctx, WorkflowStartInput{RunID: "run-1", ToolCallID: "call-1", WorkflowID: "12", Inputs: map[string]any{"prompt": "rain"}})
    require.NoError(t, err)
    require.Equal(t, first.TaskID, second.TaskID)
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/studio/application -run 'TestResolveWorkflows|TestStartWorkflow' -count=1`

Expected: compile failure because the catalog and starter do not exist.

- [ ] **Step 3: Implement the port without Bot Session reuse**

Define application-owned interfaces for Case lookup, Case input validation, task creation, blob staging, and `TaskCreated` publication. Convert `string`, `number`, `boolean`, `image`, and `video` values to Case `InputValue`; image/video values must resolve only owned Studio asset versions to a `BlobRef`. Reject missing required fields and unknown keys before creating a task. Create a task with a Studio-prefixed `SessionID` and empty ChatID; update the task completion notification path to safely skip channel notification when no ChatID exists.

- [ ] **Step 4: Wire real adapters at application startup**

Pass Case repository, task repository, validator, blob store, and the in-process bus publisher into the Studio workflow service. Keep all Bot `Facade` dependencies out of the Studio service.

- [ ] **Step 5: Run focused tests**

Run: `go test ./internal/studio/application ./apps/pixoma/internal/application -run 'TestResolveWorkflows|TestStartWorkflow|TestApplicationModelsIncludeStudioSchema' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/studio/application/workflow.go internal/studio/application/workflow_test.go internal/studio/application/capability_config.go apps/pixoma/internal/application/app.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): start enabled workflow tasks'
```

### Task 3: Expose dynamic workflow tools to Eino and create the operation Road node

**Files:**
- Create: `internal/studio/infrastructure/workflowtool/runtime.go`
- Create: `internal/studio/infrastructure/workflowtool/runtime_test.go`
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Modify: `internal/studio/infrastructure/einoagent/engine_test.go`

**Consumes:** `ResolvedWorkflow`, `WorkflowStarter`, `AgentSink`, `PermissionMode` and persisted approvals.

**Produces:** one Eino tool per enabled Case. Each tool validates model JSON against the Case-derived schema, requires approval for `workflow.execute:<workflow-id>:<canonical-input-hash>` outside full access, creates one operation Flow node, starts one Task, persists its execution mapping, and returns the task ID/status to the model.

- [ ] **Step 1: Write failing runtime tests**

```go
func TestWorkflowToolCreatesOperationAndStartsTask(t *testing.T) {
    result, err := tool.InvokableRun(ctx, `{"prompt":"rain"}`)
    require.NoError(t, err)
    require.Contains(t, result, "task-1")
    require.Equal(t, []string{studioapp.EventToolCallStart, studioapp.EventToolCallEnd}, sink.events)
    require.Equal(t, domain.FlowNodeOperation, sink.nodes[0].Type)
}

func TestWorkflowToolRejectsUnknownAsset(t *testing.T) {
    _, err := tool.InvokableRun(ctx, `{"image":"asset-other-account"}`)
    require.Error(t, err)
    require.Empty(t, starter.calls)
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/studio/infrastructure/workflowtool ./internal/studio/infrastructure/einoagent -run 'TestWorkflowTool|TestEngineInvokesWorkflow' -count=1`

Expected: package or symbols do not exist.

- [ ] **Step 3: Implement the Eino tool adapter**

Use a `studio_workflow_<case-id>` name, `ResolvedWorkflow.JSONSchema`, and only the Case display metadata in `schema.ToolInfo`. Canonicalize input before approval/idempotency lookup. Emit trace-safe IDs and field names only; never emit input values, asset blobs, or task secrets. The adapter must create the mapping before returning so retries resolve the same task.

- [ ] **Step 4: Register both MCP and workflow tools**

Extend `einoagent.Engine` with a workflow resolver/starter dependency and append its tools to `resolveTools`. Preserve MCP behavior and the existing approval authorizer. Add an engine test with an OpenAI-compatible model that calls `studio_workflow_12` and then produces a user-visible submission acknowledgement.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/studio/infrastructure/workflowtool ./internal/studio/infrastructure/einoagent -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/studio/infrastructure/workflowtool internal/studio/infrastructure/einoagent/engine.go internal/studio/infrastructure/einoagent/engine_test.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): expose workflows as Agent tools'
```

### Task 4: Reconcile completed tasks into Studio assets and Flow

**Files:**
- Create: `internal/studio/application/workflow_reconciler.go`
- Create: `internal/studio/application/workflow_reconciler_test.go`
- Modify: `internal/studio/application/runner.go`
- Modify: `apps/pixoma/internal/application/app.go`

**Consumes:** pending `WorkflowExecution` mappings, task statuses/outputs, existing Studio asset and flow persistence.

**Produces:** a bounded background reconciler that idempotently marks terminal workflow executions, creates output asset records with `AssetOriginWorkflow`, creates asset Flow nodes connected to the operation node, and emits `ASSET_CREATED`, `FLOW_UPDATED`, and terminal workflow trace events on the original Studio Run.

- [ ] **Step 1: Write failing reconciliation tests**

```go
func TestReconcileSucceededWorkflowAdoptsOutputsWithoutCopyingBlob(t *testing.T) {
    require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
    asset := mustSessionAsset(t, repo, "account-a", "session-a")
    require.Equal(t, task.Outputs[0].Blob.Key, asset.Versions[0].BlobKey)
    require.Equal(t, domain.AssetOriginWorkflow, asset.Origin)
}

func TestReconcileTerminalWorkflowIsIdempotent(t *testing.T) {
    require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
    require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
    require.Len(t, mustSessionAssets(t, repo, "account-a", "session-a"), 1)
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/studio/application -run 'TestReconcile.*Workflow' -count=1`

Expected: compile failure because no reconciler exists.

- [ ] **Step 3: Implement terminal adoption**

For a succeeded task, map each `OutputRef` MIME type to `AssetKind`, create an immutable `AssetVersion` referencing the same `BlobKey`, and create an `asset` Flow node after the existing operation node. For failed/cancelled tasks, mark the execution terminal and append one safe failure event; do not create an output asset. Preserve terminal idempotency with one transaction/state update.

- [ ] **Step 4: Start/recover the bounded monitor**

Add a ticker-owned reconciler lifecycle next to `BackgroundRunner` in application startup. Run one reconciliation at startup and then at a bounded interval. It must use background context, stop on application shutdown, and never hold an Agent worker while waiting for a Task.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/studio/application ./apps/pixoma/internal/application -run 'TestReconcile.*Workflow|TestApplicationModelsIncludeStudioSchema' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/studio/application/workflow_reconciler.go internal/studio/application/workflow_reconciler_test.go internal/studio/application/runner.go apps/pixoma/internal/application/app.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): adopt completed workflow outputs'
```

### Task 5: Verify API, AG-UI recovery, and the full Studio scenario

**Files:**
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: `internal/studio/application/mock_executor_test.go`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Consumes:** dynamic runtime tools and reconciled workflow assets.

**Produces:** proof that enabled workflow settings are executable rather than display-only, a task submission has a visible operation node, terminal output is visible after a snapshot refresh, and the existing mock comic scenario still creates Story Markdown plus a workflow image asset and connected asset Road.

- [ ] **Step 1: Write failing integration assertions**

```go
func TestStudioWorkflowTaskOutputAppearsInSnapshot(t *testing.T) {
    // Submit a model-selected enabled workflow, complete the task, reconcile,
    // then GET the Session snapshot as its owning account.
    require.Contains(t, snapshot.Body.String(), `"origin":"workflow"`)
    require.Contains(t, snapshot.Body.String(), `"type":"operation"`)
}
```

- [ ] **Step 2: Run test and verify RED**

Run: `go test ./internal/httpapi/studio -run TestStudioWorkflowTaskOutputAppearsInSnapshot -count=1`

Expected: FAIL before the bridge is fully wired.

- [ ] **Step 3: Verify front-end connected states**

Ensure the existing Flow and asset queries refresh from the server snapshot/event path after `ASSET_CREATED` and `FLOW_UPDATED`. Do not add mock UI data or new visual component primitives. Extend the source contract only when an actual connected recovery path is missing.

- [ ] **Step 4: Run the final verification suite**

Run:

```bash
go test ./...
pnpm --dir web/admin exec vitest run src/features/studio/studio-workspace.contract.test.ts src/lib/api/studio.test.ts
pnpm --dir web/admin build
git diff --check
```

Expected: all targeted Studio tests and build PASS. Record unrelated pre-existing frontend suite failures separately; do not modify unrelated dashboard topology code.

- [ ] **Step 5: Request code review and commit**

```bash
git add internal/httpapi/studio/handler_test.go internal/studio/application/mock_executor_test.go web/admin/src/features/studio/studio-workspace.contract.test.ts
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'test(studio): cover workflow tool asset recovery'
```

## Plan Self-Review

- Spec coverage: Tasks 1–4 cover dynamic Case exposure, approval/idempotency, existing task scheduling, asset adoption, Flow Road updates, and background recovery. Task 5 proves the user-visible scenario and guards the existing mock flow.
- No placeholders: all tasks name concrete files, interfaces, test commands, and expected test state.
- Type consistency: `WorkflowExecution` is the only durable binding; its `TaskID` is consumed by the reconciler and its `RunID`/`SessionID` are used for events, assets, and Road nodes.
