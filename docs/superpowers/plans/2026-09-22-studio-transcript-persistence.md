# Studio Transcript Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make every new Studio conversation replayable from persisted data, including user and assistant text, reasoning, tool calls and arguments, tool results, approvals, assets, flow updates, run state, and their original order.

**Architecture:** `studio_events` becomes the canonical append-only transcript source. Every observable AG-UI event is emitted once with a complete, safe payload; a server-side transcript projector replays those events together with persisted user messages into assistant-ui-compatible messages for session history. The chat UI and Trace UI consume the same projected transcript, so they cannot drift or independently lose reasoning/tool state.

**Tech Stack:** Go application/domain layers, GORM persistence, existing Studio HTTP/AG-UI WebSocket transport, React + assistant-ui, React Query, Vitest, Go tests.

## Global Constraints

- Do not add one-off UI-only persistence for reasoning, tools, or loading state.
- Persist observable data at the event emission source before it is streamed to clients.
- Keep account/session/run ownership checks at the repository and HTTP boundaries.
- Do not persist model API keys, connector credentials, or other secrets; reuse the existing connector redaction rules for tool output.
- Preserve existing user-message and assistant-text API compatibility while adding the canonical transcript projection.
- Existing historical records that never stored reasoning/tool payloads cannot be reconstructed; the API must fall back to their available text/events without inventing data.

## File Map

- Create `internal/studio/application/transcript.go`: canonical transcript event payloads, replay/projector logic, and compatibility conversion.
- Create `internal/studio/application/transcript_test.go`: projector tests covering text, reasoning, tools, approvals, assets, flow, ordering, and legacy fallback.
- Modify `internal/studio/application/executor.go`: route all run output through the canonical event writer and preserve complete assistant/reasoning/tool state.
- Modify `internal/studio/application/service.go`: keep user messages compatible with the transcript projector and expose the session transcript read model.
- Modify `internal/studio/domain/model.go` and `internal/studio/domain/repository.go`: add the minimum run/session event read interfaces needed for ordered projection.
- Modify `internal/studio/infrastructure/persistence/gorm_repository.go` and tests: implement complete ordered session-event reads and any additive message metadata required by the projector.
- Modify `internal/studio/infrastructure/studiotool/runtime.go`, `internal/studio/infrastructure/mcpconnector/runtime.go`, and `internal/studio/infrastructure/workflowtool/runtime.go`: emit complete safe tool-call arguments/results rather than byte counts only.
- Modify `internal/httpapi/studio/handler.go`: return the projected transcript in session detail and preserve legacy `messages` for existing clients.
- Modify `internal/httpapi/studio/agui.go`: stream the exact persisted event payloads without reconstructing or dropping transcript fields.
- Modify `web/admin/src/lib/api/studio.ts`: type the transcript wire format.
- Modify `web/admin/src/features/studio/studio-chat.tsx`: hydrate assistant-ui from the projected transcript; remove the assumption that `initialMessages` is the history source.
- Modify `web/admin/src/features/studio/studio-trace.tsx`: consume the same transcript/event projection and stop presenting a second lossy interpretation.
- Modify `web/admin/src/features/studio/studio-workspace.contract.test.ts` and add focused frontend tests for transcript hydration and legacy fallback.

### Task 1: Define the canonical transcript contract

**Files:**
- Create: `internal/studio/application/transcript.go`
- Create: `internal/studio/application/transcript_test.go`

**Interfaces:**
- Produces `TranscriptEventPayload`, `ProjectSessionTranscript`, and a stable wire representation with ordered `messages` and `events`.
- The projector accepts persisted user messages, runs, and ordered events; it returns assistant-ui/AG-UI-compatible message records without depending on React.

- [ ] **Step 1: Write failing projector tests** for a run containing `TEXT_MESSAGE_*`, `REASONING_MESSAGE_*`, `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_END`, `TOOL_CALL_RESULT`, `APPROVAL_REQUIRED`, `APPROVAL_RESOLVED`, `ASSET_CREATED`, `FLOW_UPDATED`, and `RUN_FINISHED`.
- [ ] **Step 2: Run** `go test ./internal/studio/application -run Transcript -count=1` and verify failure because the projector types/functions do not exist.
- [ ] **Step 3: Implement the minimal canonical types and ordered replay state machine.** It must merge reasoning and tool parts into the relevant assistant message, attach tool results by `tool_call_id`, preserve event sequence, and retain unknown event types as custom transcript events instead of dropping them.
- [ ] **Step 4: Add legacy fallback tests** showing a session with only existing text messages still projects correctly and does not fabricate reasoning/tool content.
- [ ] **Step 5: Run the focused tests** and verify they pass.

### Task 2: Persist complete events at the output sources

