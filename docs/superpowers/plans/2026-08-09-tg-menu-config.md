---
change: tg-menu-config
design-doc: docs/superpowers/specs/2026-08-09-tg-menu-config-design.md
base-ref: 6e0d1afba44e9ffabcfc621da960b691a0b21ab7
---

# TG Menu 可配置化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Telegram 主菜单从硬编码改为可落库配置：Bot 按配置渲染键盘并执行动作（含 `reply_media`），admin-api + 控制台可整份读写。

**Architecture:** 新建 `internal/tgmenu`（domain / application / GORM 持久化）作为 Menu 真相源；`channel/tg` 只依赖只读端口；`internal/httpapi/tgmenu` 挂到 admin-api；`web/admin` 增加 `/tg-menu` 页与侧栏。首次读空表时 upsert 默认种子；校验失败不写库。

**Tech Stack:** Go、GORM/SQLite、chi、既有 catalog Case 仓储、go-telegram/bot、React + TanStack Query/Router、react-i18next、sonner

## Global Constraints

- 产物语言：zh-CN
- **单 Bot、无 `bot_id` / 多租户**；文档 id 固定 `"default"`
- **禁止** `apps/admin-api` / `internal/httpapi/tgmenu` 依赖 `channel/tg`
- **禁止** 管理端图片本地上传 / 对象存储；图片仅 http(s) URL
- **禁止** 前端 mock 作为验收路径；Network 只打 `VITE_ADMIN_API_BASE`
- 无鉴权，与现 admin-api 一致；README 须警示仅内网
- PUT 为整份替换；400 校验 / 500 存储；404 不用于整份文档
- 空库或读失败：运行时回退内存种子并对齐现网 `MainMenuRows`，打 error/warn 日志
- 刷新：每次构建键盘读仓储（可选进程内 TTL≤5s）；不要求热推送 / 重启
- Comfy Mock 开关与本 change 无关；不破坏既有 Case/实例 API
- 触及数据模型时同步 `docs/architecture/data-model.md`（及必要时 runtime / bounded-contexts）
- 遵循仓库 golang skills；改动范围限于本 change

---

## 文件结构（先锁定职责）

| 路径 | 职责 |
|---|---|
| `internal/tgmenu/domain/document.go` | `MenuDocument` / `MenuItem` / `MenuAction` / `ReplyPayload`；常量 `DocumentIDDefault` |
| `internal/tgmenu/domain/validate.go` | `Validate(doc, caseExists)`；label 唯一、动作字段、`reply_media` URL |
| `internal/tgmenu/domain/seed.go` | `DefaultSeed()` 对齐现网主菜单 |
| `internal/tgmenu/domain/repository.go` | `Repository`：`Get` / `Replace` |
| `internal/tgmenu/domain/errors.go` | `ErrValidation` 等 |
| `internal/tgmenu/application/service.go` | `Get`（空则种子 upsert）、`Replace`（校验后写） |
| `internal/tgmenu/infrastructure/persistence/gorm_repository.go` | 表 `tg_menu_configs`；`MenuRow` |
| `internal/httpapi/tgmenu/handler.go` | `GET/PUT /api/v1/tg-menu` |
| `internal/channel/tg/menu_source.go` | 窄只读端口 + 键盘构建 / label→item 查找 |
| `internal/channel/tg/{adapter,bot,messenger,menu}.go` | 读配置渲染键盘；动作分发；`SendPhotoURL` |
| `apps/bot/cmd/comfyui-bot/main.go` | AutoMigrate MenuRow；注入 Menu 到 Adapter / Messenger |
| `apps/admin-api/cmd/admin-api/main.go` + `internal/server/server.go` | 迁移 + 挂载 handler |
| `web/admin/src/config/menu.ts` + i18n | 侧栏「TG 菜单」 |
| `web/admin/src/lib/api/tg-menu.ts` + features/routes | Menu 管理页 |
| `docs/architecture/data-model.md`（+ 必要时 runtime / bounded-contexts） | 新表与主菜单来源 |
| `apps/admin-api/README.md`、`web/admin/README.md` | curl / 联调补充 |

---

### Task 1: tgmenu domain — 模型、种子、校验

**Files:**
- Create: `internal/tgmenu/domain/document.go`
- Create: `internal/tgmenu/domain/errors.go`
- Create: `internal/tgmenu/domain/seed.go`
- Create: `internal/tgmenu/domain/validate.go`
- Create: `internal/tgmenu/domain/validate_test.go`
- Create: `internal/tgmenu/domain/repository.go`

