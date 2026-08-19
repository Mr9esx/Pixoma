# TG 菜单/卡片重构 · 运行时（渲染 + 动作分发 + 返回链）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** TG 运行时按新模型工作：主键盘 = `Menu` 布局（无上限）；点菜单项/卡片按钮 → 按 `Action` 分发（打开卡片/工作流/文字/媒体/链接/复制/占位）；卡片自动生成「返回」按钮（来源链）。

**Architecture:** `menu_runtime.go` 提供新 `MenuReader`（返回 `domain.Menu`）与 `CardProvider`（按 id 取卡片）；`adapter.go` 的 `menuItemDispatch` 替换为 `actionDispatch`；返回链按 chat 内存缓存来源栈。

**Tech Stack:** Go、`go-telegram/bot/models`、现有 `capability.Registry`（open_workflow 复用 open_case 能力入口）。

## Global Constraints

- 测试命令：仓库根 `go test ./...`；构建 `go build ./...`。
- 主键盘无数量上限；列数 `Menu.Columns`（1-8，自由）。
- `open_card` 目标卡片不存在时返回可读错误（不崩溃）；`open_workflow` 复用现有 `open_case` 能力（列表模式）。
- 返回按钮由运行时生成（回来源卡片 / 回主菜单），不落库。
- 旧 `MenuNode` 树路径（`showGroup` / `FindEnabledRootByLabel` / `CBMenuFolder`）在本计划内删除；`menu_runtime.go` / `adapter.go` 若混有你在途改动，实现+测试后不中途提交，结尾统一归档。

---

### Task R1: 主键盘按新 Menu 渲染

**Files:**
- Modify: `internal/channel/tg/menu_runtime.go`
- Modify: `internal/channel/tg/menu_runtime_test.go`

**Interfaces:**
- Consumes: `domain.Menu`（新）
- Produces:
  - `type MenuReader interface { GetMenu(ctx context.Context) (domain.Menu, error) }`（替换旧 `MenuTree` 返回）
  - `func BuildReplyKeyboard(menu domain.Menu) *models.ReplyKeyboardMarkup`（按 `Columns` 分行的按钮网格；`Columns<=0` 时默认 2）
  - `func FindEnabledItemByLabel(menu domain.Menu, label string) (domain.MenuItem, bool)`

- [ ] **Step 1: 重写失败测试**

`menu_runtime_test.go` 的 `TestBuildReplyKeyboardColumns` 改为：

```go
func TestBuildReplyKeyboardFreeColumns(t *testing.T) {
	menu := domain.Menu{ID: "m", Name: "主", Columns: 3, Items: []domain.MenuItem{
		{ID: "a", Label: "A", Action: domain.Action{Type: "placeholder"}},
		{ID: "b", Label: "B", Action: domain.Action{Type: "placeholder"}},
		{ID: "c", Label: "C", Action: domain.Action{Type: "placeholder"}},
		{ID: "d", Label: "D", Action: domain.Action{Type: "placeholder"}},
		{ID: "e", Label: "E", Action: domain.Action{Type: "placeholder"}},
		{ID: "f", Label: "F", Action: domain.Action{Type: "placeholder"}},
		{ID: "g", Label: "G", Action: domain.Action{Type: "placeholder"}},
	}}
	kb := BuildReplyKeyboard(menu)
	if len(kb.Keyboard) != 3 { // 3+3+1，7 个按钮不设上限
		t.Fatalf("rows=%d kb=%+v", len(kb.Keyboard), kb.Keyboard)
	}
	if len(kb.Keyboard[0]) != 3 || len(kb.Keyboard[2]) != 1 {
		t.Fatalf("layout=%+v", kb.Keyboard)
	}
}

func TestFindEnabledItemByLabel(t *testing.T) {
	menu := domain.Menu{Items: []domain.MenuItem{
		{ID: "a", Label: "图片", Action: domain.Action{Type: "open_card", CardID: "c1"}},
	}}
	it, ok := FindEnabledItemByLabel(menu, "图片")
	if !ok || it.Action.CardID != "c1" {
		t.Fatalf("item=%+v ok=%v", it, ok)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/ -run 'TestBuildReplyKeyboardFreeColumns|TestFindEnabledItemByLabel'`
Expected: 编译失败（`domain.Menu` 无此签名）。

- [ ] **Step 3: 实现**

`menu_runtime.go` 中 `MenuReader` / `BuildReplyKeyboard` / `FindEnabledRootByLabel` 替换为新版本（`FindEnabledItemByLabel`）；删除 `BuildReplyKeyboard(tree domain.MenuTree)` 旧签名与 `CBMenuFolder` 相关旧逻辑可留到 R4。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/tg/`
Expected: PASS（若旧调用方编译失败，按 R4 一并替换或临时保留旧函数）。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/tg/menu_runtime.go internal/channel/tg/menu_runtime_test.go
git commit -m "feat(tg): render main keyboard from menu model"
```

