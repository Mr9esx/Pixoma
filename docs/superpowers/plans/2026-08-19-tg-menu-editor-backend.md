# TG 菜单编辑器后端 + 协议 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让菜单编辑器完全能力驱动：内置展示能力（提示文字/回复图文）、管理 API 暴露参数 schema、菜单校验补「叶子必须有能力」、旧数据读取时兼容转换、TG 运行时统一走能力调用。

**Architecture:** 五层改动，各自可测：协议（`protocol.Result` 增加 URL 媒体直发）、能力注册表（新增 `reply_text` / `reply_media` + 可选 `AdminParamsSchema` 接口）、管理 API（`ListCapabilities` 返回 `params_schema`）、菜单领域校验与默认种子、TG 运行时（`menuItemDispatch` 统一走能力）。

**Tech Stack:** Go（module `github.com/mr9esx/comfyui_tgbot`，GOTOOLCHAIN=go1.25.0）、chi、JSON Schema（santhosh-tekuri/jsonschema/v6）、标准库测试。

## Global Constraints

- 测试命令：仓库根目录 `go test ./...`；构建 `go build ./...`；格式 `gofmt -w <files>`。
- 不破坏既有 `ParamsSchema` 运行时校验语义（运行期参数仍用完整 schema 校验）。
- 管理端只暴露 `AdminParamsSchema`（open_case 仅 `case_ids`），运行期参数（`step`/`text`/`blob` 等）不得出现在管理 API。
- 旧数据（`placeholder_text` / `reply`）在 `application.Service.Get` 读取路径一次性转换为能力条目；Put 后落库为新形态。
- 默认种子树改为能力条目（占位按钮用 `reply_text`），不得出现「无能力的叶子」。
- 用户文案沿用「工作流」术语（open_case 展示名「打开工作流」）。

---

### Task B1: protocol.Result 支持 URL 媒体直发

**Files:**
- Modify: `internal/channel/protocol/invoke.go`
- Modify: `internal/channel/tg/adapter.go`（`renderResult`）
- Modify: `internal/channel/tg/menu_runtime_test.go`（或新增 `protocol_test.go`）

**Interfaces:**
- Consumes: 无
- Produces: `protocol.Result` 新增 `MediaURLs []string \`json:"media_urls,omitempty"\``；`renderResult` 在 Media 之后、Options 之前发送 `MediaURLs`（用 `a.Out.SendMediaURL(ctx, addr, url, res.Text)`）

- [ ] **Step 1: 写失败测试**

`internal/channel/tg/menu_runtime_test.go` 追加（`package tg` 内部测试，直接调 `ad.renderResult`）：

```go
type mediaURLOutbound struct {
	mediaURLs []string
}

func (m *mediaURLOutbound) SendText(context.Context, sharedkernel.ChannelAddr, string) error { return nil }
func (m *mediaURLOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []ports.MenuEntry) error { return nil }
func (m *mediaURLOutbound) SendList(context.Context, sharedkernel.ChannelAddr, string, [][]ports.Button) error { return nil }
func (m *mediaURLOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string) error { return nil }
func (m *mediaURLOutbound) SendMediaURL(_ context.Context, _ sharedkernel.ChannelAddr, imageURL, _ string) error {
	m.mediaURLs = append(m.mediaURLs, imageURL)
	return nil
}

func TestRenderResultSendsMediaURLs(t *testing.T) {
	out := &mediaURLOutbound{}
	ad := New(out)
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	res := protocol.Result{Text: "联系方式", MediaURLs: []string{"https://a/qr.png", "https://a/b.png"}}
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", protocol.CapabilityInvoke{}, res); err != nil {
		t.Fatal(err)
	}
	if len(out.mediaURLs) != 2 || out.mediaURLs[0] != "https://a/qr.png" {
		t.Fatalf("mediaURLs=%q", out.mediaURLs)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/ -run TestRenderResultSendsMediaURLs`
Expected: 编译失败（`protocol.Result` 无 `MediaURLs`）。

- [ ] **Step 3: 实现**

`internal/channel/protocol/invoke.go` 的 `Result` 增加：