**Interfaces:**
- Produces:
  - `const DocumentIDDefault = "default"`
  - `type MenuAction string`：`open_case` | `list_cases_by_tag` | `placeholder` | `reply_media`
  - `type ReplyPayload struct { Text string; Images []string }`
  - `type MenuItem struct { ID, Label string; Row, Col int; Enabled bool; Action MenuAction; CaseID, Tag, PlaceholderText string; Reply *ReplyPayload }`
  - `type MenuDocument struct { ID string; Items []MenuItem; UpdatedAt time.Time }`
  - `func DefaultSeed() MenuDocument`
  - `type CaseExistsFunc func(ctx context.Context, caseID string) (bool, error)`
  - `func Validate(ctx context.Context, doc MenuDocument, caseExists CaseExistsFunc) error`
  - `type Repository interface { Get(ctx, id string) (MenuDocument, error); Replace(ctx, doc MenuDocument) error }`
  - `var ErrNotFound, ErrValidation error`

- [x] **Step 1: 写失败测试**

```go
package domain_test

func TestDefaultSeed_MatchesLegacyLayout(t *testing.T) {
	doc := domain.DefaultSeed()
	if doc.ID != domain.DocumentIDDefault {
		t.Fatalf("id=%q", doc.ID)
	}
	if len(doc.Items) != 6 {
		t.Fatalf("want 6 items, got %d", len(doc.Items))
	}
	byID := map[string]domain.MenuItem{}
	for _, it := range doc.Items {
		byID[it.ID] = it
	}
	img := byID["btn-image"]
	if img.Label != "🖼 图片" || img.Action != domain.ActionListCasesByTag || img.Tag != "image" {
		t.Fatalf("btn-image: %+v", img)
	}
	for _, id := range []string{"btn-video", "btn-recharge", "btn-checkin", "btn-profile", "btn-help"} {
		if byID[id].Action != domain.ActionPlaceholder {
			t.Fatalf("%s action=%s", id, byID[id].Action)
		}
	}
}

func TestValidate_RejectsDuplicateLabelAndBadReplyMedia(t *testing.T) {
	ctx := context.Background()
	exists := func(context.Context, string) (bool, error) { return true, nil }

	dup := domain.DefaultSeed()
	dup.Items[1].Label = dup.Items[0].Label
	if err := domain.Validate(ctx, dup, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("dup label: %v", err)
	}

	emptyReply := domain.DefaultSeed()
	emptyReply.Items[0].Action = domain.ActionReplyMedia
	emptyReply.Items[0].Tag = ""
	emptyReply.Items[0].Reply = &domain.ReplyPayload{}
	if err := domain.Validate(ctx, emptyReply, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty reply: %v", err)
	}

	badURL := domain.DefaultSeed()
	badURL.Items[0].Action = domain.ActionReplyMedia
	badURL.Items[0].Tag = ""
	badURL.Items[0].Reply = &domain.ReplyPayload{Images: []string{"ftp://x/a.png"}}
	if err := domain.Validate(ctx, badURL, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad url: %v", err)
	}
}

func TestValidate_OpenCaseRequiresExistingCase(t *testing.T) {
	ctx := context.Background()
	doc := domain.DefaultSeed()
	doc.Items[0].Action = domain.ActionOpenCase
	doc.Items[0].CaseID = "missing"
	doc.Items[0].Tag = ""
	exists := func(_ context.Context, id string) (bool, error) { return id == "ok", nil }
	if err := domain.Validate(ctx, doc, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing case: %v", err)
	}
	doc.Items[0].CaseID = "ok"
	if err := domain.Validate(ctx, doc, exists); err != nil {
		t.Fatal(err)
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tgmenu/domain/ -count=1`

Expected: FAIL（包/符号不存在）

- [x] **Step 3: 最小实现**

`document.go`：动作常量与结构体；JSON tag 用 `case_id`、`placeholder_text`、`reply`、`images`。

`seed.go`：行/列与现网 `MainMenuRows` 一致：

| id | label | row,col | action |
|---|---|---|---|
| btn-image | 🖼 图片 | 0,0 | list_cases_by_tag tag=image |
| btn-video | 🎬 视频 | 0,1 | placeholder |
| btn-recharge | 💰 充值积分 | 1,0 | placeholder |
| btn-checkin | 📅 签到 | 1,1 | placeholder |
| btn-profile | 👤 个人中心 | 2,0 | placeholder |
| btn-help | 🆘 帮助 | 2,1 | placeholder |

