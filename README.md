# ComfyUI Telegram Bot

Go monorepo：DDD 限界上下文（Catalog / Conversation / Runtime / ChannelTG / Platform）。

## 跑通 TG 对话（默认 mock）

```bash
export TG_BOT_TOKEN=你的BotToken
go run ./apps/bot/cmd/comfyui-bot
```

配置：`configs/bot.yaml`（可从 `configs/bot.example.yaml` 复制）。**一键关 mock：**

```yaml
comfy_mock: false
comfyui_base_url: "http://127.0.0.1:8188"
```

或环境变量：`COMFY_MOCK=0`（`1`/`true` 开 mock）。

在 Telegram 里：

1. `/start` → 弹出主菜单（图片 / 视频 / 充值…，与产品图一致）
2. 点 **🖼 图片** → 列出 Case（二次元 / 写真 / Logo）
3. 点某个 Case → **预览** → **开始 Case**
4. 输入 prompt（可选字段可「跳过」）→ **确认生成**
5. 完成后 Bot 发回图片（mock 模式为示例 PNG）

种子 Case：`configs/cases/*.json`（启动自动导入/更新）。

## 其它

```bash
go test ./...
curl localhost:8080/healthz
```

事件链：ConfirmRun → `task.created` → Orchestrator → `dispatch.*` → Actuator → `task.status` → 发图。
