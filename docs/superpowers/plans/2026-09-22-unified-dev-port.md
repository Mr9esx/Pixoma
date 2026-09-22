# Unified Development Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `make dev PORT=<port>` propagate one port to Pixoma and the Vite API proxy, while retaining `HTTP_ADDR` compatibility for direct binary runs.

**Architecture:** `PORT` is the single Make/dev input. The Makefile exports it and derives `HTTP_ADDR` only when the caller has not supplied an explicit address. Vite reads the inherited `HTTP_ADDR` and derives a loopback proxy target; Pixoma accepts `PORT` as a fallback when `HTTP_ADDR` is absent.

**Tech Stack:** GNU Make, Bash, Vite/TypeScript, Go, Vitest.

## Global Constraints

- Preserve explicit `HTTP_ADDR` and `PUBLIC_URL` overrides.
- Keep the existing Vite same-origin `/api` development flow.
- Update contract tests with the new dynamic configuration behavior.

### Task 1: Lock the unified port contract

**Files:**
- Modify: `web/admin/src/vite-proxy.contract.test.ts`
- Modify: `web/admin/src/dev-loop.contract.test.ts`
- Create: `apps/pixoma/internal/application/env_test.go`

- [x] Add assertions that Vite derives its target from `HTTP_ADDR`, dev uses `PORT`, and Go maps `PORT` to the server address.
- [x] Run focused tests and verify they fail against the current hard-coded configuration.

### Task 2: Implement propagation

**Files:**
- Modify: `Makefile`
- Modify: `scripts/dev.sh`
- Modify: `web/admin/vite.config.ts`
- Modify: `apps/pixoma/internal/application/app.go`

- [x] Export `PORT` and derived `HTTP_ADDR` from `make dev`.
- [x] Derive the banner URL from the effective address.
- [x] Derive Vite's loopback proxy target from the inherited effective address.
- [x] Add `PORT` fallback in Pixoma's environment options.

### Task 3: Verify all affected paths

- [x] Run focused frontend contract tests.
- [x] Run Go application tests.
- [x] Run formatting and inspect the final diff.
