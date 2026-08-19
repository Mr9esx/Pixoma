# TG 菜单/卡片重构 · 后端核心（领域模型 + 存储 + 管理 API）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立全新的 TG 菜单/卡片领域模型（Menu / MenuItem / Card / CardButton / Action）、持久化与新管理 API，完全替代旧 MenuNode 树模型（不兼容、不迁移）。

**Architecture:** 新类型放 `internal/menu/domain`（`menu.go` / `card.go`，旧类型暂留待运行时计划删除）；存储新增两张表（`channel_main_menus`、`channel_cards`）；管理 API 挂到现有 `/api/v1/channels/{id}/menu` 与新增 `/api/v1/channels/{id}/cards`。

**Tech Stack:** Go（module `github.com/mr9esx/comfyui_tgbot`）、chi、GORM（SQLite）、标准库测试。

## Global Constraints

- 测试命令：仓库根 `go test ./...`；构建 `go build ./...`；格式 `gofmt -w`。
- 完全重构：不兼容旧 `MenuNode` / `MenuTree` / `capability_id` 菜单字段；旧类型在运行时计划中删除，本计划只新增不动旧代码（除路由挂载）。
- 校验只在保存时；`open_card` 必须指向存在的卡片；`open_workflow` 至少一个 `workflow_ids`；`open_url` 必须 http/https；媒体 URL 非空。
- 卡片按渠道隔离（`channel_id` 归属），可被本渠道多个入口引用；删除被引用卡片必须拒绝。

---

### Task B1: 领域模型（Menu / MenuItem / Card / CardButton / Action / Media）

**Files:**
- Create: `internal/menu/domain/menu.go`
- Create: `internal/menu/domain/card.go`
- Create: `internal/menu/domain/menu_test.go`、`internal/menu/domain/card_test.go`

**Interfaces:**
- Produces:
  - `type Menu struct { ID string; Name string; Columns int; Items []MenuItem }`
  - `type MenuItem struct { ID string; Label string; Action Action }`
  - `type Card struct { ID string; Name string; Media []Media; Text string; Buttons []CardButton }`
  - `type CardButton struct { ID string; Label string; Action Action }`
  - `type Media struct { Kind string; URL string; Caption string }`（kind: image|video|animation）
  - `type Action struct { Type string; CardID string; WorkflowIDs []string; Mode string; DirectID string; Text string; Media []Media; URL string }`（type: open_card|open_workflow|send_text|send_media|open_url|copy_text|placeholder；`Mode`=list|direct）
  - `func ValidateMenu(m Menu) error`、`func ValidateCard(c Card) error`、`func ValidateAction(a Action) error`

- [ ] **Step 1: 写失败测试**

`menu_test.go`：

```go
package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

func TestMenuJSONRoundTrip(t *testing.T) {
	m := domain.Menu{
		ID: "menu-1", Name: "主菜单", Columns: 2,
		Items: []domain.MenuItem{
			{ID: "mi-1", Label: "图片生成", Action: domain.Action{Type: "open_card", CardID: "card-1"}},
			{ID: "mi-2", Label: "充值", Action: domain.Action{Type: "send_text", Text: "即将上线"}},
		},
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Menu
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Items[0].Action.CardID != "card-1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidateActionRules(t *testing.T) {
	if err := domain.ValidateAction(domain.Action{Type: "open_card"}); err == nil {
		t.Fatal("open_card without card_id must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "open_workflow", WorkflowIDs: nil}); err == nil {
		t.Fatal("open_workflow without workflows must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "open_url", URL: "ftp://x"}); err == nil {
		t.Fatal("non-http url must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "placeholder"}); err != nil {
		t.Fatalf("placeholder must pass: %v", err)
	}
}
```

`card_test.go`：

```go
func TestCardJSONRoundTrip(t *testing.T) {
	c := domain.Card{
		ID: "card-1", Name: "开始生成", Text: "选一种风格：",
		Media: []domain.Media{{Kind: "image", URL: "https://a/img.png"}},
		Buttons: []domain.CardButton{
			{ID: "cb-1", Label: "写实风格", Action: domain.Action{Type: "open_workflow", WorkflowIDs: []string{"w1"}}},
		},
	}
	raw, _ := json.Marshal(c)
	var got domain.Card
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Media[0].URL != "https://a/img.png" || got.Buttons[0].Action.WorkflowIDs[0] != "w1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidateCardRequiresContent(t *testing.T) {
	if err := domain.ValidateCard(domain.Card{ID: "c", Name: "x"}); err == nil {
		t.Fatal("empty card must fail")
	}
	ok := domain.Card{ID: "c", Name: "x", Text: "hi", Buttons: []domain.CardButton{
		{ID: "b", Label: "B", Action: domain.Action{Type: "placeholder"}},
	}}
	if err := domain.ValidateCard(ok); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	noButtons := domain.Card{ID: "c2", Name: "终局", Media: []domain.Media{{Kind: "image", URL: "https://a/x.png"}}}
	if err := domain.ValidateCard(noButtons); err != nil {
		t.Fatalf("card without buttons must pass (auto back still applies): %v", err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/menu/domain/`
