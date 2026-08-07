# ComfyUI Telegram Bot

Go monorepo：DDD 限界上下文（Catalog / Conversation / Runtime / ChannelTG / Platform）。

## 快速开始

```bash
go test ./...
export TG_BOT_TOKEN=...          # 可选；不设则只起 healthz
export HTTP_ADDR=:8080
export COMFY_MOCK=1              # 使用 mock ComfyUI
go run ./apps/bot/cmd/comfyui-bot
```

健康检查：`GET /healthz`

## 本地命令（Phase1 文本协议）

- `/menu` `/cases`
- `/start_case text2img-demo`
- 按提示输入；`/skip` `/exit` `/confirm`

种子 Case：`configs/cases/text2img.example.json`

## 架构要点

- ConfirmRun → `task.created` → Orchestrator → `dispatch.*` → Actuator → `task.status` → `notify.user`
- Task 终态仅 Orchestrator `applyStatus` 写入
- Queue/Blob 一期：Memory + LocalFS

后台预留：`apps/admin-api`、`web/admin`（禁止依赖 `channel/tg`）。