---

### Task R2: 动作分发（actionDispatch）

**Files:**
- Modify: `internal/channel/tg/adapter.go`
- Modify: `internal/channel/tg/menu_runtime_test.go`

**Interfaces:**
- Consumes: `domain.MenuItem.Action` / `domain.CardButton.Action`、`CardProvider`
- Produces:
  - `type CardProvider interface { GetCard(ctx context.Context, channelID, cardID string) (domain.Card, error) }`
  - `func (a *Adapter) actionDispatch(ctx, chatID, addr, channelID string, action domain.Action, backID string) error`
  - 分支：`open_card` → `sendCard`（媒体+文字+按钮+自动返回）；`open_workflow` → 复用 open_case 能力（`inv.CapabilityID="open_case"; params.workflow_ids=...`，由能力侧转换为 `case_ids` 列表或直接列表）；`send_text` / `copy_text` → `SendText`；`send_media` → `SendMediaURL`；`open_url` → `SendMediaURL`（或发送带链接按钮的卡片）；`placeholder` → `SendText("暂未开放")`

- [ ] **Step 1: 写失败测试**

`menu_runtime_test.go` 追加：

```go
type cardProviderStub struct{ card domain.Card }

func (p cardProviderStub) GetCard(_ context.Context, _, _ string) (domain.Card, error) {
	return p.card, nil
}

func TestActionDispatchOpenCardSendsCard(t *testing.T) {
	out := &captureListOutbound{} // 记录 SendList 行（按钮）
	ad := New(out)
	ad.ChannelID = "ch1"
	ad.Cards = cardProviderStub{card: domain.Card{
		ID: "c1", Name: "x", Text: "选一种风格：",
		Buttons: []domain.CardButton{{ID: "b", Label: "写实", Action: domain.Action{Type: "open_workflow", WorkflowIDs: []string{"w1"}}}},
	}}
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, "ch1", domain.Action{Type: "open_card", CardID: "c1"}, "menu"); err != nil {
		t.Fatal(err)
	}
	if len(out.lists) != 1 {
		t.Fatalf("lists=%d", len(out.lists))
	}
	rows := out.lists[0]
	if rows[0][0].Text != "写实" {
		t.Fatalf("row0=%+v", rows[0])
	}
	// 最后一行是自动返回
	if rows[len(rows)-1][0].Text != "‹ 返回" {
		t.Fatalf("back=%+v", rows[len(rows)-1])
	}
}
```

（`captureListOutbound` 复用现有 stub 或新增，记录 `SendList` 的 `[][]ports.Button`；`Adapter` 新增 `Cards CardProvider` 字段。）

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/ -run TestActionDispatchOpenCardSendsCard`
Expected: 编译失败。

- [ ] **Step 3: 实现**

`adapter.go` 增加字段 `Cards CardProvider`；`actionDispatch` 实现各分支；`sendCard`：

```go
func (a *Adapter) sendCard(ctx context.Context, addr sharedkernel.ChannelAddr, card domain.Card, backID string) error {
	for _, m := range card.Media {
		if err := a.Out.SendMediaURL(ctx, addr, m.URL, card.Text); err != nil {
			return err
		}
	}
	if len(card.Buttons) == 0 && card.Text != "" {
		return a.Out.SendText(ctx, addr, card.Text)
	}
	rows := make([][]ports.Button, 0, len(card.Buttons)+1)
	for _, b := range card.Buttons {
		inv := a.cardButtonInvoke(b)
		rows = append(rows, []ports.Button{{Text: b.Label, Data: CBInvoke + a.store.put(inv)}})
	}
	rows = append(rows, []ports.Button{{Text: "‹ 返回", Data: CBMenuBack + backID}})
	return a.Out.SendList(ctx, addr, card.Text, rows)
}
```

`cardButtonInvoke` 把 `CardButton.Action` 编码进 invoke params（`action` JSON），点击回调后由 `menuItemDispatch` → `actionDispatch` 执行。`open_workflow` 分支：`inv.CapabilityID = "open_case"; inv.Params = {"step":"list","workflow_ids": action.WorkflowIDs}`（open_case 能力新增对 `workflow_ids` 的支持或转换为 `case_ids`——见 R4 说明）。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/tg/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/tg/adapter.go internal/channel/tg/menu_runtime_test.go
git commit -m "feat(tg): dispatch actions for menu and card buttons"
```

---

### Task R3: 返回链（来源栈 + 自动返回）