`validate.go` 规则：

1. 每项 `id` 非空；`label` trim 后非空；**全部项** label 文档内唯一（trim 后比较）。
2. `open_case`：`case_id` 必填；`caseExists` 必须 true（nil `caseExists` 时视为无法校验 → ErrValidation）。
3. `list_cases_by_tag`：`tag` 必填。
4. `placeholder`：不要求 case/tag；`placeholder_text` 可选。
5. `reply_media`：`reply` 非 nil；`text` trim 非空 **或** `images` 长度>0；每个 image 经 `url.Parse` 且 scheme 为 `http`/`https`、Host 非空。
6. 未知 `action` → ErrValidation。

`repository.go`：仅接口，无实现。

`errors.go`：

```go
var (
	ErrNotFound    = errors.New("tg menu not found")
	ErrValidation  = errors.New("tg menu validation failed")
)
```

Validate 返回 `fmt.Errorf("%w: ...", ErrValidation)`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/tgmenu/domain/ -count=1`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/tgmenu/domain/
git commit -m "$(cat <<'EOF'
feat(tgmenu): add menu domain model, seed, and validation

EOF
)"
```

---

### Task 2: tgmenu persistence — GORM 表与种子 upsert

**Files:**
- Create: `internal/tgmenu/infrastructure/persistence/gorm_repository.go`
- Create: `internal/tgmenu/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:**
- Consumes: `domain.MenuDocument`、`domain.Repository`、`domain.DefaultSeed`、`domain.DocumentIDDefault`
- Produces:
  - `type MenuRow struct { ID string; ItemsJSON string; UpdatedAt time.Time }` → 表名 `tg_menu_configs`
  - `func NewGormRepository(db *gorm.DB) *GormRepository`
  - `Get`：无行 → `domain.ErrNotFound`
  - `Replace`：整份 Save（含 `UpdatedAt = time.Now().UTC()`）
  - `EnsureDefault(ctx) (MenuDocument, error)`：无行则写入 `DefaultSeed()` 后返回；有行则 Get

- [x] **Step 1: 写失败测试**

```go
func openRepo(t *testing.T) *persistence.GormRepository {
	t.Helper()
	dsn := "file:tgmenu_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MenuRow{}); err != nil {
		t.Fatal(err)
	}
	return persistence.NewGormRepository(gdb)
}

