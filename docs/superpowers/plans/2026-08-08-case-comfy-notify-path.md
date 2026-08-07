# case-comfy-notify-path 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

---
change: case-comfy-notify-path
design-doc: docs/superpowers/specs/2026-08-08-case-comfy-notify-path-design.md
base-ref: eba6a71755051501e9c96760be288af3aedbd011
---

**Goal:** 打通 Case → 文本与用户图片输入 → Comfy 执行工作流 → 产物图 → 通知用户（Mock 与真实 Comfy 同一路径）。

**Architecture:** 执行时按任务查 Case，深拷贝工作流 JSON，读已存输入；用户图片先 Upload 再写入节点；`comfy_mock` 开关选型客户端；TG 按字段类型收文本/用户图片。

**Tech Stack:** Go、现有 actuator/comfyui/tg/botapp、本地对象存储、内存队列。

## Global Constraints

- 产物语言：zh-CN（文档）；代码与测试保持仓库现有英文标识符风格
- 保持 `comfy_mock` / `COMFY_MOCK`；功能与 Mock 同步改
- 不落盘 Session/Task；不做视频采集
- 说话与注释避免歧义词；预置 Case 用工作流 `klein9b-edit.api.json`

## 文件结构

| 文件 | 职责 |
|---|---|
| `internal/runtime/infrastructure/actuator/snapshot.go` | 按任务复制工作流并写入输入 |
| `internal/runtime/infrastructure/comfyui/*.go` | UploadImage；Mock/HTTP |
| `apps/bot/cmd/comfyui-bot/main.go` | 接线 |
| `configs/workflows/klein9b-edit.api.json` | 预置工作流 |
| `configs/cases/*.json` | 预置 Case（含绑定） |
| `internal/channel/tg/*` | 收用户图片、解析数字、发产物图文件名 |

---

## 任务 1：按任务准备可提交工作流

**Files:**
- Create: `internal/runtime/infrastructure/actuator/snapshot.go`
- Test: `internal/runtime/infrastructure/actuator/snapshot_test.go`

- [x] 1.1 写失败测试：已存文本写入节点 `inputs.text`；缺绑定时返回错误
- [x] 1.2 跑测试确认失败
- [x] 1.3 实现：查任务/Case、深拷贝工作流、读输入前缀下文件、按绑定写入；空工作流/缺绑定失败
- [x] 1.4 测试通过
- [x] 1.5 提交

## 任务 2：UploadImage 与用户图片写入

**Files:**
- Modify: `internal/runtime/infrastructure/comfyui/client.go`（或拆出 interface）
- Modify: `internal/runtime/infrastructure/comfyui/http.go`
- Modify: `internal/runtime/infrastructure/comfyui/client_select.go` 如需
- Modify: `internal/runtime/infrastructure/actuator/snapshot.go`（或 worker 注入前调用）
- Test: comfyui + actuator 相关测试

- [x] 2.1 写失败测试：HTTP/Mock UploadImage；写入用户图片字段时调用 Upload
- [x] 2.2 跑测试确认失败
- [x] 2.3 实现 UploadImage；注入路径对 image 类型先上传再写 filename
- [x] 2.4 测试通过；确认 `NewClient` 选型仍受 `comfy_mock` 控制
- [x] 2.5 提交

## 任务 3：main 接线

**Files:**
- Modify: `apps/bot/cmd/comfyui-bot/main.go`

- [x] 3.1 将执行器的工作流提供方接到「按任务查 Case」实现，传入 Tasks/Cases/Blob
- [x] 3.2 编译/`go test` 相关包通过
- [x] 3.3 提交

## 任务 4：预置 Case 与工作流文件

**Files:**
- Create: `configs/workflows/klein9b-edit.api.json`
- Create/Modify: `configs/cases/` 下文本+用户图片 Case
- Modify: seed 加载如需读外部工作流文件

- [x] 4.1 从已确认来源放入工作流 API JSON
- [x] 4.2 新增/更新 Case：输入含用户图片+文本；绑定节点 10/20；`input_schema` 拒绝图片字段纯文本
- [x] 4.3 校验相关测试通过
- [x] 4.4 提交

## 任务 5：TG 收用户图片与发产物图

**Files:**
- Modify: `internal/channel/tg/bot.go`、`adapter.go`
- Test: `adapter_test.go` 等

- [ ] 5.1 写失败测试：当前字段为 image 时收 Photo；number 解析；SendPhoto 文件名无 `/`
- [ ] 5.2 跑测试确认失败
- [ ] 5.3 实现 Photo/Document 处理、标量解析、安全文件名
- [ ] 5.4 测试通过
- [ ] 5.5 提交

## 任务 6：联调与回归

**Files:**
- Modify: `test/integration/smoke_test.go`（或新增）

- [ ] 6.1 冒烟：ConfirmRun → 可观测写入 → Mock 成功 → 通知（含用户图片输入）
- [ ] 6.2 `go test` 相关包全绿
- [ ] 6.3 文档/配置示例：关 Mock 时 `comfyui_base_url` 指向可达 Comfy
- [ ] 6.4 提交
