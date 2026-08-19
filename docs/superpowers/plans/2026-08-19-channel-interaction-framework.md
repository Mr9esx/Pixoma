---
change: channel-interaction-framework
design-doc: docs/superpowers/specs/2026-08-19-channel-interaction-framework-design.md
base-ref: 2c32050a52105d30761ace6cd5ef4416b713fb50
---

# 渠道交互框架 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建渠道交互框架：能力注册表 + 统一交互协议（事件→能力→结果）+ 菜单改为能力入口树 + TG 适配器按协议渲染（主键盘显式≤6、一层分组、消息按钮流程）+ 管理端能力编辑器与同源预览；用户文案禁用内部术语。

**Architecture:** `internal/channel/capability`（Capability/Registry/JSON Schema/open_case）→ `internal/channel/protocol`（CapabilityInvoke/Result/Nav/AccountCtx）→ `internal/menu`（MenuNode 能力入口）→ `internal/channel/tg`（按协议渲染 + 预览 DTO 同源）→ 管理台（能力选择 + schema 表单 + 预览）。

**Tech Stack:** Go 1.24/1.25（GORM/SQLite）；React 18 + TanStack（shadcn-admin，pnpm）；go-telegram/bot；JSON Schema（jsonschema 库）。

## Global Constraints

- 模块名 `github.com/mr9esx/comfyui_tgbot`；测试 `GOTOOLCHAIN=go1.25.0 go test ./...`、前端 `cd web/admin && pnpm test && pnpm build`
- 能力/协议 MUST NOT import `internal/channel/tg`；适配器 MUST NOT 为具体业务写分支
- 主键盘直达入口 ≤6（管理员显式配置，平台不自动塞满）；分组最多一层（消息按钮内无嵌套）
- 菜单中立核心无平台专有字段；`params` 用 JSON Schema 校验；`render_override` 按 key 合并能力默认渲染声明
- 面向用户文案禁用 inline/callback/extras/capability 等内部术语
- 按渠道账户（channel + external_user_id），不做跨渠道合并
- open_case 行为等价保留（预览/填表/确认/出图）
- 每个任务结束提交；commit message 用 conventional 风格

---

### Task 1: 能力注册表与协议类型

**Files:**
- Create: `internal/channel/capability/capability.go`、`registry.go`、`open_case.go`、`capability_test.go`
- Create: `internal/channel/protocol/invoke.go`、`protocol_test.go`

**Interfaces:**
- Produces: `type Capability interface { ID() string; DisplayName() string; ParamsSchema() json.RawMessage; Render(channelID string, override map[string]any) (RenderDecl, error); Invoke(ctx context.Context, acct AccountCtx, nav Nav, params map[string]any) (Result, error) }`
- Produces: `RenderDecl{Entry string; Config map[string]any}`、`AccountCtx{ChannelID, ExternalUserID, InternalUserID}`、`Nav{Back string; Step string}`、`CapabilityInvoke{CapabilityID string; Params map[string]any; Account AccountCtx; Nav Nav}`、`Result{Text string; Options []Option; Media []MediaRef; Error *string}`、`Option{Label string; Value map[string]any}`
- Produces: `Registry.Register(cap Capability) error`（重复 id 报错）、`Registry.Get(id)`、`Registry.List()`

- [x] **Step 1: 写失败测试** — capability_test：注册/重复注册报错；Get/List；Render 合并 override（能力默认 `{columns:2}` + override `{columns:3}` → 3）；protocol_test：CapabilityInvoke JSON 往返、Nav/AccountCtx 字段
- [x] **Step 2: 运行确认失败** — `GOTOOLCHAIN=go1.25.0 go test ./internal/channel/capability/... ./internal/channel/protocol/...`
- [x] **Step 3: 实现** — 按接口写类型与 Registry；Render 合并语义（仅替换 override 指定 key）
- [x] **Step 4: 运行通过** — 测试全绿
- [x] **Step 5: 提交** — `git commit -m "feat(channel): capability registry and interaction protocol types"`

### Task 2: open_case 能力 + JSON Schema 校验

**Files:**
- Modify: `internal/channel/capability/open_case.go`、`registry.go`
- Add dependency: `github.com/santhosh-tekuri/jsonschema/v6`
- Test: `internal/channel/capability/open_case_test.go`、`registry_test.go`

