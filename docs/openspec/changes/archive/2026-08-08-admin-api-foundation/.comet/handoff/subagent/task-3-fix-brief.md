# Task 3 Fix Brief

## Finding (IMPORTANT)
`apps/bot/cmd/comfyui-bot/main.go` still opens DB via `filepath.Join(dataDir, "app.db")` and ignores `cfg.DatabaseDSN`.
README / design require bot and admin-api share the same `database_dsn`.

## Required fix
- When `cfg.DatabaseDSN` is non-empty, pass it to appboot as DSN.
- When empty, keep current default behavior (DATA_DIR/app.db) for backward compatibility.
- Prefer a small failing test if extractable; otherwise document why and still fix main.
- Do NOT implement Task 4 Probe Refresh.
- Do NOT check off plan checkboxes.

## Commit
`fix(bot): honor database_dsn for shared admin-api database`

## Report
DONE|... with RED/GREEN if tests, commit hash, confirm both processes can share DSN from yaml.
