---
change: tg-menu-config
design-doc: docs/superpowers/specs/2026-08-09-tg-menu-config-design.md
supersedes-flat-plan-for: tree-upgrade
prior-flat-plan: docs/superpowers/plans/2026-08-09-tg-menu-config.md
base-ref: 6e0d1afba44e9ffabcfc621da960b691a0b21ab7
---

# TG Menu 树形主键盘 + Case 关联 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把已落地的扁平 `tg_menu_configs` 升级为关系表树（文件夹下钻 + 菜单项↔Case），Bot 支持 Inline 分层浏览，admin「主键盘」可编辑树，Case 详情可反查挂载路径。

**Architecture:** 在 `internal/tgmenu` 用 `tg_menus` / `tg_menu_items` / `tg_menu_item_cases` 替换 JSON 单表；领域以扁平 `MenuItem`（含 `ParentID`/`Kind`/`CaseIDs`）存储，API/UI 用嵌套 `MenuNode`；`channel/tg` 根层 ReplyKeyboard + `mf:`/`mb:` callback 进文件夹；placements 经 admin-api 独立 GET。启动时若新表空则从种子（挂载 tag=image 的 Case）或旧 JSON 迁移。

**Tech Stack:** Go、GORM/SQLite、chi、catalog List、go-telegram/bot、React + TanStack Query、react-i18next

## Global Constraints

- 产物语言：zh-CN；继续 OpenSpec change `tg-menu-config`
- **单 Bot 运行**；`bot_id` / 菜单 id 恒 `"default"`；不做多 Bot UI / Token CRUD
- **禁止** `apps/admin-api` / `internal/httpapi/*` import `channel/tg`
- **禁止** Case.categories 当文件夹来源；文件夹只来自菜单树
- **禁止** 管理端图片本地上传；`reply_media` 仅 http(s) URL
- **禁止** 前端 mock 验收；Network 只打 `VITE_ADMIN_API_BASE`
- PUT 整棵树事务替换；校验失败不写；读失败 Bot 回退 `DefaultSeedTree()`
- 树最大深度 **5**（根 depth=0）；同层 label trim 唯一；至少 1 个启用根项
- callback data ≤64 字节：`mf:<itemID>` / `mb:<parentID|root>`；Case 继续用既有 `CBCasePreview`
- 触及表/主链路时同步 `docs/architecture/{data-model,runtime,bounded-contexts}.md`
- TDD：先红后绿；每任务末提交；不碰无关脏文件（filters / admin-shell-sidebar-footer 等）

---

## 文件结构（锁定职责）

| 路径 | 职责 |
|---|---|
| `internal/tgmenu/domain/document.go` | `MenuKind`、`MenuItem`（ParentID/CaseIDs）、`MenuTree`/`MenuNode`、`MenuPlacement`；`BotIDDefault` |
| `internal/tgmenu/domain/tree.go` | `Flatten` / `BuildTree` / `RootItems` / `DepthOf` / `PathTo` |
| `internal/tgmenu/domain/validate.go` | 树校验（含 folder/open_case 关联规则） |
| `internal/tgmenu/domain/seed.go` | `DefaultSeedTree()`：「图片」=`folder`，其余 placeholder |
| `internal/tgmenu/domain/repository.go` | `GetTree` / `ReplaceTree` / `ListPlacementsByCase` |
| `internal/tgmenu/application/service.go` | Get/Replace/ListPlacements；EnsureDefault 挂 image Case |
| `internal/tgmenu/infrastructure/persistence/gorm_repository.go` | 三表 + 旧表迁移 + 事务 Replace |
| `internal/httpapi/tgmenu/handler.go` | GET/PUT 树 DTO；`listPlacements` |
| `apps/admin-api/internal/server/server.go` | 挂 `GET /api/v1/cases/{id}/menu-placements` |
| `internal/channel/tg/menu_runtime.go` + `adapter.go` | 根键盘；folder Inline；callback |
| `web/admin/src/lib/api/tg-menu.ts` | 树类型 + placements API |
| `web/admin/src/features/tg-menu/menu-editor.tsx` | 树编辑 UI |
| `web/admin/src/features/cases/detail-panel.tsx` + 小节 | 「出现在主键盘」只读 |
| `docs/architecture/*`、admin README | 三表 / 树浏览说明 |