**Interfaces:**
- Produces: `open_case` 能力：params schema `{case_ids: [string], back: string}`；`Invoke` 包装 `botapp.Facade`（case 列表/预览/开始/填表/确认/结果，沿用现有会话逻辑）
- Produces: `Registry.Invoke(ctx, inv CapabilityInvoke) (Result, error)`（先 JSON Schema 校验 params，再调用能力）

- [x] **Step 1: 写失败测试** — schema 校验：缺 case_ids 拒绝；非 string 数组拒绝；合法 params 通过；Invoke 分发到 Facade 流程
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/capability/...`
- [x] **Step 3: 实现** — open_case 包装 Facade（迁移 adapter 中 case 流程为能力）；Registry.Invoke 注入 schema 校验
- [x] **Step 4: 运行通过** — 测试全绿
- [x] **Step 5: 提交** — `git commit -m "feat(channel): open_case capability with JSON Schema validation"`

### Task 3: 菜单模型改能力入口

**Files:**
- Modify: `internal/menu/domain/document.go`、`validate.go`、`seed.go`
- Modify: `internal/menu/infrastructure/persistence/gorm_repository.go`、`gorm_repository_test.go`
- Test: `internal/menu/domain/validate_test.go`

**Interfaces:**
- Produces: `MenuNode` 增加 `CapabilityID string`、`Params map[string]any`、`RenderOverride map[string]any`；移除业务 `Kind` 枚举语义（保留展示字段 IntroText/PlaceholderText/Reply）
- Produces: 校验：未知 capability_id 拒绝（通过注入的 capability 存在性检查）、params 校验交给能力层、根层直达入口 ≤6、一层分组（分组节点 children 不得再含分组）

- [x] **Step 1: 写失败测试** — 根层 >6 拒绝；分组嵌套拒绝；未知 capability_id 拒绝；params/override 往返
- [x] **Step 2: 运行确认失败** — `go test ./internal/menu/...`
- [x] **Step 3: 实现** — 模型字段、校验、持久化（capability_id/params_json/render_override_json 列）、种子改能力入口
- [x] **Step 4: 运行通过** — 全绿；`go build ./internal/menu/...`
- [x] **Step 5: 提交** — `git commit -m "refactor(menu): capability-entry menu model"`

### Task 4: 交互协议接入适配器

**Files:**
- Modify: `internal/channel/ports/ports.go`（Action → CapabilityInvoke 泛化）
- Modify: `internal/channel/tg/adapter.go`、`callback.go`、`messenger.go`
- Test: `internal/channel/tg/callback_test.go`、`adapter_test.go`

**Interfaces:**
- Consumes: Task 1/2 协议与 registry；Task 3 菜单模型
- Produces: 适配器 UI 事件 → `CapabilityInvoke` → `Registry.Invoke` → `Result` → `SendList/SendText/SendMedia`；Nav 渲染返回/退出按钮

- [ ] **Step 1: 写失败测试** — 事件翻译：按钮 → CapabilityInvoke{open_case, params}；Nav 返回按钮生成（root/分组）；Result.Options → 消息按钮（encodeAction 改为 capability 载荷）
- [ ] **Step 2: 运行确认失败** — `go test ./internal/channel/tg/...`
- [ ] **Step 3: 实现** — ports.Action 泛化；adapter 改走 registry（移除 open_folder/open_case 等业务 switch）；Nav 统一渲染返回/退出
- [ ] **Step 4: 运行通过** — 全绿
- [ ] **Step 5: 提交** — `git commit -m "refactor(tg): route events through capability protocol"`

### Task 5: TG 交互落地（主键盘≤6 + 一层分组 + 消息按钮流程）

**Files:**
- Modify: `internal/channel/tg/menu_runtime.go`、`messenger.go`
- Test: `internal/channel/tg/menu_runtime_test.go`

- [x] **Step 1: 写失败测试** — 主键盘：根层能力入口 ≤6（超限由校验拒绝）；每行按钮数读渲染声明（默认 2，override 生效）；分组点击 → 消息按钮列出能力入口（无嵌套分组）；每步含返回/退出
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/tg/...`
- [x] **Step 3: 实现** — 主键盘构建读 render decl；分组一层；Nav 返回/退出按钮统一生成
- [x] **Step 4: 运行通过** — 全绿
- [x] **Step 5: 提交** — `git commit -m "feat(tg): explicit main keyboard and one-level grouping"`

### Task 6: 渲染 DTO（预览=真实，同源）

**Files:**
- Create: `internal/channel/protocol/render.go`、`render_test.go`
- Modify: `internal/httpapi/channelmenu/handler.go`（`GET /api/v1/channels/{id}/menu/preview`）
- Test: `internal/httpapi/channelmenu/handler_test.go`