```go
MediaURLs []string `json:"media_urls,omitempty"`
```

`internal/channel/tg/adapter.go` 的 `renderResult` 中，在 `for _, m := range res.Media` 循环之后、Options 判断之前插入：

```go
for _, u := range res.MediaURLs {
	if err := a.Out.SendMediaURL(ctx, addr, u, res.Text); err != nil {
		return err
	}
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/protocol/invoke.go internal/channel/tg/adapter.go internal/channel/tg/menu_runtime_test.go
git commit -m "feat(protocol): support url media in capability result"
```

---

### Task B2: 内置展示能力 reply_text / reply_media + AdminParamsSchema

**Files:**
- Modify: `internal/channel/capability/capability.go`（可选接口 `AdminSchemaProvider`）
- Create: `internal/channel/capability/display.go`
- Create: `internal/channel/capability/display_test.go`
- Modify: `internal/channel/capability/open_case.go`（`DisplayName` 改「打开工作流」+ 实现 `AdminParamsSchema`）
- Modify: `apps/admin-api/cmd/admin-api/main.go`、`apps/pixoma/internal/app/telegram.go`（注册新能力）

**Interfaces:**
- Consumes: `protocol.Result`（含 `MediaURLs`）
- Produces:
  - `type AdminSchemaProvider interface { AdminParamsSchema() json.RawMessage }`
  - `func AdminSchema(c Capability) json.RawMessage`：实现 `AdminSchemaProvider` 则返回其值，否则返回 `c.ParamsSchema()`
  - `type ReplyText struct{}`：`ID()="reply_text"`、`DisplayName()="提示文字"`、schema `{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`；`Invoke` 返回 `protocol.Result{Text: params["text"].(string)}`
  - `type ReplyMedia struct{}`：`ID()="reply_media"`、`DisplayName()="回复图文"`、schema `{"type":"object","properties":{"text":{"type":"string"},"images":{"type":"array","items":{"type":"string"}}},"required":["text"]}`；`Invoke` 返回 `protocol.Result{Text, MediaURLs: images}`
  - `OpenCase.DisplayName()` 返回「打开工作流」；`OpenCase.AdminParamsSchema()` 返回仅 `case_ids` 的 schema

- [ ] **Step 1: 写失败测试**

创建 `display_test.go`：

```go
package capability_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestReplyTextInvokeReturnsText(t *testing.T) {
	res, err := (capability.ReplyText{}).Invoke(context.Background(), protocol.AccountCtx{}, protocol.Nav{}, "tg:1", map[string]any{"text": "即将上线"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "即将上线" || len(res.Options) != 0 {
		t.Fatalf("res=%+v", res)
	}
}

func TestReplyMediaInvokeReturnsMediaURLs(t *testing.T) {
	res, err := (capability.ReplyMedia{}).Invoke(context.Background(), protocol.AccountCtx{}, protocol.Nav{}, "tg:1", map[string]any{
		"text":   "联系方式",
		"images": []any{"https://a/qr.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "联系方式" || len(res.MediaURLs) != 1 || res.MediaURLs[0] != "https://a/qr.png" {
		t.Fatalf("res=%+v", res)
	}
}

func TestAdminSchemaReturnsOpenCaseCaseIDsOnly(t *testing.T) {
	schema := capability.AdminSchema(capability.OpenCase{})
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatal(err)
	}
	props, _ := doc["properties"].(map[string]any)
	if _, ok := props["case_ids"]; !ok {
		t.Fatalf("case_ids missing: %v", props)
	}
	if _, ok := props["step"]; ok {
		t.Fatalf("step must not be admin-configurable: %v", props)
	}
}

func TestAdminSchemaFallsBackToParamsSchema(t *testing.T) {
	if got := capability.AdminSchema(stubCap{}); string(got) != `{"type":"object"}` {
		t.Fatalf("fallback schema=%s", got)
	}
}

type stubCap struct{}

func (stubCap) ID() string                                      { return "stub" }
func (stubCap) DisplayName() string                             { return "stub" }
func (stubCap) ParamsSchema() json.RawMessage                   { return json.RawMessage(`{"type":"object"}`) }
func (stubCap) Render(string, map[string]any) (protocol.RenderDecl, error) { return protocol.RenderDecl{}, nil }
func (stubCap) Invoke(context.Context, protocol.AccountCtx, protocol.Nav, sharedkernel.ChatID, map[string]any) (protocol.Result, error) {
	return protocol.Result{}, nil
}
```

