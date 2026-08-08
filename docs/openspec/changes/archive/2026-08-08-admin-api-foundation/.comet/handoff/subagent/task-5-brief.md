# Task 5 Brief: 端到端验收与文档收尾

## Full plan task text

### Task 5: 端到端验收与文档收尾

**Files:**
- Modify: `README.md`、`apps/admin-api/README.md`、`configs/admin-api.example.yaml`
- Modify: `docs/openspec/changes/admin-api-foundation/tasks.md` — 勾选已完成项（NOTE: coordinator will check off OpenSpec; you may prepare evidence only — DO NOT leave tasks unchecked if you are told to check section 3; actually implementer must NOT check off — coordinator does that. You document evidence in report.)

Steps:
- Step 1: 手工或脚本 — 启动 admin-api，curl 列表/创建；打 bot 旧管理路径应失败
- Step 2: 确认 mock 下观测仍可用
- Step 3: 勾选 tasks.md 对应项 — **COORDINATOR ONLY; you report which OpenSpec items are done**
- Step 4: Commit — `docs(admin-api): document admin-api foundation rollout`

## Required verification evidence

1. Build both binaries
2. Start admin-api against a temp/shared sqlite DSN (or document commands run)
3. curl admin-api `/healthz` and `/api/v1/comfy-instances` list (and preferably create)
4. Prove bot no longer serves management API (start bot with empty TG token if needed, or unit/integration proof; at minimum show bot router has no comfy-instances mount via code grep + prior tests)
5. Mock observation path still works (handler tests or live curl system with mock)

Prefer automating a short script under `scripts/` ONLY if brief needs it; otherwise shell evidence in report is enough.

## Allowed
- README files, configs/admin-api.example.yaml, Makefile docs comments
- Optional small scripts for smoke
- Forbidden: new product features beyond docs/smoke

## TDD
Docs-heavy; still run relevant tests green. No need for fake RED on markdown.

Commit message: `docs(admin-api): document admin-api foundation rollout`

Report DONE with commands run and outcomes, commit hash, risk signals.
