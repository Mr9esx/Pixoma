# Explicit Routing Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove implicit default-topic fallback and require explicit routing rules, starting from a default unconditional rule.

**Architecture:** Extend the condition engine with an independent `{always:true}` branch. Make empty/unmatched routing a terminal task failure. Migrate legacy empty routing at startup and update the admin editor/readiness model to treat all routing as explicit.

**Tech Stack:** Go, GORM, React, TypeScript, Vitest, i18next.

## Global Constraints

- No implicit fallback to `default` when rules are empty or unmatched.
- `{always:true}` must not mix with leaf fields or `and` / `or`.
- Existing Cases with no routing migrate to `{when:{always:true},topic:"default"}`.
- Keep transient condition-provider failures pending/retryable.
- Use `apply_patch` for edits; do not commit; current branch is `feat-init`.
- UI copy follows `docs/voice-profile.md` and uses `Alert` / semantic components where appropriate.

---

### Task 1: Go Condition Support

**Files:**
- Modify: `internal/runtime/domain/condition/rule.go`
- Modify: `internal/runtime/domain/condition/validate.go`
- Modify: `internal/runtime/domain/condition/evaluate.go`
- Test: `internal/runtime/domain/condition/rule_test.go`
- Test: `internal/runtime/domain/condition/evaluate_test.go`

**Interfaces:**
- Produces: `Rule.Always bool`; accepts and validates `{"always":true}`; rejects mixed and `{"always":false}` conditions.

- [ ] Add failing tests for parsing, evaluating, and validating `always`.
- [ ] Run condition tests and verify new cases fail.
- [ ] Add `Always` support with strict parsing and evaluation.
- [ ] Run `go test ./internal/runtime/domain/condition`.

### Task 2: Explicit Routing Resolution

**Files:**
- Modify: `internal/runtime/application/routing/router.go`
- Test: `internal/runtime/application/routing/router_test.go`

**Interfaces:**
- Produces: `routing.ErrNoMatch`.

- [ ] Replace fallback tests with `ErrNoMatch` tests for nil routing, empty rules, and unmatched rules.
- [ ] Add an `always` first-match test.
- [ ] Run `go test ./internal/runtime/application/routing`.
- [ ] Remove implicit fallback and return `ErrNoMatch`.

### Task 3: Terminal Routing Failure

**Files:**
- Modify: `internal/sharedkernel/ids.go`
- Modify: `internal/runtime/application/orchestrator/service.go`
- Test: `internal/runtime/application/orchestrator/topic_dispatch_test.go`

**Interfaces:**
- Consumes: `routing.ErrNoMatch`.
- Produces: failed task with `routing_no_match` / `未命中路由规则`.

- [ ] Add failing tests for no rules and unmatched rules.
- [ ] Run topic dispatch tests and verify failures.
- [ ] Map `routing.ErrNoMatch` to terminal failure and notification.
- [ ] Run `go test ./internal/runtime/application/orchestrator`.

### Task 4: Routing Validation

**Files:**
- Modify: `internal/catalog/infrastructure/validation/routing.go`
- Test: `internal/catalog/infrastructure/validation/routing_test.go`

**Interfaces:**
- Consumes: condition `always` support.

- [ ] Change validation tests to require non-nil, non-empty routing and accept `always`.
- [ ] Run validation tests and verify failures.
- [ ] Implement routing cardinality validation.
- [ ] Run `go test ./internal/catalog/infrastructure/validation`.

### Task 5: No Legacy Startup Migration

- [ ] Confirm `routing` validation rejects missing or empty rules.
- [ ] Do not add startup migration or implicit default routing.

### Task 6: Frontend Explicit Rules

**Files:**
- Modify: `web/admin/src/features/task-flow/types.ts`
- Modify: `web/admin/src/features/task-flow/lib/rule-operations.ts`
- Modify: `web/admin/src/features/task-flow/lib/validate.ts`
- Modify: `web/admin/src/features/task-flow/condition-form.tsx`
- Modify: `web/admin/src/features/task-flow/task-flow-table.tsx`
- Modify: `web/admin/src/features/cases/empty-case.ts`
- Test: `web/admin/src/features/task-flow/lib/validate.test.ts`

**Interfaces:**
- Produces: `AlwaysCondition`, default empty-case routing, unconditional toggle, and empty-routing validation failure.

- [ ] Add failing tests for `always` display/validation and empty-rule rejection.
- [ ] Run task-flow validation tests and verify failures.
- [ ] Add types, display, validation, unconditional toggle, empty-case default, and footer copy.
- [ ] Run `pnpm exec vitest run src/features/task-flow/lib/validate.test.ts`.

### Task 7: Frontend Readiness And Payload

**Files:**
- Modify: `web/admin/src/features/quick-config/lib/readiness.ts`
- Modify: `web/admin/src/features/quick-config/done-screen.tsx`
- Test: `web/admin/src/features/quick-config/lib/readiness.test.ts`

**Interfaces:**
- Consumes: explicit rules; no synthetic `default`.

- [ ] Add failing tests showing empty routing is gap and only rule topics count.
- [ ] Run readiness tests and verify failures.
- [ ] Update readiness and completion-topic calculation.
- [ ] Run `pnpm exec vitest run src/features/quick-config/lib/readiness.test.ts`.

### Task 8: Full Verification

**Files:**
- No new files.

- [ ] Run `go test ./...`.
- [ ] Run `pnpm --dir web/admin exec prettier --check src/features/task-flow src/features/quick-config src/features/cases/empty-case.ts`.
- [ ] Run `pnpm --dir web/admin lint`.
- [ ] Run focused frontend tests.
- [ ] Re-run `git diff --check`.
- [ ] Compare implementation against the design document and report any unrelated pre-existing failures.