---

### Task 1: Domain — 树模型、Flatten/BuildTree、校验、种子

**Files:**
- Modify: `internal/tgmenu/domain/document.go`
- Create: `internal/tgmenu/domain/tree.go`
- Modify: `internal/tgmenu/domain/validate.go`
- Modify: `internal/tgmenu/domain/seed.go`
- Modify: `internal/tgmenu/domain/repository.go`
- Modify: `internal/tgmenu/domain/validate_test.go`
- Create: `internal/tgmenu/domain/tree_test.go`
- Create: `internal/tgmenu/domain/seed_test.go`

**Interfaces:**
- Produces:
  - `const BotIDDefault = "default"`（与 `DocumentIDDefault` 同值可用）
  - `type MenuKind string`：`folder` \| `open_case` \| `placeholder` \| `reply_media` \| `list_cases_by_tag`
  - `type MenuItem struct { ID, ParentID, Label string; Row, Col int; Enabled bool; Kind MenuKind; CaseIDs []string; Tag, PlaceholderText string; Reply *ReplyPayload }`
  - `type MenuNode struct { /* 同 MenuItem 导出字段 */ Children []MenuNode }`（JSON：`kind`,`case_ids`,`children`）
  - `type MenuTree struct { ID, BotID string; Items []MenuNode; UpdatedAt time.Time }`
  - `type MenuPlacement struct { MenuID, ItemID string; Path []PlacementStep }`；`PlacementStep { ID, Label string }`
  - `func Flatten(nodes []MenuNode) []MenuItem`
  - `func BuildTree(items []MenuItem) ([]MenuNode, error)`
  - `func Validate(ctx, tree MenuTree, caseExists CaseExistsFunc) error`
  - `func DefaultSeedTree() MenuTree`
  - `type Repository interface { GetTree(ctx, id string) (MenuTree, error); ReplaceTree(ctx, tree MenuTree) error; ListPlacementsByCase(ctx, caseID string) ([]MenuPlacement, error) }`
- Notes: 删除/停用扁平 `MenuDocument`+`Action` 的对外 API；过渡期可在 persistence 迁移里读旧 JSON `action`/`case_id`。保留 `ErrValidation`/`ErrNotFound`。

- [ ] **Step 1: 写失败测试（种子 + 树校验）**

```go
func TestDefaultSeedTree_ImageIsFolder(t *testing.T) {
	tree := domain.DefaultSeedTree()
	if tree.ID != domain.DocumentIDDefault || tree.BotID != domain.BotIDDefault {
		t.Fatalf("ids: %+v", tree)
	}
	roots := tree.Items
	if len(roots) != 6 {
		t.Fatalf("roots=%d", len(roots))
	}
	img := roots[0]
	if img.ID != "btn-image" || img.Kind != domain.KindFolder {
		t.Fatalf("want folder btn-image, got %+v", img)
	}
	if img.ParentID != "" {
		t.Fatal("root must have empty parent")
	}
}

func TestValidate_FolderCaseMustExist(t *testing.T) {
	tree := domain.DefaultSeedTree()
	tree.Items[0].CaseIDs = []string{"missing"}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidate_OpenCaseNeedsExactlyOne(t *testing.T) {
	tree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "x", Label: "X", Enabled: true, Kind: domain.KindOpenCase, CaseIDs: nil,
		}},
	}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildTree_RoundTrip(t *testing.T) {
	flat := []domain.MenuItem{
		{ID: "r", Label: "R", Enabled: true, Kind: domain.KindFolder, Row: 0, Col: 0},
		{ID: "c", ParentID: "r", Label: "C", Enabled: true, Kind: domain.KindFolder, Row: 0, Col: 0},
	}
	nodes, err := domain.BuildTree(flat)
	if err != nil || len(nodes) != 1 || len(nodes[0].Children) != 1 {
		t.Fatalf("%v %+v", err, nodes)
	}
	back := domain.Flatten(nodes)
	if len(back) != 2 {
		t.Fatalf("%d", len(back))
	}
}
```

- [ ] **Step 2: 跑测确认失败**

Run: `go test ./internal/tgmenu/domain/ -count=1`
Expected: FAIL（缺 Kind/BuildTree/新 Validate 等）