同时修改 `internal/channel/capability/capability_test.go` 里现有 stub（若它实现 `Capability` 接口，增加 `AdminParamsSchema` 无需改动——可选接口不破坏）。

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/capability/`
Expected: 编译失败（`ReplyText` undefined）。

- [ ] **Step 3: 实现**

`capability.go` 追加：

```go
// AdminSchemaProvider lets a capability expose a reduced schema for admin
// configuration, hiding runtime-only params from the editor.
type AdminSchemaProvider interface {
	AdminParamsSchema() json.RawMessage
}

// AdminSchema returns the admin-editable params schema for a capability.
func AdminSchema(c Capability) json.RawMessage {
	if p, ok := c.(AdminSchemaProvider); ok {
		return p.AdminParamsSchema()
	}
	return c.ParamsSchema()
}
```

创建 `display.go`：

```go
package capability

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ReplyText replies with a fixed text (placeholder-style button).
type ReplyText struct{}

func (ReplyText) ID() string          { return "reply_text" }
func (ReplyText) DisplayName() string { return "提示文字" }
func (ReplyText) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","x-admin":{"widget":"text"}}},"required":["text"]}`)
}
func (ReplyText) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{Entry: "message_button"}, nil
}
func (ReplyText) Invoke(_ context.Context, _ protocol.AccountCtx, _ protocol.Nav, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	text, _ := params["text"].(string)
	return protocol.Result{Text: text}, nil
}

// ReplyMedia replies with text plus image URLs.
type ReplyMedia struct{}

func (ReplyMedia) ID() string          { return "reply_media" }
func (ReplyMedia) DisplayName() string { return "回复图文" }
func (ReplyMedia) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","x-admin":{"widget":"text"}},"images":{"type":"array","items":{"type":"string"}}},"required":["text"]}`)
}
func (ReplyMedia) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{Entry: "message_button"}, nil
}
func (ReplyMedia) Invoke(_ context.Context, _ protocol.AccountCtx, _ protocol.Nav, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	text, _ := params["text"].(string)
	var urls []string
	raw, ok := params["images"].([]any)
	if !ok {
		return protocol.Result{}, fmt.Errorf("reply_media: images must be an array")
	}
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			urls = append(urls, s)
		}
	}
	return protocol.Result{Text: text, MediaURLs: urls}, nil
}
```

`open_case.go`：

```go
func (OpenCase) DisplayName() string { return "打开工作流" }

// AdminParamsSchema exposes only the admin-configurable list entry params.
func (OpenCase) AdminParamsSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"case_ids": {
				"type": "array",
				"items": { "type": "string" },
				"x-admin": { "widget": "workflow_picker" }
			}
		}
	}`)
}
```

> `x-admin.widget` 是管理端表单控件提示（非标准 JSON Schema 关键字，校验器忽略）：`workflow_picker` → 工作流多选；`text` → 多行文本。`ReplyText` / `ReplyMedia` 的 schema 里 `text` 属性同样加 `"x-admin": {"widget": "text"}`。

注册（`apps/admin-api/cmd/admin-api/main.go:140` 与 `apps/pixoma/internal/app/telegram.go:165` 的 `Register(capability.OpenCase{...})` 旁追加）：

```go
_ = adminCaps.Register(capability.ReplyText{})
_ = adminCaps.Register(capability.ReplyMedia{})
```