Expected: 编译失败（类型不存在）。

- [ ] **Step 3: 实现 menu.go / card.go**

```go
package domain

// Menu is the main keyboard definition for a channel.
type Menu struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Columns int        `json:"columns"`
	Items   []MenuItem `json:"items"`
}

type MenuItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Action Action `json:"action"`
}

type Action struct {
	Type        string  `json:"type"` // open_card | open_workflow | send_text | send_media | open_url | copy_text | placeholder
	CardID      string  `json:"card_id,omitempty"`
	WorkflowIDs []string `json:"workflow_ids,omitempty"`
	Mode        string  `json:"mode,omitempty"` // list | direct
	DirectID    string  `json:"direct_id,omitempty"`
	Text        string  `json:"text,omitempty"`
	Media       []Media `json:"media,omitempty"`
	URL         string  `json:"url,omitempty"`
}

func ValidateMenu(m Menu) error {
	if m.ID == "" {
		return ErrValidation // 复用现有 sentinel
	}
	for _, it := range m.Items {
		if it.Label == "" {
			return ErrValidation
		}
		if err := ValidateAction(it.Action); err != nil {
			return err
		}
	}
	return nil
}
```

`card.go` 定义 `Card` / `CardButton` / `Media` 与 `ValidateCard`；`ValidateAction` 规则：

```go
func ValidateAction(a Action) error {
	switch a.Type {
	case "open_card":
		if a.CardID == "" {
			return ErrValidation
		}
	case "open_workflow":
		if len(a.WorkflowIDs) == 0 {
			return ErrValidation
		}
	case "open_url":
		if !strings.HasPrefix(a.URL, "http://") && !strings.HasPrefix(a.URL, "https://") {
			return ErrValidation
		}
	case "send_media":
		if len(a.Media) == 0 {
			return ErrValidation
		}
		for _, m := range a.Media {
			if m.URL == "" {
				return ErrValidation
			}
		}
	case "send_text", "copy_text", "placeholder":
		// no extra constraint
	default:
		return ErrValidation
	}
	return nil
}
```

