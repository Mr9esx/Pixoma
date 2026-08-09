---
comet_change: tg-menu-config
role: technical-design
canonical_spec: openspec
---

# TG Menu 可配置化 — 技术设计（树形主键盘 + Case 关联）

## 1. 背景与目标

OpenSpec change `tg-menu-config`。首期已落地扁平 `tg_menu_configs`（JSON items）+ Bot/admin 读写。产品确认升级为：

- 主键盘可配 **树**：底部 ReplyKeyboard → 📁 文件夹（Inline）→ 再列 Case；可返回上一级
- DB **关系表**维护菜单项 ↔ Case；**Case 详情**可看到挂在哪个菜单路径
- **单 Bot 运行**；表预留 `bot_id`（本期恒 `default`），不做多 Bot 切换 UI / Token 后台
- 管理台侧栏文案为「主键盘」

非目标：多 Bot 切换、Token CRUD、用 Case.categories 当文件夹来源、管理端图片上传、充值等真业务、前端 mock。

## 2. 数据模型

### 2.1 表

#### `tg_menus`

| 列 | 说明 |
|----|------|
| `id` | PK，本期固定 `"default"` |
| `bot_id` | 预留；本期恒 `"default"`，唯一约束 `(bot_id)` 或与 id 对齐 |
| `updated_at` | UTC |

#### `tg_menu_items`

| 列 | 说明 |
|----|------|
| `id` | PK（稳定字符串，如 `btn-image`、`folder-undress`） |
| `menu_id` | → `tg_menus.id` |
| `parent_id` | 可空；空 = 根层（ReplyKeyboard）；非空 = 某文件夹下的子项 |
| `label` | 展示文案 |
| `row` / `col` | 同层内排布（0-based） |
| `enabled` | 是否启用 |
| `kind` | `folder` \| `open_case` \| `placeholder` \| `reply_media` \| `list_cases_by_tag`（兼容） |
| `placeholder_text` | 可选 |
| `reply_json` | `reply_media` 的 text/images |
| `tag` | `list_cases_by_tag` 兼容字段 |
| `sort` / 或仅靠 row,col | 同层排序 |

#### `tg_menu_item_cases`

| 列 | 说明 |
|----|------|
| `menu_item_id` | → `tg_menu_items.id` |
| `case_id` | → catalog case id（逻辑 FK） |
| `sort` | 同项下 Case 按钮顺序 |

唯一约束：`(menu_item_id, case_id)`。

- `kind=open_case`：关联表恰好 1 条 Case（或等价强制）
- `kind=folder`：关联表 0..N 条（进入文件夹时与子 `menu_items` 一并展示）
- 其它 kind：关联表应为空

### 2.2 废弃 / 迁移

| 旧 | 新 |
|----|-----|
| `tg_menu_configs.items_json` | 一次性迁移为 `tg_menus` + `tg_menu_items` + `tg_menu_item_cases`；迁移后停止写入旧表；可读兼容可选，推荐启动时若新表空且旧表有数据则迁移 |

### 2.3 校验

- 同 `menu_id` 下 `id` 唯一；同层（同 `parent_id`）`label` trim 后唯一；至少 1 个启用的根项
- `folder`：允许子项与/或关联 Case；深度建议上限（如 5）防止滥用
- `open_case`：关联恰好一存在的 Case
- `folder` 挂载的每个 `case_id` 必须存在
- `reply_media`：text 与 images 至少其一；images 均为 http(s)
- `list_cases_by_tag`：`tag` 必填（兼容种子；新配置优先用 folder+挂载）
- 写失败不落库（事务：items + links 同事务替换或按菜单文档版本替换）

### 2.4 默认种子（根层）

对齐现网六键；「图片」改为 **`folder`**（可先无子文件夹，挂载所有 `tag=image` 的 Case，或空挂载 + 兼容读 tag——实现选：**种子将现有 image Case 写入关联表**，运行时 folder 只读关联+子项，不再依赖 tag 列表作为主路径）。

其余根项：`placeholder`（视频/充值/签到/个人中心/帮助）。

## 3. Bot 运行时

1. **ReplyKeyboard**：`parent_id IS NULL` 且 enabled → 按 row/col 构建（失败回退内存种子树）
2. **点根 folder**：发 Inline 消息：
   - 子 `folder` 按钮文案建议前缀 `📁 `
   - 本项 `tg_menu_item_cases` 列出的 Case（可用 `name · ¥price`）
   - `⬅️ 返回` → 回主菜单或上一层（根 folder 的返回 = 主菜单文案）
3. **点子 folder**：换一层 Inline（callback 带 `menu_item_id`，注意 ≤64 字节：如 `mf:<id>` / `mc:<case_id>` / `mb:<parent_id>`）
4. **点 Case**：现有预览/开跑
5. **open_case / placeholder / reply_media**：与现语义一致
6. **list_cases_by_tag**：保留实现兼容旧配置；新种子不依赖它作为「图片」主路径

刷新：每次构建键盘/进文件夹读库；可选 TTL≤5s。

## 4. API

| 方法 | 路径 | 行为 |
|------|------|------|
| GET | `/api/v1/tg-menu` | 返回树形 DTO：`{ id, bot_id, items: TreeNode[] }`（嵌套 children + `case_ids`/`cases` 摘要） |
| PUT | `/api/v1/tg-menu` | 整棵树替换（事务）；校验后写 menus/items/links |
| GET | `/api/v1/cases/{id}/menu-placements` | Case 反查：`[{ menu_id, path: [{id,label}], item_id }]` |

（也可挂在 Case GET 的扩展字段；推荐独立子资源便于权限与缓存。）

禁止 `httpapi` / admin-api import `channel/tg`。

## 5. 管理控制台

- 侧栏「主键盘」；页：左树/列表 + 右编辑（kind、挂 Case 多选、子项）
- Case 详情：只读「出现在主键盘」路径列表（调 placements API）
- 无 mock；图片 URL only

## 6. 包与依赖

```
internal/tgmenu/
  domain/          // Menu, Item, Kind, Validate, DefaultSeedTree, Placements
  application/     // GetTree, ReplaceTree, ListPlacementsByCase
  infrastructure/persistence/

channel/tg → 只读端口（GetRoot / GetChildren / …）
httpapi/tgmenu + cases 扩展 placements
```

## 7. 测试

- domain：树校验、深度、关联数量、placements 聚合
- persistence：事务替换、反查
- httpapi：PUT 树、placements 200
- channel/tg：进 folder → Inline 含 📁 与 Case；BACK；callback 不崩
- 手工：主键盘改树后 TG 分层浏览；Case 详情可见挂载

## 8. 架构文档

更新 `data-model.md`：三表 + ER；`runtime.md`：主菜单树浏览；`bounded-contexts`：tgmenu 职责含树与反查。

## 9. Spec 对齐

Canonical：`docs/openspec/changes/tg-menu-config/specs/**`（本修订同步 delta）。

## 10. 决策记录（产品）

| 决策 | 选择 |
|------|------|
| 文件夹来源 | 主键盘自配树，不用 Case.categories |
| 多 Bot | 预留 `bot_id`，本期单 Bot |
| Case→菜单可见性 | Case 详情「出现在主键盘」 |
| 落地方式 | 继续本 change，扁平模型升级为树 |
| 实现方案 | 关系表树（非 JSON 双写） |
