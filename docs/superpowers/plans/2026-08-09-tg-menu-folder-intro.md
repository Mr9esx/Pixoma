---
design-doc: docs/superpowers/specs/2026-08-09-tg-menu-folder-intro-design.md
branch: feat-init
---

# 主键盘每层说明 + 后台树编辑 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 文件夹可配「本层说明」并在 Telegram 进夹时展示；管理台左边改为可展开树，右边用直白话编辑这一层。

**Architecture:** 在现有 `MenuNode`/`tg_menu_items` 增加 `intro_text`；`showMenuFolder` 用 intro（空则 label），按钮顺序改为先 Case 再子文件夹再返回；`menu-editor` 改为树导航 + i18n 直白话，不改 PUT 整树语义。

**Tech Stack:** Go、GORM、channel/tg、React、i18n

## Global Constraints

- 产物语言：zh-CN
- 字段名 JSON：`intro_text`；仅 folder 有意义；长度 ≤ 3500；空合法
- 返回：沿用 `mb:` / `mb:root`；不实现 editMessage
- 不做：两列键盘、自定义✍️按钮、多 Bot
- 后台文案：见设计 §5（「本层说明」「本层模板」「下面的分类」「加一个分类」「目录」）
- TDD；禁止 httpapi import channel/tg；不碰无关脏文件
- 触及行为时必要时补一句 architecture/runtime

---

## 文件结构

| 路径 | 职责 |
|---|---|
| `internal/tgmenu/domain/document.go` + `tree.go` + `validate.go` | `IntroText` 字段贯通 Flatten/BuildTree；校验长度 |
| `internal/tgmenu/infrastructure/persistence/gorm_repository.go` | `MenuItemRow.IntroText` 列 |
| `internal/channel/tg/adapter.go` | `showMenuFolder` 正文与按钮顺序 |
| `web/admin/src/lib/api/tg-menu.ts` | `intro_text?: string` |
| `web/admin/src/features/tg-menu/menu-editor.tsx` | 左树 + 右层编辑 |
| `web/admin/src/lib/i18n/locales/{zh,en}.json` | 直白话 |
| `web/admin/src/routes/_app/tg-menu/index.tsx` | 页说明 |

---

### Task 1: Domain — IntroText + 校验

**Files:**
- Modify: `internal/tgmenu/domain/document.go`（`MenuItem`/`MenuNode` 加 `IntroText`，JSON `intro_text`）
- Modify: `internal/tgmenu/domain/tree.go`（Flatten/BuildTree 拷贝字段）
- Modify: `internal/tgmenu/domain/validate.go` + `validate_test.go`
- Constant: `MaxIntroTextLen = 3500`

**Interfaces:**
- Produces: `IntroText string` on item/node；Validate rejects folder intro longer than MaxIntroTextLen；non-folder with non-empty intro → ErrValidation（或 strip—选 **拒绝**）

- [ ] **Step 1: 写失败测试**

```go
func TestValidate_FolderIntroTooLong(t *testing.T) {
	long := strings.Repeat("x", domain.MaxIntroTextLen+1)
	tree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "f", Label: "F", Enabled: true, Kind: domain.KindFolder, IntroText: long,
		}},
	}
	err := domain.Validate(context.Background(), tree, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidate_NonFolderIntroRejected(t *testing.T) {
	tree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "p", Label: "P", Enabled: true, Kind: domain.KindPlaceholder, IntroText: "nope",
		}},
	}
	err := domain.Validate(context.Background(), tree, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 2:** `go test ./internal/tgmenu/domain/ -count=1` → FAIL  
- [ ] **Step 3:** 实现字段 + 校验 + tree 拷贝  
- [ ] **Step 4:** 测试 PASS  
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(tgmenu): add folder intro_text to domain model"
```

---

### Task 2: Persistence — 列读写

**Files:**
- Modify: `gorm_repository.go`（`IntroText string \`gorm:"type:text"\``；itemToRow/rowToItem）
- Modify: `gorm_repository_test.go`（round-trip 带 intro）

