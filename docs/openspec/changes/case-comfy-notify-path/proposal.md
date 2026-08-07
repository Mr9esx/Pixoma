## Why

一期骨架虽已有 Case→ConfirmRun→Mock→通知冒烟，但执行面仍用空 `StaticWorkflows`，不加载 Case `bindings.workflow`、不注入已物化输入；TG 只收文本，协议声明的 `image` 输入无法采集。结果是「开始 Case → 输入 → Comfy 执行 → 取产物 → 完成 → 通知」在真实语义与真实 Comfy 下走不通，图片输入更是断链。

## What Changes

- Actuator 按 Task→Case 加载 workflow 模板，按 `bindings.inputs` 注入文本与图片（图片先入 Blob，再按绑定提交给 Comfy；Mock 与真实 HTTP 共用同一注入路径）
- 保持 `comfy_mock` / `COMFY_MOCK` 一键开关；Mock 开时端到端成功，关且 Comfy 可达时走真实 Submit/Wait
- Case 协议与种子：明确支持「文本 + 图片」混合输入的 Case（含校验、绑定与至少一个可跑种子）
- TG：当前字段为 `image` 时接收 Photo（及必要的 document/图片文件），写入 Session Draft 的 Blob；`number`/`boolean` 按字段类型解析；发图文件名不含路径分隔符
- 补齐接线与回归：集成冒烟覆盖「文本+图片 → 执行 → 通知」；映射缺失/空图在调用 Comfy 前失败上报

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `workflow-protocol`: 强化文本+图片混合 Case 的协议/校验与绑定约定（image 输入物化后可被执行面消费）
- `comfyui-executor`: 从 Case 快照加载图并注入文本/图片；Mock 与真实客户端共用；禁止空 stub 冒充成功路径
- `channel-tg`: 采集图片输入、按类型解析标量、可靠投递产物图
- `dialog-session`: Session 草稿支持图片 Blob 值进入后续 ConfirmRun 物化

## Impact

- 代码：`actuator`（CaseSnapshot/注入）、`comfyui`（必要时图片上传）、`main` 接线、`channel/tg`、Case 种子 JSON、集成测试
- 配置：`comfy_mock`、`comfyui_base_url`、真实 Comfy 可用的 workflow fixture/种子
- 非目标：Session/Task 持久化、视频输入采集、多实例调度、管理后台