（pixoma 侧 `r.Register(...)` 同理。）

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/capability/ ./internal/channel/tg/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/capability apps/admin-api/cmd/admin-api/main.go apps/pixoma/internal/app/telegram.go
git commit -m "feat(capability): add display capabilities and admin params schema"
```

---

### Task B3: ListCapabilities 返回 params_schema

**Files:**
- Modify: `internal/httpapi/channelmenu/handler.go`（`ListCapabilities`）
- Modify: `internal/httpapi/channelmenu/handler_test.go` 或新增 `capabilities_test.go`

**Interfaces:**
- Consumes: `capability.AdminSchema`
- Produces: DTO `{id, display_name, params_schema}`；`params_schema` = `AdminSchema(c)` 的 raw JSON

- [ ] **Step 1: 写失败测试**

`handler_test.go` 追加（沿用现有 handler 构造；`h.Capabilities` 需要真实 registry，注册 `ReplyText` 与 `OpenCase`）：

```go
func TestListCapabilitiesIncludesParamsSchema(t *testing.T) {
	reg := capability.NewRegistry()
	if err := reg.Register(capability.ReplyText{}); err != nil {
		t.Fatal(err)
	}
	h := &channelmenu.Handler{Capabilities: reg}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/capabilities", nil)
	w := httptest.NewRecorder()
	h.ListCapabilities(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("caps=%+v", out)
	}
	schema, ok := out[0]["params_schema"].(map[string]any)
	if !ok {
		t.Fatalf("params_schema missing: %+v", out[0])
	}
	if out[0]["display_name"] != "提示文字" {
		t.Fatalf("display_name=%v", out[0]["display_name"])
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["text"]; !ok {
		t.Fatalf("schema=%v", schema)
	}
}
```

（import 需补 `github.com/mr9esx/comfyui_tgbot/internal/channel/capability`。）

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/httpapi/channelmenu/`
Expected: FAIL（`params_schema` 缺失）。

- [ ] **Step 3: 实现**

`ListCapabilities` 的 DTO 与填充改为：

```go
type capDTO struct {
	ID           string          `json:"id"`
	DisplayName  string          `json:"display_name"`
	ParamsSchema json.RawMessage `json:"params_schema"`
}

for _, c := range caps {
	out = append(out, capDTO{ID: c.ID(), DisplayName: c.DisplayName(), ParamsSchema: capability.AdminSchema(c)})
}
```

`handler.go` 已 import `capability` 包（`Capabilities *capability.Registry`），确认 `encoding/json` 已导入。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/httpapi/channelmenu/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/httpapi/channelmenu/handler.go internal/httpapi/channelmenu/handler_test.go
git commit -m "feat(channelmenu): expose capability admin params schema"
```

---

### Task B4: 菜单校验「非分组叶子必须有能力」+ 默认种子改造

**Files:**
- Modify: `internal/menu/domain/validate.go`
- Modify: `internal/menu/domain/seed.go`（默认种子）
- Modify: `internal/menu/domain/validate_test.go`、`internal/menu/domain/seed_test.go`

**Interfaces:**
- Consumes: 无
- Produces: `Validate` 新增规则：叶子节点（无 children）且 `CapabilityID == ""` → `ErrValidation`（提示 `leaf item %q must have a capability`）；`DefaultSeedTree` 的占位按钮改为 `reply_text` 能力 + `params.text = "{{label}}：暂未开放"`（或等价中文）

- [ ] **Step 1: 写失败测试**

`validate_test.go` 追加：

```go
func TestValidate_LeafWithoutCapabilityRejected(t *testing.T) {
	tree := domain.MenuTree{Items: []domain.MenuNode{
		{ID: "a", Label: "A", Order: 0, Enabled: true},
	}}
	err := domain.Validate(context.Background(), tree, nil, nil, nil)
	if err == nil {
		t.Fatal("expected leaf-without-capability error")
	}
}

func TestValidate_LeafWithCapabilityAccepted(t *testing.T) {
	tree := domain.MenuTree{Items: []domain.MenuNode{
		{ID: "a", Label: "A", Order: 0, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "x"}},
	}}
	err := domain.Validate(context.Background(), tree, nil, func(_ context.Context, id string) (bool, error) { return id == "reply_text", nil }, func(_ context.Context, _ string, _ map[string]any) error { return nil })
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
```

`seed_test.go` 追加断言：默认种子每个叶子都有 `CapabilityID`。

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/menu/...`
Expected: 新测试 FAIL。

- [ ] **Step 3: 实现**

`validate.go` 的 `validateItem` 中（或 `Validate` 循环内）追加：

```go
if len(it.Children) == 0 && it.CapabilityID == "" {
	return fmt.Errorf("%w: leaf item %q must have a capability", ErrValidation, it.ID)
}
```

> 注意放对位置：`MenuNode` 展开为扁平 `MenuItem`（`Flatten`），children 信息在扁平结构里不可见——改为在 `validateNodeChildren(nodes)` 里做：遍历时叶子节点（`len(n.Children)==0`）且 `n.CapabilityID==""` 返回错误。若 `validateNodeChildren` 只能看到树结构，就在这里加。

`seed.go` 的默认种子：`btn-video` / `btn-recharge` / `btn-checkin` 改为：

```go
{ID: "btn-video", Label: "🎬 视频", Order: 1, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "🎬 视频：暂未开放"}},
{ID: "btn-recharge", Label: "💰 充值积分", Order: 2, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "💰 充值积分：暂未开放"}},
{ID: "btn-checkin", Label: "📅 签到", Order: 3, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "📅 签到：暂未开放"}},
```

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/menu/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/menu/domain/validate.go internal/menu/domain/seed.go internal/menu/domain/validate_test.go internal/menu/domain/seed_test.go
git commit -m "feat(menu): require capability on leaf items and reseed placeholders"
```

---

### Task B5: 旧数据兼容转换（Service.Get 读路径）

**Files:**
- Modify: `internal/menu/application/service.go`
- Modify: `internal/menu/application/service_test.go`

**Interfaces:**
- Consumes: `domain.MenuNode`
- Produces: `func convertLegacyNodes(nodes []domain.MenuNode) []domain.MenuNode`（同包函数）：`placeholder_text != ""` → `CapabilityID="reply_text"`, `Params={"text": placeholder_text}`（保留原字段不删，便于回滚）；`Reply != nil` → `CapabilityID="reply_media"`, `Params={"text": reply.Text, "images": reply.Images}`；递归 children；`Service.Get` 返回前对 `tree.Items` 应用转换

- [ ] **Step 1: 写失败测试**

`service_test.go` 追加（沿用现有 fake Store 构造）：

```go
func TestGetConvertsLegacyPlaceholderAndReply(t *testing.T) {
	store := &fakeStore{tree: domain.MenuTree{Items: []domain.MenuNode{
		{ID: "p", Label: "占位", Order: 0, Enabled: true, PlaceholderText: "即将上线"},
		{ID: "r", Label: "联系", Order: 1, Enabled: true, Reply: &domain.ReplyPayload{Text: "hi", Images: []string{"https://a/x.png"}}},
	}}}
	svc := &application.Service{Store: store}
	got, err := svc.Get(context.Background(), "ch")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]domain.MenuNode{}
	for _, it := range got.Items {
		byID[it.ID] = it
	}
	p := byID["p"]
	if p.CapabilityID != "reply_text" || p.Params["text"] != "即将上线" {
		t.Fatalf("placeholder item=%+v", p)
	}
	r := byID["r"]
	if r.CapabilityID != "reply_media" || r.Params["images"] == nil {
		t.Fatalf("reply item=%+v", r)
	}
}
```

（`fakeStore` 以现有 service_test 为准，若无则新建最小 fake 实现 `Store` 接口。）

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/menu/application/`
Expected: FAIL（转换未实现）。