func TestEnsureDefaultAndReplaceRoundTrip(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	doc, err := repo.EnsureDefault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 6 {
		t.Fatalf("seed items=%d", len(doc.Items))
	}
	again, err := repo.EnsureDefault(ctx)
	if err != nil || again.Items[0].Label != doc.Items[0].Label {
		t.Fatalf("second ensure mutated: %v %+v", err, again)
	}
	doc.Items[0].Label = "🖼 图生图"
	doc.Items[0].Action = domain.ActionReplyMedia
	doc.Items[0].Tag = ""
	doc.Items[0].Reply = &domain.ReplyPayload{
		Text:   "hello",
		Images: []string{"https://example.com/a.png"},
	}
	if err := repo.Replace(ctx, doc); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, domain.DocumentIDDefault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Label != "🖼 图生图" || got.Items[0].Reply == nil || len(got.Items[0].Reply.Images) != 1 {
		t.Fatalf("got=%+v", got.Items[0])
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tgmenu/infrastructure/persistence/ -count=1`

Expected: FAIL

- [x] **Step 3: 最小实现**

```go
type MenuRow struct {
	ID        string    `gorm:"primaryKey;size:64"`
	ItemsJSON string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
func (MenuRow) TableName() string { return "tg_menu_configs" }
```

- `Get`：`First` by id；`json.Unmarshal` → `MenuDocument`（ID/UpdatedAt 来自行）。
- `Replace`：marshal `doc.Items`；`Save` 整行；忽略调用方 UpdatedAt，写入 `time.Now().UTC()`。
- `EnsureDefault`：`Get`；若 `ErrNotFound` 则 `Replace(DefaultSeed())` 再 `Get`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/tgmenu/infrastructure/persistence/ -count=1`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/tgmenu/infrastructure/persistence/
git commit -m "$(cat <<'EOF'
feat(tgmenu): persist menu config with default seed upsert

EOF
)"
```

---

### Task 3: tgmenu application — Get / Replace

**Files:**
- Create: `internal/tgmenu/application/service.go`
- Create: `internal/tgmenu/application/service_test.go`

**Interfaces:**
- Consumes: `domain.Repository`、`domain.Validate`、`persistence.EnsureDefault`（若 Ensure 在 repo 上）或 service 内调用 `Get`+种子
- Produces:
  - `type CaseChecker interface { CaseExists(ctx context.Context, caseID string) (bool, error) }`
  - `type Service struct { Repo domain.Repository; Cases CaseChecker }`（Repo 需支持 EnsureDefault——可把 EnsureDefault 放在具体类型，或扩展接口）
  - 推荐扩展仓储：在 persistence 暴露 `EnsureDefault`；Service 持有 `*persistence.GormRepository` **或** 窄接口：

```go
type Store interface {
	domain.Repository
	EnsureDefault(ctx context.Context) (domain.MenuDocument, error)
}

type Service struct {
	Store Store
	Cases CaseChecker
}

func (s *Service) Get(ctx context.Context) (domain.MenuDocument, error)
func (s *Service) Replace(ctx context.Context, items []domain.MenuItem) (domain.MenuDocument, error)
```

- `Get`：`EnsureDefault`（懒种子写入）。
- `Replace`：构造 `MenuDocument{ID: default, Items: items}` → `Validate`（用 Cases）→ 失败不写 → 成功 `Replace` → 再 `Get`。

CaseChecker 适配 catalog：

```go
type CatalogCaseChecker struct{ Repo catalogdomain.Repository }

func (c CatalogCaseChecker) CaseExists(ctx context.Context, caseID string) (bool, error) {
	_, err := c.Repo.Get(ctx, sharedkernel.CaseID(caseID))
	if errors.Is(err, catalogdomain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
```

（可放在 `application/case_checker.go`。）

- [x] **Step 1: 写失败测试**

用内存 fake Store + fake Cases：

```go
func TestService_ReplaceRejectsInvalidWithoutWriting(t *testing.T) {
	store := newMemStore(t) // EnsureDefault 写入种子
	svc := &application.Service{Store: store, Cases: alwaysMissing{}}
	_, err := svc.Replace(context.Background(), []domain.MenuItem{{
		ID: "x", Label: "X", Action: domain.ActionOpenCase, CaseID: "nope", Enabled: true,
	}})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err=%v", err)
	}
	doc, _ := store.Get(context.Background(), domain.DocumentIDDefault)
	if len(doc.Items) != 6 { // 仍是种子
		t.Fatalf("store mutated: %d", len(doc.Items))
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tgmenu/application/ -count=1`

Expected: FAIL

- [x] **Step 3: 最小实现** + Step 4 测试 PASS

- [x] **Step 5: Commit**

```bash
git add internal/tgmenu/application/
git commit -m "$(cat <<'EOF'
feat(tgmenu): add Get/Replace application service

EOF
)"
```

---

### Task 4: channel/tg — 键盘来自 Menu + 动作分发（含 reply_media）

**Files:**
- Create: `internal/channel/tg/menu_runtime.go`
- Modify: `internal/channel/tg/messenger.go` — 增加 `SendPhotoURL`
- Modify: `internal/channel/tg/bot.go` — `BotMessenger` 实现 `SendPhotoURL`；`SendMenu` 用配置键盘
- Modify: `internal/channel/tg/adapter.go` — 注入 Menu 源；`HandleText` 按 label 分发
- Modify: `internal/channel/tg/adapter_test.go`、必要时 `bot_test.go`
- Keep: `internal/channel/tg/menu.go` 常量作种子文案/测试夹具；**运行时不得只靠 `MainMenuRows()` 拼键盘**

**Interfaces:**
- Consumes: `tgmenu` 只读（避免 channel → application 厚依赖；推荐窄接口）：

```go
// MenuReader is the read port for main ReplyKeyboard config.
type MenuReader interface {
	GetMenu(ctx context.Context) (tgmenudomain.MenuDocument, error)
}
```

- Adapter 增加字段 `Menu MenuReader`（可 nil → 回退 `DefaultSeed()` + warn 日志）。
- Produces:
  - `func BuildReplyKeyboard(doc tgmenudomain.MenuDocument) *models.ReplyKeyboardMarkup` — 仅 `enabled`，按 `(row,col)` 排序分行
  - `func FindEnabledByLabel(doc, label) (MenuItem, bool)`
  - `Messenger.SendPhotoURL(ctx, chatID int64, imageURL, caption string) error`
  - `BotMessenger.SendMenu`：若持有 `Menu MenuReader`，则读配置构建键盘；否则回退种子
  - 或更干净：`SendMenu` 仍收 markup——让 Adapter `sendMainMenu` 先 `BuildReplyKeyboard` 再调用新方法 `SendMenuMarkup`。**推荐最小改动**：给 `BotMessenger` / `Adapter` 都注入同一 `MenuReader`，`SendMenu` 内部读 Menu。

`SendPhotoURL` 实现（go-telegram/bot）：

```go
func (m *BotMessenger) SendPhotoURL(ctx context.Context, chatID int64, imageURL, caption string) error {
	_, err := m.Bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  chatID,
		Caption: caption,
		Photo:   &models.InputFileString{Data: imageURL},
	})
	return err
}
```

（若当前 bot 模型字段名不同，以模块内 `InputFileString` / 等价 URL 类型为准。）

点击分发（替换 `HandleText` 里对 `BtnImage`/`BtnVideo`/… 的硬编码 switch 主体）：

1. `/start` `/menu` → `sendMainMenu`
2. `/cases` → 兼容：`list_cases_by_tag` tag=`image`（或查找种子项）
3. `/help` → 可保留现帮助文案 **或** 若 label 匹配到 menu 项则走配置
4. 否则 `doc := GetMenu`；`FindEnabledByLabel`：
   - `list_cases_by_tag` → 泛化 `showCasesByTag(ctx, chatID, tag)`（从 `showImageCases` 抽出 tag 参数）
   - `open_case` → `showCasePreview` 或 `startCase`（与 inline 预览一致：**先预览**）
   - `placeholder` → `SendMenu` 提示：`placeholder_text` 或默认「暂未开放」
   - `reply_media` → 有 text 则 `SendText`；再对每个 URL `SendPhotoURL`；单张失败 `slog.Error` 继续；若全部图片失败且无 text → `SendText` 错误提示
5. `isMenuCommand`：改为「slash 命令 **或** label 命中当前 enabled 项」（读 Menu；失败时回退常量列表以免会话吞掉菜单点击）

读失败：`slog.Error` + 使用 `domain.DefaultSeed()`。

- [x] **Step 1: 写失败测试**

```go
type staticMenu struct{ doc tgmenudomain.MenuDocument }

func (s staticMenu) GetMenu(context.Context) (tgmenudomain.MenuDocument, error) { return s.doc, nil }

func TestHandleText_ReplyMediaSendsTextAndPhotos(t *testing.T) {
	out := &memOut{}
	ad := tg.New(newFacade(&memCases{}), out)
	ad.Menu = staticMenu{doc: tgmenudomain.MenuDocument{
		ID: "default",
		Items: []tgmenudomain.MenuItem{{
			ID: "btn-help", Label: "🆘 帮助", Row: 0, Col: 0, Enabled: true,
			Action: tgmenudomain.ActionReplyMedia,
			Reply: &tgmenudomain.ReplyPayload{
				Text:   "hi",
				Images: []string{"https://example.com/a.png", "https://example.com/b.png"},
			},
		}},
	}}
	if err := ad.HandleText(context.Background(), 1, "🆘 帮助", "u"); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) == 0 || out.texts[0] != "hi" {
		t.Fatalf("texts=%v", out.texts)
	}
	if len(out.photoURLs) != 2 { // memOut 需记录 SendPhotoURL
		t.Fatalf("photos=%v", out.photoURLs)
	}
}

func TestHandleText_OpenCaseGoesToPreview(t *testing.T) {
	cases := &memCases{}
	_ = cases.Create(context.Background(), sampleCase("c1", "C1"))
	out := &memOut{}
	ad := tg.New(newFacade(cases), out)
	ad.Menu = staticMenu{doc: tgmenudomain.MenuDocument{Items: []tgmenudomain.MenuItem{{
		ID: "btn", Label: "Go", Enabled: true, Action: tgmenudomain.ActionOpenCase, CaseID: "c1",
	}}}}
	if err := ad.HandleText(context.Background(), 1, "Go", "u"); err != nil {
		t.Fatal(err)
	}
	if len(out.inlines) == 0 || !strings.Contains(out.inlines[0], "C1") {
		t.Fatalf("inlines=%v", out.inlines)
	}
}
```

同步扩展 `memOut` 实现 `SendPhotoURL`，并让现有 `SendPhoto` 测试仍编译。

另加表驱动：给定 Menu → `BuildReplyKeyboard` 行文案顺序。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/channel/tg/ -run 'TestHandleText_ReplyMedia|TestHandleText_OpenCase|TestBuildReplyKeyboard' -count=1`

Expected: FAIL

- [x] **Step 3: 最小实现**（按上面 Interfaces）

注意：`HandleUserNotify` 里硬编码 `BtnImage` 文案可暂保留（种子默认仍是该 label）；不必本期改为动态查找。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/channel/tg/ -count=1`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/channel/tg/
git commit -m "$(cat <<'EOF'
feat(tg): drive main keyboard and actions from menu config

EOF
)"
```

---

### Task 5: 组合根注入 — bot + admin-api 迁移 Menu 表

**Files:**
- Modify: `apps/bot/cmd/comfyui-bot/main.go`
- Modify: `apps/admin-api/cmd/admin-api/main.go`
- Modify: `apps/admin-api/internal/server/server.go`
- Modify: `apps/admin-api/internal/server/server_test.go`（若 Options 变更导致编译失败）

**Interfaces:**
- Consumes: Task 2–4 的 Store / Service / MenuReader
- Produces: 两端 AutoMigrate `&tgmenupersist.MenuRow{}`；bot 将 `application.Service`（或 Store）注入 `tgAdapter.Menu` 与 `BotMessenger.Menu`

- [x] **Step 1: bot main 接线**

在 `appboot.Options.Models` 追加 `&tgmenupersist.MenuRow{}`。

```go
menuStore := tgmenupersist.NewGormRepository(gdb)
menuSvc := &tgmenuapp.Service{
	Store: menuStore,
	Cases: tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
}
// Bot 运行时只读：
type menuReader struct{ svc *tgmenuapp.Service }
func (m menuReader) GetMenu(ctx context.Context) (tgmenudomain.MenuDocument, error) {
	return m.svc.Get(ctx)
}
```

在 `*tgAdapter = *tg.New(...)` 之后：`tgAdapter.Menu = menuReader{svc: menuSvc}`；若 `BotMessenger` 需要 Menu，同样赋值。

- [x] **Step 2: 编译 bot**

Run: `go build -o /dev/null ./apps/bot/cmd/comfyui-bot`

Expected: PASS

- [x] **Step 3: Commit bot 接线**

```bash
git add apps/bot/cmd/comfyui-bot/main.go
git commit -m "$(cat <<'EOF'
feat(bot): wire tgmenu repository into telegram adapter

EOF
)"
```

（admin-api 挂载放到 Task 6 同一提交或下一任务，避免半挂载。）

---

### Task 6: httpapi tgmenu — GET/PUT + 挂载 admin-api

**Files:**
- Create: `internal/httpapi/tgmenu/handler.go`
- Create: `internal/httpapi/tgmenu/handler_test.go`
- Modify: `apps/admin-api/internal/server/server.go`
- Modify: `apps/admin-api/cmd/admin-api/main.go`
- Modify: `apps/admin-api/README.md`
- Modify: `apps/admin-api/internal/server/server_test.go`

**Interfaces:**
- Produces:

```go
type Handler struct{ Svc *application.Service } // 或接口 Get/Replace

func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.get)
	r.Put("/", h.put)
}
```

DTO：

```go
type menuDTO struct {
	ID        string             `json:"id"`
	Items     []domain.MenuItem  `json:"items"`
	UpdatedAt time.Time          `json:"updated_at"`
}
type putBody struct {
	Items []domain.MenuItem `json:"items"`
}
```

- GET → 200 + DTO（调用 `Svc.Get`，触发懒种子）
- PUT → 解码 body；`Svc.Replace`；`ErrValidation` → 400 `{"error":"..."}`；其它 → 500；成功 200 + 最新 DTO
- **禁止** import `channel/tg`

- [x] **Step 1: 写失败测试**

模式对齐 `internal/httpapi/cases/handler_test.go`：内存 SQLite + 可选预置一个 catalog case。

```go
func TestTgMenuHandler_GetPutValidation(t *testing.T) {
	srv := openTgMenuServer(t) // migrate menu + cases；seed one case "c1"
	res, err := http.Get(srv.URL + "/api/v1/tg-menu")
	// expect 200, 6 items
	body := map[string]any{
		"items": []map[string]any{{
			"id": "btn-image", "label": "🖼 图片", "row": 0, "col": 0, "enabled": true,
			"action": "open_case", "case_id": "missing",
		}},
	}
	// PUT → 400
	body["items"].([]map[string]any)[0]["case_id"] = "c1"
	// PUT → 200；GET 可见
	// PUT reply_media with "ftp://x" → 400
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/tgmenu/ -count=1`

Expected: FAIL

- [x] **Step 3: 实现 handler + 挂载**

`server.Options` 增加 `TGMenu *tgmenuapi.Handler`；

```go
r.Route("/api/v1/tg-menu", func(r chi.Router) {
	if opts.TGMenu != nil {
		opts.TGMenu.Mount(r)
	}
})
```

`main.go`：Models 加 `MenuRow`；组装 `menuSvc` + Handler。

README 追加 curl 示例与无鉴权警示。

- [x] **Step 4: 测试通过**

Run:

```bash
go test ./internal/httpapi/tgmenu/ ./apps/admin-api/... -count=1
go build -o /dev/null ./apps/admin-api/cmd/admin-api
```

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/httpapi/tgmenu/ apps/admin-api/
git commit -m "$(cat <<'EOF'
feat(admin-api): expose GET/PUT /api/v1/tg-menu

EOF
)"
```

---

### Task 7: web/admin — 侧栏 + i18n

**Files:**
- Modify: `web/admin/src/config/menu.ts`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify（若有断言侧栏项数）: 相关 layout / menu 测试

**Interfaces:**
- Produces: 侧栏在 Case 后增加一项：

```ts
{ id: 'tg-menu', titleKey: 'menu.tgMenu', path: '/tg-menu', icon: Menu } // lucide-react Menu 或 Keyboard
```

文案：

- zh: `menu.tgMenu` = `TG 菜单`；页面键 `tgMenu.title` / `tgMenu.description` 等
- en: `TG Menu`

- [x] **Step 1: 改 menu.ts + locales**

- [x] **Step 2: 跑前端既有测试**

Run: `cd web/admin && pnpm test -- --run src/config src/lib/i18n 2>/dev/null || pnpm exec vitest run src/lib/i18n`

（以仓库实际 test 脚本为准：`pnpm test` / `vitest`。）

- [x] **Step 3: Commit**

```bash
git add web/admin/src/config/menu.ts web/admin/src/lib/i18n/locales/
git commit -m "$(cat <<'EOF'
feat(admin-web): add TG Menu sidebar entry and i18n

