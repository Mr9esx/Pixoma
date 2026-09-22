# Studio WebSocket streaming, reasoning, and loading Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task by task.

**Goal:** Replace the Studio chat transport with WebSocket, deliver incremental AG-UI events, expose model reasoning when the configured provider returns it, and make the assistant loading state unambiguous.

**Architecture:** Keep the existing run queue and persisted event log as the source of truth. Add a WebSocket AG-UI adapter that consumes the same run events as the legacy HTTP stream, and implement a WebSocket-backed `AbstractAgent` on the admin side. Extend the model/agent boundary with streaming chunks and reasoning metadata so the executor emits AG-UI reasoning and text events without duplicating messages.

**Tech Stack:** Go, chi, gorilla/websocket, Eino, React, `@ag-ui/client`, `@assistant-ui/react-ag-ui`, assistant-ui primitives, Vitest.

## Global Constraints

- Preserve the existing SSE endpoint as a compatibility fallback while the Studio client moves to WebSocket.
- Do not add explanatory copy to forms or duplicate operation-result messages.
- Keep API keys out of logs and WebSocket URLs.
- Use the existing AG-UI event names and assistant-ui reasoning support.

## Tasks

1. Add a shared AG-UI event pump and authenticated `/api/v1/studio/agui/ws` endpoint; cover handshake, run start, incremental events, errors, and disconnects with Go tests.
2. Add a browser WebSocket `AbstractAgent` implementation and switch `StudioChat` from `HttpAgent` to it; enable Vite WebSocket proxying and cover the request/event contract with frontend tests.
3. Enable Eino streaming and provider SSE parsing for supported OpenAI-compatible Chat/Responses/Anthropic responses, preserving a safe non-streaming fallback for tool calls and unsupported streams.
4. Persist streamed assistant output once while emitting text deltas, map reasoning events to AG-UI, and render reasoning plus an explicit sending indicator in the Studio thread.
5. Run focused Go/TypeScript tests, typecheck/build, inspect the diff, and run `git diff --check`.

## Verification

- `go test ./internal/httpapi/studio ./internal/studio/application ./internal/studio/infrastructure/...`
- `pnpm --dir web/admin test --run`
- `pnpm --dir web/admin build`
- `git diff --check`