- [ ] **Step 3: 实现最小领域代码**

`document.go` 关键：

```go
const BotIDDefault = "default"
const MaxTreeDepth = 5

type MenuKind string
const (
	KindFolder           MenuKind = "folder"
	KindOpenCase         MenuKind = "open_case"
	KindPlaceholder      MenuKind = "placeholder"
	KindReplyMedia       MenuKind = "reply_media"
	KindListCasesByTag   MenuKind = "list_cases_by_tag"
)

type MenuItem struct {
	ID, ParentID, Label string
	Row, Col            int
	Enabled             bool
	Kind                MenuKind
	CaseIDs             []string
	Tag, PlaceholderText string
	Reply               *ReplyPayload
}

type MenuNode struct {
	ID, ParentID, Label string `json:"id"`
	// ... same fields with json tags; Children []MenuNode `json:"children,omitempty"`
	CaseIDs  []string   `json:"case_ids,omitempty"`
	Children []MenuNode `json:"children,omitempty"`
}
```

`Validate` 要点：
- Flatten 后 id 全局唯一；**同 ParentID** 下 label 唯一（不是全局）
- 启用根项 ≥1
- depth ≤ MaxTreeDepth
- `folder`：CaseIDs 每个须存在；允许 0 Case
- `open_case`：恰好 1 CaseID 且存在；无 children
- `list_cases_by_tag`：Tag 非空；CaseIDs 空
- `placeholder` / `reply_media`：同旧规则；CaseIDs 空
- 未知 parent_id → ErrValidation

`DefaultSeedTree`：六根；`btn-image` KindFolder、CaseIDs 空（挂载在 EnsureDefault）。

- [ ] **Step 4: 跑测通过**

Run: `go test ./internal/tgmenu/domain/ -count=1`
Expected: PASS（更新/删除依赖旧 `Action`/`MenuDocument` 的测试）

- [ ] **Step 5: Commit**

```bash
git add internal/tgmenu/domain/
git commit -m "$(cat <<'EOF'
feat(tgmenu): introduce tree domain model and seed folder

Replace flat Action documents with Kind/ParentID/CaseIDs trees so
folder navigation and case links can be validated before persistence.
EOF
)"
```

---

### Task 2: Persistence — 三表、事务 Replace、旧 JSON 迁移、Placements

**Files:**
- Rewrite: `internal/tgmenu/infrastructure/persistence/gorm_repository.go`
- Rewrite: `internal/tgmenu/infrastructure/persistence/gorm_repository_test.go`
- Modify: `apps/bot/cmd/comfyui-bot/main.go`、`apps/admin-api/cmd/admin-api/main.go`（AutoMigrate 新模型）

**Interfaces:**
- Consumes: `domain.MenuTree`、`Flatten`/`BuildTree`、`DefaultSeedTree`
- Produces:
  - `type MenuHeaderRow struct { ID, BotID string; UpdatedAt time.Time }` → `tg_menus`
  - `type MenuItemRow struct { ID, MenuID string; ParentID *string; Label string; Row, Col int; Enabled bool; Kind string; PlaceholderText, Tag, ReplyJSON string }` → `tg_menu_items`
  - `type MenuItemCaseRow struct { MenuItemID, CaseID string; Sort int }` → `tg_menu_item_cases`；唯一 `(menu_item_id, case_id)`
  - `EnsureDefault(ctx, listImageCaseIDs func(ctx) ([]string, error)) (MenuTree, error)`
  - 启动：`AutoMigrate` 三表；`MigrateFromLegacyIfNeeded`：若 `tg_menus` 无行且 `tg_menu_configs` 有 JSON，映射 `action→kind`、`case_id→case_ids` 后写入新表

- [ ] **Step 1: 写失败测试**