EOF
)"
```

---

### Task 8: web/admin — TG Menu 管理页

**Files:**
- Create: `web/admin/src/lib/api/tg-menu.ts`
- Modify: `web/admin/src/lib/api/query-keys.ts` — `tgMenu: { all: ['tg-menu'] }`
- Modify: `web/admin/src/lib/api/types.ts`（或就地定义 Menu 类型）
- Create: `web/admin/src/features/tg-menu/menu-editor.tsx`
- Create: `web/admin/src/routes/_app/tg-menu/index.tsx`（或 `route.tsx`）
- Modify: `web/admin/README.md` — 去掉「无 TG/Menu」非目标；补充联调 Menu
- 路由树：按项目惯例跑 codegen / 手写后 `pnpm exec tsr generate`（若仓库用 TanStack Router 插件自动生成则保存文件即可）

**Interfaces:**
- API：

```ts
export type MenuAction = 'open_case' | 'list_cases_by_tag' | 'placeholder' | 'reply_media'
export type MenuItem = {
  id: string
  label: string
  row: number
  col: number
  enabled: boolean
  action: MenuAction
  case_id?: string
  tag?: string
  placeholder_text?: string
  reply?: { text?: string; images?: string[] }
}
export type TgMenuDocument = { id: string; items: MenuItem[]; updated_at: string }