`ValidateCard`：`Text == "" && len(Media) == 0` 时报错（卡片必须有内容）；按钮列表可选（运行时自动生成「返回」）。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/menu/domain/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/menu/domain/menu.go internal/menu/domain/card.go internal/menu/domain/menu_test.go internal/menu/domain/card_test.go
git commit -m "feat(menu): add card/menu domain model"
```

---

### Task B2: 存储（channel_main_menus / channel_cards）

**Files:**
- Create: `internal/menu/infrastructure/persistence/card_repository.go`
- Create: `internal/menu/infrastructure/persistence/card_repository_test.go`

**Interfaces:**
- Consumes: `domain.Menu` / `domain.Card`
- Produces:
  - `type CardRepository interface { GetMenu(ctx, channelID) (domain.Menu, error); PutMenu(ctx, channelID, domain.Menu) error; ListCards(ctx, channelID) ([]domain.Card, error); GetCard(ctx, channelID, id) (domain.Card, error); CreateCard(ctx, channelID, domain.Card) error; UpdateCard(ctx, channelID, domain.Card) error; DeleteCard(ctx, channelID, id) error; CardReferences(ctx, channelID, id) ([]string, error) }`
  - `type GormCardRepository struct{ db *gorm.DB }`，表 `channel_main_menus`（channel_id PK、doc_json）、`channel_cards`（id + channel_id、doc_json）

- [ ] **Step 1: 写失败测试**

`card_repository_test.go`（沿用现有 `db.Open` SQLite 内存模式）：

```go
func TestCardRepositoryMenuAndCardsRoundTrip(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:cards_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &persistence.CardRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()
	if err := repo.PutMenu(ctx, "ch1", domain.Menu{ID: "m", Name: "主", Columns: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetMenu(ctx, "ch1")
	if err != nil || got.Name != "主" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	card := domain.Card{ID: "c1", Name: "开始生成", Text: "hi", Buttons: []domain.CardButton{{ID: "b", Label: "B", Action: domain.Action{Type: "placeholder"}}}}
	if err := repo.CreateCard(ctx, "ch1", card); err != nil {
		t.Fatal(err)
	}
	refs, err := repo.CardReferences(ctx, "ch1", "c1")
	if err != nil || len(refs) != 0 {
		t.Fatalf("refs=%v err=%v", refs, err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/menu/infrastructure/persistence/`
Expected: 编译失败。

- [ ] **Step 3: 实现**

行类型（同文件）：

```go
type MainMenuRow struct {
	ChannelID string `gorm:"primaryKey;size:64"`
	DocJSON   string `gorm:"type:text;not null"`
}
func (MainMenuRow) TableName() string { return "channel_main_menus" }

type CardRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	ChannelID string `gorm:"index;size:64;not null"`
	DocJSON   string `gorm:"type:text;not null"`
}
func (CardRow) TableName() string { return "channel_cards" }
```

`CardReferences`：遍历本渠道所有卡片 doc，统计 `open_card.CardID == id` 的引用来源（返回引用来源描述，如 `menu:mi-1` / `card:c2:cb-1`）。

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/menu/infrastructure/persistence/
git add internal/menu/infrastructure/persistence/card_repository.go internal/menu/infrastructure/persistence/card_repository_test.go
git commit -m "feat(menu): card and main menu persistence"
```

---

### Task B3: 管理 API（menu / cards CRUD + references）

**Files:**
- Modify: `internal/httpapi/adminhost/server.go`（挂载新路由）
- Create: `internal/httpapi/menucards/handler.go`、`handler_test.go`
- Modify: `apps/admin-api/cmd/admin-api/main.go`、`apps/pixoma/cmd/pixoma/main.go`（装配 handler）

**Interfaces:**
- Consumes: `CardRepository`、`domain.ValidateMenu/ValidateCard`
- Produces:
  - `GET/PUT /api/v1/channels/{id}/menu` → `domain.Menu`
  - `GET /api/v1/channels/{id}/cards?q=` → `[]domain.Card`
  - `POST /api/v1/channels/{id}/cards` → `domain.Card`（校验 + 创建）
  - `PATCH /api/v1/channels/{id}/cards/{cardID}` → `domain.Card`
  - `DELETE /api/v1/channels/{id}/cards/{cardID}` → 被引用时 409 + 引用列表
  - `GET /api/v1/channels/{id}/cards/{cardID}/references` → `[]string`

- [ ] **Step 1: 写失败测试**

`handler_test.go` 用内存 fake 仓库覆盖：PUT menu → GET menu 往返；POST card → PATCH → DELETE（无引用成功）；DELETE 被引用卡片返回 409。核心断言如：

```go
func TestMenuCardsHandlerCRUD(t *testing.T) {
	repo := newMemCardRepo()
	h := menucardsapi.NewHandler(repo)
	r := chi.NewRouter()
	r.Route("/api/v1/channels/{id}", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	// PUT menu
	body, _ := json.Marshal(domain.Menu{ID: "m", Name: "主", Columns: 2})
	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", body))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("put menu status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()

	// POST card
	cardBody, _ := json.Marshal(domain.Card{ID: "c1", Name: "x", Text: "hi", Buttons: []domain.CardButton{{ID: "b", Label: "B", Action: domain.Action{Type: "placeholder"}}}})
	cardResp, err := http.DefaultClient.Do(mustReq(t, http.MethodPost, srv.URL+"/api/v1/channels/ch1/cards", cardBody))
	if err != nil || cardResp.StatusCode != http.StatusCreated {
		t.Fatalf("post card status=%v err=%v", cardResp.StatusCode, err)
	}
	cardResp.Body.Close()

	// DELETE referenced card → 409
	menuWithRef, _ := json.Marshal(domain.Menu{ID: "m", Name: "主", Columns: 2, Items: []domain.MenuItem{{ID: "mi", Label: "L", Action: domain.Action{Type: "open_card", CardID: "c1"}}}})
	_, _ = http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", menuWithRef))
	delResp, err := http.DefaultClient.Do(mustReq(t, http.MethodDelete, srv.URL+"/api/v1/channels/ch1/cards/c1", nil))
	if err != nil || delResp.StatusCode != http.StatusConflict {
		t.Fatalf("delete referenced status=%v err=%v", delResp.StatusCode, err)
	}
	delResp.Body.Close()
}
```

（`mustReq` 为测试辅助；`newMemCardRepo` 为内存 fake 实现接口。）

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/httpapi/menucards/`
Expected: 编译失败。

- [ ] **Step 3: 实现 handler.go**

`NewHandler(repo)` 返回 `*Handler`；`Mount(r chi.Router)` 挂 `/menu` 与 `/cards` 子路由。每个写操作先 `ValidateMenu/ValidateCard`；DELETE 先查 `CardReferences`，非空 → `http.StatusConflict` + `{"error":...,"references":[...]}`；GET cards 支持 `?q=` 按 Name 过滤。

- [ ] **Step 4: 挂载 + 运行确认通过**

`adminhost/server.go` 的 channels 路由内追加：

```go
if opts.MenuCards != nil {
	opts.MenuCards.Mount(r) // 挂到 /api/v1/channels/{id} 下
}
```

两个 main 里 `NewGormCardRepository(gdb)` 装配并 AutoMigrate 新表。

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/httpapi/menucards/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/httpapi/adminhost/server.go internal/httpapi/menucards apps/admin-api/cmd/admin-api/main.go apps/pixoma/cmd/pixoma/main.go
git commit -m "feat(adminapi): menu and cards management endpoints"
```

---

### Task B4: 全量验证

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma
go build ./...
go vet ./internal/menu/... ./internal/httpapi/menucards/
go test ./...
```
Expected: 通过（除用户 WIP 区域既有的 edge-metrics 两个失败）。

- [ ] **Step 2: 提交遗留改动**

```bash
git add -A
git commit -m "chore: finalize menu/card backend core"
```
