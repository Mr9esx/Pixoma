# 添加模型弹窗测试连接实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow administrators to test the current add-model form configuration with one minimal model message before saving it.

**Architecture:** Add an account-scoped transient model-test endpoint that reuses the existing model connection tester without persisting a model. Reuse one frontend payload builder for both create and test mutations, and render the test control/result in the dialog footer.

**Tech Stack:** Go/Chi, Studio application service, React, TanStack Query, shadcn/ui, Vitest.

## Global Constraints

- The test request MUST NOT create or update a model record.
- API keys MUST be used only for the provider request and sanitized from errors.
- Keep existing saved-model testing and save behavior unchanged.
- Use existing shadcn `Button`, `DialogFooter`, semantic status colors, and current Chinese copy style.

### Task 1: Lock backend transient-test behavior

**Files:**
- Modify: `internal/studio/application/model_config_test.go`
- Modify: `internal/httpapi/studio/handler_test.go`

- [ ] Add service coverage proving a transient input reaches the tester with its plaintext key but creates no repository row.
- [ ] Add HTTP coverage for `POST /models/test`, including account scoping through the request context and a successful latency response.
- [ ] Run the focused Go tests and observe the expected missing-method/route failures.

### Task 2: Implement the transient test API

**Files:**
- Modify: `internal/studio/application/model_config.go`
- Modify: `internal/httpapi/studio/handler.go`

- [ ] Add an input-based test method that validates protocol/address/model/key, constructs an in-memory resolved config, and reuses the existing timeout, tester, latency, and sanitization path.
- [ ] Mount `POST /models/test` before the `{modelID}` route and decode the existing create-model payload shape without setting `AccountID` from the client.
- [ ] Run backend focused tests and keep saved-model testing green.

### Task 3: Lock frontend API and dialog behavior

**Files:**
- Modify: `web/admin/src/lib/api/studio.test.ts`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

- [ ] Add API coverage for the transient test request body and endpoint.
- [ ] Add dialog contract assertions for the footer-left test button, loading/result copy, and shared form configuration.
- [ ] Run focused frontend tests and observe failures before implementation.

### Task 4: Implement the modal interaction

**Files:**
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/features/studio/studio-settings.tsx`

- [ ] Add the transient test client function and shared form payload mapper.
- [ ] Add native form validity gating, a TanStack mutation, stale-result clearing on edits, and footer-left result rendering.
- [ ] Preserve existing save mutation and close/reset behavior.

### Task 5: Verify

- [ ] Run `gofmt` and focused/full Go tests.
- [ ] Run focused frontend tests and a production frontend build.
- [ ] Inspect the diff for secret leakage, persistence side effects, and footer placement.