export function getTgMenu() {
  return apiFetch<TgMenuDocument>('/api/v1/tg-menu')
}
export function putTgMenu(items: MenuItem[]) {
  return apiFetch<TgMenuDocument>('/api/v1/tg-menu', {
    method: 'PUT',
    body: JSON.stringify({ items }),
  })
}
```

- UI（单页，非 Master–Detail）：
  - 加载 GET → 表格编辑各项：label / row / col / enabled / action + 条件字段
  - `open_case`：`listCases({ limit: 200 })` 下拉
  - `list_cases_by_tag`：tag 输入
  - `placeholder`：placeholder_text 输入
  - `reply_media`：textarea（text）+ 图片 URL 多行（增删）
  - 保存 → PUT；成功 `toast.success(t('common.successSaved'))` + invalidate query；失败 `ErrorBanner`，不假装成功
  - **无**文件上传控件；**无** mock

- [x] **Step 1: 实现 api 客户端 + 页面骨架**

- [x] **Step 2: 补齐条件字段与保存反馈**

参考 `features/cases/case-form.tsx` 的 mutation / ErrorBanner / toast 模式。

- [x] **Step 3: 类型检查**

Run: `cd web/admin && pnpm exec tsc -p tsconfig.json --noEmit`

Expected: PASS

- [x] **Step 4: Commit**

```bash
git add web/admin/src/lib/api/tg-menu.ts web/admin/src/lib/api/query-keys.ts web/admin/src/features/tg-menu/ web/admin/src/routes/_app/tg-menu/ web/admin/src/routeTree.gen.ts web/admin/README.md
git commit -m "$(cat <<'EOF'
feat(admin-web): add TG Menu editor page backed by admin-api