**Interfaces:**
- Produces: `BuildKeyboardPreview(tree, decls) PreviewDTO{MainKeyboard [][]string; Groups map[string]GroupPreview}`；与 TG 真实渲染共用同一构建函数
- Produces: `GET /api/v1/channels/{id}/menu/preview` 返回 PreviewDTO

- [x] **Step 1: 写失败测试** — preview 与真实渲染同源（相同输入→相同结构）；API 返回 200 且结构正确
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/protocol/... ./internal/httpapi/channelmenu/...`
- [x] **Step 3: 实现** — 抽取共用构建函数；handler 挂预览端点
- [x] **Step 4: 运行通过** — 全绿
- [x] **Step 5: 提交** — `git commit -m "feat(channel): same-source render preview DTO"`

### Task 7: 管理台编辑器（能力 + JSON Schema 表单 + 预览）

**Files:**
- Modify: `web/admin/src/lib/api/channel-menu.ts`（MenuNode 加 capability_id/params/render_override；preview API）
- Modify: `web/admin/src/features/tg-menu/menu-editor.tsx`（kind 选择 → 能力下拉 + schema 表单 + 展示微调）
- Modify: `web/admin/src/features/tg-menu/extras-editor.tsx`（移除）
- Modify: `web/admin/src/lib/api/query-keys.ts`、`routes/_app/channels/$id.tsx`（预览 tab）
- Modify: i18n zh/en（直白文案；移除 extras 相关）

- [ ] **Step 1: 写契约测试** — 能力下拉来自 registry；schema 表单按 params 渲染；预览 DTO 绘制
- [ ] **Step 2: 运行确认失败** — `cd web/admin && pnpm test`
- [ ] **Step 3: 实现** — 编辑器改版 + 预览绘制 + 文案术语约束（无 inline/capability/extras 等词）
- [ ] **Step 4: 运行通过** — `pnpm test && pnpm build`；`rg -n "inline|capability|extras" web/admin/src` 仅代码/类型（无用户文案）
- [ ] **Step 5: 提交** — `git commit -m "feat(admin): capability editor with schema form and preview"`

### Task 8: 迁移与接线

**Files:**
- Modify: `apps/pixoma/cmd/pixoma/main.go`、`apps/pixoma/internal/app/telegram.go`、`apps/bot/cmd/comfyui-bot/main.go`
- Modify: `apps/admin-api/cmd/admin-api/main.go`

- [ ] **Step 1: 接线** — registry 注入应用；menu service 校验能力存在；channelmenu handler 挂 preview
- [ ] **Step 2: 迁移逻辑** — 旧 kind → 能力入口映射（folder→分组、open_case→open_case 能力、placeholder/reply→展示字段）；tg_root_layout extras → render_override
- [ ] **Step 3: 构建** — `GOTOOLCHAIN=go1.25.0 go build ./... && go test ./...` 全绿
- [ ] **Step 4: 冒烟** — pixoma 启动；`curl /api/v1/channels/<id>/menu/preview` 返回结构
- [ ] **Step 5: 提交** — `git commit -m "feat(app): wire capability registry and preview"`

### Task 9: 全量回归

- [ ] **Step 1: 后端** — `GOTOOLCHAIN=go1.25.0 go test ./...`、`go vet ./...` 全绿
- [ ] **Step 2: 前端** — `cd web/admin && pnpm test && pnpm build` 全绿
- [ ] **Step 3: 手工回归** — TG：主键盘显式 ≤6、一层分组、消息按钮流程、返回/退出、open_case 等价（预览/填表/确认/出图）；管理台预览与实际一致；文案无内部术语
- [ ] **Step 4: 勾选** — tasks.md 全部勾选并提交；`git commit -m "test(channel): full regression for interaction framework"`

---

## Self-Review

- Spec 覆盖：capability-registry→Task1/2；channel-interaction-protocol→Task1/4/6；channel-account-context→Task1（AccountCtx）；channel-menu-interaction→Task3/5；channel-menu-config→Task3；channel-tg→Task4/5；tg-menu-admin-api→Task6/8；admin-web-shell→Task7
- 类型一致：`CapabilityInvoke`/`Result`/`Nav`/`AccountCtx` 在 Task1 定义、Task2/4 消费；`MenuNode.CapabilityID` Task3 定义、Task4/7 消费；`PreviewDTO` Task6 定义、Task7 绘制