```go
func TestReplaceTree_RoundTripAndPlacements(t *testing.T) {
	db := openTestDB(t)
	repo := persistence.NewGormRepository(db)
	require.NoError(t, db.AutoMigrate(
		&persistence.MenuHeaderRow{}, &persistence.MenuItemRow{}, &persistence.MenuItemCaseRow{},
	))
	tree := domain.DefaultSeedTree()
	tree.Items[0].CaseIDs = []string{"case-a"}
	tree.Items[0].Children = []domain.MenuNode{{
		ID: "folder-x", ParentID: "btn-image", Label: "子夹", Enabled: true, Kind: domain.KindFolder,
	}}
	require.NoError(t, repo.ReplaceTree(context.Background(), tree))
	got, err := repo.GetTree(context.Background(), domain.DocumentIDDefault)
	require.NoError(t, err)
	require.Equal(t, []string{"case-a"}, got.Items[0].CaseIDs)
	require.Len(t, got.Items[0].Children, 1)

	ps, err := repo.ListPlacementsByCase(context.Background(), "case-a")
	require.NoError(t, err)
	require.Len(t, ps, 1)
	require.Equal(t, "btn-image", ps[0].ItemID)
	require.Equal(t, "🖼 图片", ps[0].Path[len(ps[0].Path)-1].Label)
}
```

- [ ] **Step 2: 跑测确认失败**

Run: `go test ./internal/tgmenu/infrastructure/persistence/ -count=1`
Expected: FAIL

- [ ] **Step 3: 实现仓储**

`ReplaceTree`：事务内 `DELETE` 该 menu 的 item_cases → items → upsert header → 按 Flatten 插入 items + cases（sort=下标）。

`GetTree`：读 header + items + cases → `BuildTree`。

`EnsureDefault`：
1. GetTree 成功则返回
2. 否则尝试 `migrateLegacyJSON`
3. 否则 `DefaultSeedTree()`；若 `listImageCaseIDs != nil`，把返回 id 写入 `btn-image.CaseIDs`
4. ReplaceTree 后 GetTree

`migrateLegacyJSON`：读 `tg_menu_configs`；每项 `action`→`kind`；若有 `case_id` 填 `CaseIDs`；全部 ParentID 空。

- [ ] **Step 4: 跑测通过**

Run: `go test ./internal/tgmenu/infrastructure/persistence/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tgmenu/infrastructure/persistence/ apps/bot/cmd/comfyui-bot/main.go apps/admin-api/cmd/admin-api/main.go
git commit -m "$(cat <<'EOF'
feat(tgmenu): persist menu tree in relational tables

Store menus/items/case links transactionally, migrate legacy JSON once,
and support case placement reverse lookup.
EOF
)"
```

---

### Task 3: Application — GetTree / ReplaceTree / ListPlacements

**Files:**
- Rewrite: `internal/tgmenu/application/service.go`
- Modify: `internal/tgmenu/application/case_checker.go`（可保留）
- Rewrite: `internal/tgmenu/application/service_test.go`
- Create（可选）: `internal/tgmenu/application/image_case_lister.go` — 适配 catalog `List(Tag:"image")`

**Interfaces:**
- Consumes: persistence Store、`CaseChecker`
- Produces:
  - `type Store interface { domain.Repository; EnsureDefault(ctx, listImage ...) (MenuTree, error) }`
  - `func (s *Service) Get(ctx) (MenuTree, error)`
  - `func (s *Service) Replace(ctx, tree MenuTree) (MenuTree, error)` — 强制 `ID/BotID=default`，Validate，ReplaceTree，再 Get
  - `func (s *Service) ListPlacementsByCase(ctx, caseID string) ([]MenuPlacement, error)`

- [ ] **Step 1: 写失败测试**（Replace 非法 folder case → ErrValidation；合法后 Get 一致）

- [ ] **Step 2: 跑测失败** — `go test ./internal/tgmenu/application/ -count=1`

- [ ] **Step 3: 改 Service 签名与实现**；组合根把 catalog List 包一层传给 EnsureDefault