EOF
)"
```

---

### Task 9: 架构文档同步

**Files:**
- Modify: `docs/architecture/data-model.md` — 新增 `### 2.6 tg_menu_configs`；ER 增加该表（可无 FK 边，仅标注逻辑真相源）；§6 代码入口表加一行
- Modify（一句）: `docs/architecture/runtime.md` — 「主菜单来自 `tg_menu_configs`（空则种子）」
- Modify: `docs/architecture/bounded-contexts.md` — 模块地图增加 `tgmenu`；HTTP API 职责含 Menu；依赖：`channel/tg` → `tgmenu` ← `httpapi/tgmenu`

- [x] **Step 1: 按 architecture-docs-sync 更新三处文档**

表字段与设计一致：`id` PK、`items_json`、`updated_at`。

- [x] **Step 2: Commit**

```bash
git add docs/architecture/data-model.md docs/architecture/runtime.md docs/architecture/bounded-contexts.md
git commit -m "$(cat <<'EOF'
docs(architecture): document tg_menu_configs and menu read path

EOF
)"
```

---

### Task 10: 端到端验收清单（手工 + 自动）

**Files:** 无新代码；必要时修测试缝隙

- [x] **Step 1: 自动回归**

```bash
go test ./internal/tgmenu/... ./internal/httpapi/tgmenu/... ./internal/channel/tg/... ./apps/admin-api/... -count=1
go build -o /dev/null ./apps/bot/cmd/comfyui-bot ./apps/admin-api/cmd/admin-api
cd web/admin && pnpm exec tsc -p tsconfig.json --noEmit
```