- [ ] **Step 3: 实现**

`service.go` 追加：

```go
func convertLegacyNodes(nodes []domain.MenuNode) []domain.MenuNode {
	out := make([]domain.MenuNode, 0, len(nodes))
	for _, n := range nodes {
		if n.CapabilityID == "" {
			if text := strings.TrimSpace(n.PlaceholderText); text != "" {
				n.CapabilityID = "reply_text"
				n.Params = map[string]any{"text": text}
			} else if n.Reply != nil {
				n.CapabilityID = "reply_media"
				images := make([]any, 0, len(n.Reply.Images))
				for _, u := range n.Reply.Images {
					images = append(images, u)
				}
				params := map[string]any{"text": n.Reply.Text}
				if len(images) > 0 {
					params["images"] = images
				}
				n.Params = params
			}
		}
		if len(n.Children) > 0 {
			n.Children = convertLegacyNodes(n.Children)
		}
		out = append(out, n)
	}
	return out
}
```

`Service.Get` 返回前：

```go
tree.Items = convertLegacyNodes(tree.Items)
return tree, nil
```

import 补 `"strings"`。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/menu/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/menu/application/service.go internal/menu/application/service_test.go
git commit -m "feat(menu): convert legacy placeholder and reply items to capabilities"
```

---

### Task B6: TG 运行时统一走能力调用

**Files:**
- Modify: `internal/channel/tg/adapter.go`（`menuItemDispatch`、删除 `sendReplyMedia` 或改为仅内部不再使用）
- Modify: `internal/channel/tg/menu_runtime_test.go`

**Interfaces:**
- Consumes: 全部前序任务
- Produces: `menuItemDispatch`：`CapabilityID != ""` → 能力调用；`len(Children)>0` → `showGroup`；两者皆无 → `SendText("菜单配置无效")`（正常数据不会走到）

- [ ] **Step 1: 写失败测试**

`menu_runtime_test.go` 追加：

```go
type textCaptureOutbound struct {
	texts []string
}