- [ ] **Step 4: 全包通过** — `go test ./internal/tgmenu/... -count=1`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(tgmenu): application API for tree get/replace/placements"
```

---

### Task 4: HTTP — 树 GET/PUT + Case placements

**Files:**
- Rewrite: `internal/httpapi/tgmenu/handler.go`
- Rewrite: `internal/httpapi/tgmenu/handler_test.go`
- Modify: `apps/admin-api/internal/server/server.go`
- Modify: `apps/admin-api/README.md`

**Interfaces:**
- DTO：

```go
type treeDTO struct {
	ID        string            `json:"id"`
	BotID     string            `json:"bot_id"`
	Items     []domain.MenuNode `json:"items"`
	UpdatedAt time.Time         `json:"updated_at"`
}
type putBody struct {
	Items []domain.MenuNode `json:"items"`
}
type placementDTO struct {
	MenuID string `json:"menu_id"`
	ItemID string `json:"item_id"`
	Path   []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	} `json:"path"`
}
```

- Routes：
  - 仍 `Mount`：`GET/PUT /` under `/api/v1/tg-menu`
  - `ListPlacements(w,r)`；在 `server.go`：`r.Get("/api/v1/cases/{id}/menu-placements", opts.TGMenu.ListPlacements)`（须在能取到 `{id}` 的位置注册；与 cases 路由并存）

- [ ] **Step 1: 写 handler 测试** — PUT 带 folder+case_ids 200；非法 case 400；GET placements 返回 path

- [ ] **Step 2: 跑测失败** — `go test ./internal/httpapi/tgmenu/ -count=1`

- [ ] **Step 3: 实现 handler + server 挂载**；更新 README curl 示例为树 JSON

- [ ] **Step 4: 通过** — `go test ./internal/httpapi/tgmenu/ ./apps/admin-api/... -count=1`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(admin-api): expose tree menu and case menu-placements"
```

---

### Task 5: Bot — 根键盘 + 文件夹 Inline 下钻

**Files:**
- Modify: `internal/channel/tg/menu_runtime.go`
- Modify: `internal/channel/tg/adapter.go`（`dispatchMenuItem`、`HandleCallback`）
- Modify: `internal/channel/tg/adapter_test.go`
- Modify: `internal/channel/tg/bot.go`（若 MenuReader 签名变）
- 常量：`CBMenuFolder = "mf:"`，`CBMenuBack = "mb:"`（`mb:root` = 主菜单）

**Interfaces:**
- `MenuReader.GetMenu(ctx) (MenuTree, error)`（改返回类型）
- `BuildReplyKeyboard(tree)`：仅 **根** enabled 项
- `FindEnabledRootByLabel(tree, label) (MenuNode, bool)`
- `func (a *Adapter) showMenuFolder(ctx, chatID, itemID string) error`：子 folder 按钮文案 `"📁 "+label` Data=`mf:`+id；Case 按钮沿用 `CBCasePreview+id` 文案 `name · ¥price`（需 `App.GetCase` 或 List）；末行 `⬅️ 返回` → `mb:`+parent 或 `mb:root`
- `dispatchMenuItem`：`KindFolder` → `showMenuFolder`；其余同旧（`list_cases_by_tag` 保留）
- `HandleCallback`：解析 `mf:` / `mb:`

- [ ] **Step 1: 写测试**

```go
func TestDispatch_FolderShowsInlineChildrenAndCases(t *testing.T) {
	// MenuReader 返回根 folder btn-image，CaseIDs=["c1"]，无子文件夹
	// 点「🖼 图片」→ Out.SendInline 被调用，按钮含 Case 与「返回」
}

func TestCallback_MenuFolderAndBack(t *testing.T) {
	// HandleCallback mf:btn-image → Inline；mb:root → sendMainMenu
}
```

- [ ] **Step 2: 跑测失败** — `go test ./internal/channel/tg/ -run 'Folder|MenuFolder|BuildReplyKeyboard' -count=1`

- [ ] **Step 3: 实现运行时**；修编译破损（所有 `MenuDocument`/`Action` 引用）