Expected: 全 PASS

- [x] **Step 2: 手工联调（记录到 verify 时再用）**

1. `make run-all`（或 bot + `make run-admin`），共用同一 `data/app.db`
2. 打开控制台侧栏「TG 菜单」→ 见种子六项
3. 改某 label / 将一项改为 `reply_media`（text + https 图）→ 保存成功
4. TG `/start`：键盘文案已更新；点击 reply_media 收到文字与图
5. 将一项设为 `open_case` 绑已有 Case → 点击进入预览
6. DevTools Network：仅 `VITE_ADMIN_API_BASE` + `/api/v1/tg-menu` 与 `/api/v1/cases`
7. PUT 非法 `case_id` → 400，库中配置不变

- [x] **Step 3: 若手工发现缺口，修代码并追加测试后单独 commit**

---

## Self-Review（写计划时已核对）

| Spec / Design 要求 | 覆盖任务 |
|---|---|
| Menu 持久化 + 种子 | Task 1–2 |
| 校验 open_case / reply_media / label | Task 1、3、6 |
| Bot 键盘与动作（含 reply_media） | Task 4–5 |
| admin-api GET/PUT、不依赖 channel/tg | Task 6 |
| 控制台侧栏 + 页 + i18n + 无 mock | Task 7–8 |
| data-model / runtime 文档 | Task 9 |
| 刷新策略（下次构建键盘） | Task 4（每次读仓储） |

无 TBD/占位步骤；类型名在任务间一致：`MenuDocument` / `MenuItem` / `ActionReplyMedia` / `SendPhotoURL` / `/api/v1/tg-menu`。
