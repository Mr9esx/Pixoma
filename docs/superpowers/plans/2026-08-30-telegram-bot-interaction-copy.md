# Telegram Bot 交互配置 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make workflow stage copy/buttons, task notifications, platform/session copy, and the two public commands configurable without changing the existing menu editor.

**Architecture:** Keep `text_templates` as the storage and fallback chain. Add group metadata to template specs, render all new workflow/user-notification copy through the existing text renderer, and group the admin data table by scenario. The existing menu/card editor remains unchanged.

**Tech Stack:** Go 1.x, GORM-backed text store, React 19, TanStack Table, shadcn/ui, i18next.

## Global Constraints

- Do not modify menu, card, or card-button data structures.
- Do not expose `/confirm`, `/skip`, or `/exit` as public commands.
- Rendering precedence remains channel override → platform default → built-in default.
- Empty overrides delete the row and fall back to the next level.
- The admin page is a single page with grouped tables; no tabs, sidebar, or Telegram preview pane.
- Preserve all current template defaults exactly.

---

### Task 1: Template model and grouped specs

**Files:**
- Modify: `internal/channel/text/text.go`
- Test: `internal/channel/text/text_test.go`

**Interfaces:**
- Produces: `Spec.Group string`
- Produces: group constants `GroupWorkflow`, `GroupNotifications`, `GroupPlatform`, `GroupCommands`
- Produces: new keys for workflow stage buttons, preview hints, validation copy, and session termination.

- [ ] Add failing tests for new keys, non-empty groups, and exact new defaults.
- [ ] Run `go test ./internal/channel/text -run 'TestSpecs' -v`; verify it fails because the fields/keys are missing.
- [ ] Extend `Spec` and all existing specs with `Group`.
- [ ] Add the new keys, defaults, descriptions, variables, and ordering.
- [ ] Run `go test ./internal/channel/text -v`; verify all tests pass.

### Task 2: Workflow capability rendering

**Files:**
- Modify: `internal/channel/capability/open_case.go`
- Test: `internal/channel/capability/open_case_test.go`

**Interfaces:**
- Consumes: `OpenCase.Texts` and `texttpl.Key*`.
- Produces: configurable preview text, start/skip/exit/confirm buttons, validation errors, and submit copy.

- [ ] Add failing tests proving preview, input, confirm, and submitted results read custom channel templates.
- [ ] Run `go test ./internal/channel/capability -run TestOpenCase -v`; verify the button/copy assertions fail.
- [ ] Replace the corresponding hard-coded strings with `o.renderText`.
- [ ] Run `go test ./internal/channel/capability -v`; verify tests pass.

### Task 3: Runtime and session notifications

**Files:**
- Modify: `internal/channel/tg/adapter.go`
- Test: `internal/channel/tg/notify_test.go`

**Interfaces:**
- Consumes: `tg.Adapter.renderText`.
- Produces: configurable session-terminated notification.

- [ ] Add a failing test for a custom session-terminated template.
- [ ] Run `go test ./internal/channel/tg -run TestHandleUserNotifySessionTerminated -v`; verify it fails.
- [ ] Render `session_terminated` through the text renderer.
- [ ] Run `go test ./internal/channel/tg -v`; verify tests pass.

### Task 4: Admin grouped configuration page

**Files:**
- Modify: `internal/httpapi/channeltext/handler.go`
- Modify: `web/admin/src/lib/api/text-templates.ts`
- Modify: `web/admin/src/features/text-templates/text-templates-editor.tsx`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Test: `internal/httpapi/channeltext/handler_test.go`
- Test: `web/admin/src/features/text-templates/text-templates-editor.test.tsx`

**Interfaces:**
- Consumes: `text.Specs()`.
- Produces: `TextTemplate.group` in the admin API.

- [ ] Add an API test asserting each DTO includes the expected group and new keys.
- [ ] Run `go test ./internal/httpapi/channeltext -v`; verify it fails.
- [ ] Expose `group` in the DTO and API type.
- [ ] Group the admin data table into workflow, notifications, platform, and commands sections.
- [ ] Add Chinese/English labels and group names.
- [ ] Add a browser test asserting all four groups render and the menu editor contract remains untouched.
- [ ] Run targeted Go and frontend tests; verify all pass.

### Task 5: Full verification

**Files:** No new files.

- [ ] Run `gofmt -w internal/channel internal/httpapi/channeltext`.
- [ ] Run `go test ./...`.
- [ ] Run `pnpm --dir web/admin test`.
- [ ] Run `pnpm --dir web/admin lint`.
- [ ] Run `git diff --check`.