- [ ] **Step 1:** 测试 ReplaceTree 后 GetTree 保留 `IntroText`  
- [ ] **Step 2:** FAIL  
- [ ] **Step 3:** 实现列映射（AutoMigrate 已有则自动加列）  
- [ ] **Step 4:** `go test ./internal/tgmenu/infrastructure/persistence/ -count=1` PASS  
- [ ] **Step 5: Commit** `feat(tgmenu): persist menu item intro_text`

---

### Task 3: Bot — 展示 intro + Case 优先

**Files:**
- Modify: `internal/channel/tg/adapter.go` `showMenuFolder`
- Modify: `adapter_test.go`

行为：

```go
text := strings.TrimSpace(node.IntroText)
if text == "" {
	text = node.Label
}
// rows: CaseIDs first, then KindFolder children with 📁, then back
```

- [ ] **Step 1:** 测试：有 IntroText 时 SendInline 文本为 intro；空则 label；按钮顺序 Case 在 📁 前  
- [ ] **Step 2–4:** TDD 实现并通过 `go test ./internal/channel/tg/ -count=1`  
- [ ] **Step 5: Commit** `feat(channel-tg): show folder intro text above case list`

---

### Task 4: Admin API 贯通（若 handler 直接序列化 MenuNode）

**Files:**
- Modify/确认: `internal/httpapi/tgmenu`（通常零改；补 handler 测试 PUT/GET 含 intro_text）
- Modify: `apps/admin-api/README.md` 示例 JSON 加一行 intro_text

- [ ] **Step 1–4:** handler 测试读写 intro  
- [ ] **Step 5: Commit** `test(admin-api): cover tg-menu intro_text round-trip`

---

### Task 5: Admin UI — 树 + 直白话 + intro

**Files:**
- Modify: `web/admin/src/lib/api/tg-menu.ts`（`intro_text?: string`）
- Rewrite UX in: `web/admin/src/features/tg-menu/menu-editor.tsx`
- Modify: `zh.json` / `en.json` / 页 `tg-menu/index.tsx` description
- Update contract tests

**UI 要求（对照设计）：**

- 左栏标题：`目录`（`listTitle`）
- 文件夹可展开/折叠；缩进显示子分类；选中高亮
- 右栏字段标签：
  - `按钮上显示的名字` ← label  
  - `本层说明` ← intro_text（Textarea，仅 folder）  
  - `本层模板` ← case_ids  
  - `下面的分类` + `加一个分类` ← children / addChild  
  - 根项仍显示 行/列/类型；子文件夹类型锁定文件夹  
- 页描述：先选一层，再写本层说明和模板  
- `normalizeNode` 保存时带上 `intro_text`

- [ ] **Step 1:** 改类型与 i18n 后 tsc 可能红 → 改编辑器  
- [ ] **Step 2:** 实现树展开状态（可用 `Set<string>` expandedIds，默认展开到选中路径）  
- [ ] **Step 3:** `cd web/admin && pnpm exec tsc --noEmit && pnpm exec vitest run src/features/tg-menu src/lib/api --passWithNoTests`  
- [ ] **Step 4: Commit** `feat(admin-web): tree editor with per-folder intro copy`

---

### Task 6: 文档一句 + 回归

**Files:**
- Modify: `docs/architecture/runtime.md`（进文件夹消息可用 intro_text）
- Optional: `data-model.md` 给 `tg_menu_items` 加 `intro_text` 列说明

- [ ] **Step 1:** 改文档  
- [ ] **Step 2:**

```bash
go test ./internal/tgmenu/... ./internal/httpapi/tgmenu/... ./internal/channel/tg/... ./apps/admin-api/... -count=1
cd web/admin && pnpm exec tsc --noEmit && pnpm exec vitest run src/features/tg-menu --passWithNoTests
```

- [ ] **Step 3: Commit** `docs(architecture): note menu folder intro_text`

---

## Spec coverage

| 设计项 | Task |
|---|---|
| intro_text 字段/校验/迁移空值 | 1–2 |
| Bot 展示 intro + 返回不变 | 3 |
| Case 先于 📁 | 3 |
| API JSON | 4 |
| 左树 + 直白话 + 本层说明 | 5 |
| 架构一句 | 6 |
| 非目标（两列/edit/✍️） | 不做 |