func (c *textCaptureOutbound) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	c.texts = append(c.texts, text)
	return nil
}
func (c *textCaptureOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []ports.MenuEntry) error { return nil }
func (c *textCaptureOutbound) SendList(context.Context, sharedkernel.ChannelAddr, string, [][]ports.Button) error { return nil }
func (c *textCaptureOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string) error { return nil }
func (c *textCaptureOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string) error { return nil }

func TestMenuItemDispatchInvokesReplyTextCapability(t *testing.T) {
	out := &textCaptureOutbound{}
	ad := New(out)
	ad.ChannelID = "tg-default"
	reg := capability.NewRegistry()
	if err := reg.Register(capability.ReplyText{}); err != nil {
		t.Fatal(err)
	}
	ad.Registry = reg
	item := domain.MenuNode{
		ID: "p", Label: "占位", Order: 0, Enabled: true,
		CapabilityID: "reply_text",
		Params:       map[string]any{"text": "即将上线"},
	}
	if err := ad.menuItemDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}, item); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "即将上线" {
		t.Fatalf("texts=%q", out.texts)
	}
}
```

> import 需补 `github.com/mr9esx/comfyui_tgbot/internal/channel/capability`。`dispatchInvoke` 经 `Registry.Invoke` → `renderResult` 发送文本。

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/`
Expected: FAIL（当前 `menuItemDispatch` 对 `Reply != nil` 走 `sendReplyMedia`，对纯 placeholder 走 `PlaceholderText`）。

- [ ] **Step 3: 实现**

`menuItemDispatch` 替换为：

```go
func (a *Adapter) menuItemDispatch(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, item domain.MenuNode) error {
	if item.CapabilityID != "" {
		inv, err := a.baseInvoke(ctx, chatID)
		if err != nil {
			return err
		}
		inv.CapabilityID = item.CapabilityID
		inv.Params = item.Params
		inv.Nav = protocol.Nav{Back: "root"}
		return a.dispatchInvoke(ctx, chatID, inv)
	}
	if len(item.Children) > 0 {
		return a.showGroup(ctx, addr, item.ID)
	}
	return a.Out.SendText(ctx, addr, "菜单配置无效")
}
```

删除 `sendReplyMedia`（或保留但无引用时删除以过 lint）；`dispatchInvoke` 走 `Registry.Invoke` → `renderResult`（已支持 `MediaURLs`）。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/tg/adapter.go internal/channel/tg/menu_runtime_test.go
git commit -m "feat(tg): dispatch menu items through capability registry only"
```

---

### Task B7: 全量验证

**Files:** 无新增。

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma
go build ./...
go vet ./internal/channel/... ./internal/menu/... ./internal/httpapi/channelmenu/
go test ./...
```
Expected: 全部通过（除用户 WIP 区域既有的 edge-metrics 两个失败外）。

- [ ] **Step 2: 提交遗留改动**

```bash
git add -A
git commit -m "chore: finalize tg menu editor backend"
```