**Files:**
- Modify: `internal/channel/tg/adapter.go`
- Modify: `internal/channel/tg/menu_runtime_test.go`

**Interfaces:**
- Produces:
  - `type backStack struct{ mu sync.Mutex; stacks map[string][]string }`（chatID → 来源栈，栈顶=上一张卡片 id 或 `"menu"`）
  - `actionDispatch` 进入 `open_card` 前 `push(chatID, backID)`，发送卡片后栈顶为该卡片；处理 `back` 动作时 `pop` 并打开栈顶（或回菜单）
  - `"‹ 返回"` 按钮 `Data` 携带 `back` 动作

- [ ] **Step 1: 写失败测试**

```go
func TestBackChainPopsToPreviousCard(t *testing.T) {
	out := &captureListOutbound{}
	ad := New(out)
	ad.ChannelID = "ch1"
	ad.Cards = cardProviderStub{card: domain.Card{ID: "c2", Name: "x", Text: "第二张", Buttons: []domain.CardButton{}}}
	ad.back = &backStack{}
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	// 先打开 c1（返回 menu），再打开 c2（返回 c1）
	_ = ad.actionDispatch(context.Background(), "tg-default:1", addr, "ch1", domain.Action{Type: "open_card", CardID: "c1"}, "menu")
	_ = ad.actionDispatch(context.Background(), "tg-default:1", addr, "ch1", domain.Action{Type: "open_card", CardID: "c2"}, "c1")
	// 点返回 → 应回到 c1（SendList 的 back 按钮 Data=CBMenuBack+c1）
	// 断言 back 栈：ad.back.top("tg-default:1") == "c1"
	if got := ad.backTop("tg-default:1"); got != "c1" {
		t.Fatalf("top=%q", got)
	}
}
```

（`backTop` 为测试辅助导出，或测试直接访问同包字段。）

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/ -run TestBackChainPopsToPreviousCard`
Expected: FAIL（无 back 栈）。

- [ ] **Step 3: 实现**

`backStack` 结构 + `push/pop/top`；`actionDispatch` 中：打开卡片时 push 来源；收到 `back` 动作（invoke params `action.type=back`）时 pop 并 `sendCard` 上一张（或 `sendMainMenu` 当栈空/`"menu"`）。

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/tg/
git add internal/channel/tg/adapter.go internal/channel/tg/menu_runtime_test.go
git commit -m "feat(tg): card back chain with source stack"
```

---

### Task R4: 接线与清理旧模型路径

**Files:**
- Modify: `internal/channel/tg/adapter.go`、`internal/channel/tg/menu_runtime.go`
- Modify: `internal/channel/tg/messenger.go`（`MenuReader` 返回类型变化）
- Modify: `apps/pixoma/internal/app/telegram.go`（装配新 reader）

**Interfaces:**
- Produces: 删除 `showGroup` / `FindEnabledRootByLabel` / 旧 `MenuTree` 读取路径；`open_case` 能力支持 `workflow_ids`（在 `open_case.go` 的 `list` 分支：优先用 `params.workflow_ids`，缺省回退 `case_ids`）

- [ ] **Step 1: 写失败测试**

`open_case` 能力测试新增：`invoke {"step":"list","workflow_ids":["w1"]}` 返回列表使用 workflow_ids（在 `open_case_test.go` 追加，断言不报错且列表含 w1）。

- [ ] **Step 2: 实现**

```go
// open_case.go list()
ids := params["workflow_ids"]
if raw, ok := ids.([]any); ok && len(raw) > 0 {
	// 转 []string 使用
} else {
	ids = params["case_ids"]
}
```

`telegram.go` 的 menu reader 改为从 `CardRepository` 读取 `domain.Menu`；`adapter.go` 删除 `showGroup` 与旧 `CBMenuFolder` 回调分支；`messenger.go` 同步 `MenuReader` 接口。

- [ ] **Step 3: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/... ./internal/menu/...`
Expected: PASS。

- [ ] **Step 4: 提交（混合文件不提交）**

```bash
git add internal/channel/capability/open_case.go internal/channel/capability/open_case_test.go apps/pixoma/internal/app/telegram.go
git commit -m "feat(capability): open_case accepts workflow_ids"
```

> `adapter.go` / `menu_runtime.go` / `messenger.go` 若包含你在途改动，按约定不提交，结尾统一归档。

---

### Task R5: 全量验证

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma
go build ./...
go vet ./internal/channel/... ./internal/menu/...
go test ./...
```
Expected: 通过（除用户 WIP 区域既有的 edge-metrics 两个失败）。

- [ ] **Step 2: 提交遗留改动**

```bash
git add -A
git commit -m "chore: finalize menu/card runtime"
```