**Files:**
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/infrastructure/studiotool/runtime.go`
- Modify: `internal/studio/infrastructure/mcpconnector/runtime.go`
- Modify: `internal/studio/infrastructure/workflowtool/runtime.go`
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Test: `internal/studio/application/transcript_test.go` and the existing tool runtime tests

**Interfaces:**
- Consumes the canonical event payload types from Task 1.
- Produces persisted events whose payloads contain the same fields sent to AG-UI clients.

- [ ] **Step 1: Add failing tests** asserting tool start events contain stable IDs, tool name, structured arguments, and safe connector/workflow metadata; tool end events contain result content and error state; reasoning events preserve every delta in order.
- [ ] **Step 2: Run the focused Go tests** and verify they fail because current events only contain `argument_bytes`/`result_bytes` and reasoning is not part of persisted messages.
- [ ] **Step 3: Update the event writer and runtimes.** Emit `TOOL_CALL_ARGS` and `TOOL_CALL_RESULT` with complete safe content; keep credentials redacted; do not duplicate cumulative reasoning chunks; preserve asset/version and flow IDs as structured references.
- [ ] **Step 4: Ensure the same event map is used by `streamAGUIWebSocket` and persistence**, so the WebSocket is a live view of the canonical event log rather than a separately reconstructed format.
- [ ] **Step 5: Run all affected Go tests** and verify payload and ordering assertions pass.

### Task 3: Add ordered session transcript reads

**Files:**
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:**
- Produces an account-scoped ordered read of all runs/events for a session with deterministic ordering; if the HTTP response needs pagination, the cursor is explicit and the frontend follows it until the transcript is complete.

- [ ] **Step 1: Write failing repository tests** for multiple runs, interleaved timestamps, event sequence gaps, account isolation, and reading every event without silently truncating the transcript.
- [ ] **Step 2: Run** `go test ./internal/studio/infrastructure/persistence -run SessionTranscript -count=1` and verify failure because no session-event read exists.
- [ ] **Step 3: Implement the repository method** using run ownership plus event sequence ordering; do not rely on timestamp ordering alone.
- [ ] **Step 4: Run the repository tests** and verify they pass.

### Task 4: Expose one session transcript API

**Files:**
- Modify: `internal/httpapi/studio/handler.go`
- Modify: `internal/httpapi/studio/agui.go`
- Modify: `internal/httpapi/studio/handler_test.go`

**Interfaces:**
- `GET /api/v1/studio/sessions/:sessionID` returns `messages` for compatibility plus `transcript` containing projected ordered messages/events.
- WebSocket live events and session history use identical event payload fields.

- [ ] **Step 1: Add failing HTTP tests** asserting reasoning, tool args/results, approval, asset, and flow records are present in the transcript and account isolation is enforced.
- [ ] **Step 2: Run** `go test ./internal/httpapi/studio -run Transcript -count=1` and verify failure because session detail only returns the legacy message list.
- [ ] **Step 3: Implement transcript projection in the handler** with complete reads (or an explicit cursor that the frontend follows until exhausted) and explicit legacy fallback.
- [ ] **Step 4: Add protocol tests** proving a live WebSocket event and the same event in a subsequent session GET have identical JSON payloads.
- [ ] **Step 5: Run all Studio HTTP tests** and verify they pass.

### Task 5: Hydrate assistant-ui from the projected transcript

**Files:**
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`
- Create: `web/admin/src/features/studio/studio-transcript.test.ts`

**Interfaces:**
- Consumes `StudioSessionDetail.transcript` from Task 4.
- Produces a single assistant-ui history repository containing user text, assistant text, reasoning parts, tool-call parts/results, and custom run events.

- [ ] **Step 1: Write failing frontend tests** for transcript-to-assistant-ui conversion, including reasoning and a tool call with a result, plus legacy messages-only fallback.
- [ ] **Step 2: Run** `pnpm exec vitest run src/features/studio/studio-transcript.test.ts` and verify failure because conversion ignores transcript records.
- [ ] **Step 3: Implement one conversion function** and use it for both initial history and post-run refresh; do not maintain separate reasoning/tool state in React.
- [ ] **Step 4: Keep Markdown rendering as a presentation concern only**; it must render text parts without filtering or replacing reasoning/tool parts.
- [ ] **Step 5: Run Studio frontend tests and `pnpm exec tsc -b --pretty false`** and verify they pass.

### Task 6: Make Trace and chat use the same read model

**Files:**
- Modify: `web/admin/src/features/studio/studio-trace.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- Both views consume the same transcript event semantics; Trace may choose a compact layout but may not discard fields required for chat replay.

- [ ] **Step 1: Add failing contract tests** asserting reasoning and tool event labels are present and no second lossy event mapping is used.
- [ ] **Step 2: Implement the shared read path** and remove redundant polling for completed historical sessions; keep polling only for an actively running run if needed.
- [ ] **Step 3: Run focused frontend tests and inspect the rendered data shape** for a completed run with tools and reasoning.

### Task 7: End-to-end verification and compatibility audit

**Files:**
- Modify: the files above only
- Test: Go and frontend test suites

- [ ] **Step 1: Run** `go test ./... -count=1`.
- [ ] **Step 2: Run** `pnpm exec vitest run src/features/studio` and `pnpm exec tsc -b --pretty false`.
- [ ] **Step 3: Run** `pnpm run build` and `git diff --check`.
- [ ] **Step 4: Verify manually with one new run** that includes reasoning, at least one tool call/result, an approval if applicable, and a generated asset; leave and reopen the session and compare the chat and Trace order.
- [ ] **Step 5: Document the legacy limitation** in the implementation notes: old runs retain the text that was stored, but missing historical reasoning/tool payloads cannot be recovered retroactively.