- [ ] **Step 4: 全 tg 测试通过** — `go test ./internal/channel/tg/ -count=1`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(channel-tg): navigate menu folders via inline keyboards"
```

---

### Task 6: Admin — 主键盘树编辑 UI

**Files:**
- Rewrite: `web/admin/src/lib/api/tg-menu.ts`
- Rewrite: `web/admin/src/features/tg-menu/menu-editor.tsx`
- Modify: i18n `zh.json` / `en.json`（folder、挂载 Case、子项）
- 可选 contract test：`menu-editor` / api 类型 smoke

**Interfaces:**

```ts
export type MenuKind = 'folder' | 'open_case' | 'placeholder' | 'reply_media' | 'list_cases_by_tag'
export type MenuNode = {
  id: string
  parent_id?: string
  label: string
  row: number
  col: number
  enabled: boolean
  kind: MenuKind
  case_ids?: string[]
  tag?: string
  placeholder_text?: string
  reply?: { text?: string; images?: string[] }
  children?: MenuNode[]
}
export type TgMenuTree = { id: string; bot_id: string; items: MenuNode[]; updated_at: string }
export function getTgMenu() { return apiFetch<TgMenuTree>('/api/v1/tg-menu') }
export function putTgMenu(items: MenuNode[]) {
  return apiFetch<TgMenuTree>('/api/v1/tg-menu', { method: 'PUT', body: JSON.stringify({ items }) })
}
```

UI（保持 MasterDetailShell）：
- 左：扁平化树列表（缩进显示深度）或仅根+选中路径；选中节点
- 右：编辑 kind、label、row/col、enabled；`folder`/`open_case` 用 Case 多选/单选（`listCases`）；`folder` 可「添加子项」
- 保存：`putTgMenu(roots)`；错误 ErrorBanner

- [ ] **Step 1: 改 API 类型后 `pnpm exec tsc --noEmit` 会红**（先改类型再改编辑器）

- [ ] **Step 2: 实现编辑器与 i18n**

- [ ] **Step 3: 校验**

Run: `cd web/admin && pnpm exec tsc --noEmit && pnpm exec vitest run src/features/tg-menu src/lib/api --passWithNoTests`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(admin-web): tree editor for keyboard menu folders"
```

---

### Task 7: Case 详情 —「出现在主键盘」

**Files:**
- Modify: `web/admin/src/lib/api/tg-menu.ts`（或 `cases.ts`）加 `getCaseMenuPlacements(id)`
- Modify: `web/admin/src/lib/api/query-keys.ts` + test
- Create: `web/admin/src/features/cases/sections/menu-placements.tsx`
- Modify: `web/admin/src/features/cases/detail-panel.tsx`
- i18n keys：`cases.menuPlacementsTitle` 等

```ts
export type MenuPlacement = {
  menu_id: string
  item_id: string
  path: { id: string; label: string }[]
}
export function getCaseMenuPlacements(caseId: string) {
  return apiFetch<MenuPlacement[]>(`/api/v1/cases/${encodeURIComponent(caseId)}/menu-placements`)
}
```

展示：路径用 `path.map(p => p.label).join(' / ')`；空则「未挂到主键盘」。

- [ ] **Step 1–3: 实现 + tsc/vitest**

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(admin-web): show keyboard placements on case detail"
```

---

### Task 8: 架构文档 + 收尾回归

**Files:**
- Modify: `docs/architecture/data-model.md` — 替换 `tg_menu_configs` 为三表 + ER + placements
- Modify: `docs/architecture/runtime.md` — 文件夹 Inline 浏览
- Modify: `docs/architecture/bounded-contexts.md` — tgmenu 树与反查
- Modify: `docs/openspec/changes/tg-menu-config/tasks.md` — 勾选 §6 完成项
- Modify: `apps/admin-api/README.md` / `web/admin/README.md`（若未在前任务写完）

- [ ] **Step 1: 改文档**

- [ ] **Step 2: 回归**

```bash
go test ./internal/tgmenu/... ./internal/httpapi/tgmenu/... ./internal/channel/tg/... ./apps/admin-api/... -count=1
cd web/admin && pnpm exec tsc --noEmit && pnpm exec vitest run
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git commit -m "docs(architecture): document relational tg menu tree"
```

---

## Spec coverage（自检）

| Spec / Design | Task |
|---|---|
| 三表树持久化 | 1–2 |
| folder / open_case / reply_media 校验 | 1 |
| 种子图片=folder + 挂 image Case | 1–2–3 |
| GET/PUT 树 API | 4 |
| Case placements API | 4 |
| Bot ReplyKeyboard + Inline 下钻/返回 | 5 |
| 主键盘树编辑 | 6 |
| Case 详情挂载 | 7 |
| 架构文档 | 8 |
| 旧 JSON 迁移 | 2 |
| list_cases_by_tag 兼容 | 1、5 |
| 无多 Bot UI / 无 categories 文件夹 | Global Constraints |

## Placeholder scan

无 TBD；类型名在 Task 1 定义后各任务沿用 `MenuTree`/`MenuNode`/`MenuKind`/`MenuPlacement`。
