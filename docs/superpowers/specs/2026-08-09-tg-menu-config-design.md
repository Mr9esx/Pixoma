---
comet_change: tg-menu-config
role: technical-design
canonical_spec: openspec
---

# TG Menu 可配置化 — 技术设计

## 1. 背景与目标

OpenSpec change `tg-menu-config`。现状主菜单硬编码于 `internal/channel/tg/menu.go`；管理面无入口。本期在**单 Bot、无多租户**前提下：Menu 落库、admin 可配、Bot 按配置渲染键盘并执行动作（含绑定 Case、按 tag 列 Case、占位提示、**回复文字+图片 URL**）。

非目标：多 Bot / `bot_id`、Bot Token CRUD、管理端图片上传/对象存储、充值等真业务、前端 mock。

## 2. 数据模型

### 2.1 表 `tg_menu_configs`

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | string PK | 固定 `"default"`（预留多文档，本期只用一行） |
| `items_json` | text/JSON | `MenuItem[]` |
| `updated_at` | time | |

### 2.2 `MenuItem` JSON

```ts
type MenuAction =
  | 'open_case'
  | 'list_cases_by_tag'
  | 'placeholder'
  | 'reply_media'

type MenuItem = {
  id: string              // 稳定 id，如 btn-image
  label: string           // ReplyKeyboard 文案；同文档内唯一
  row: number             // 0-based 行
  col: number             // 0-based 列（同行排序）
  enabled: boolean
  action: MenuAction
  case_id?: string        // open_case 必填，且须存在于 catalog
  tag?: string            // list_cases_by_tag 必填
  placeholder_text?: string
  reply?: {
    text?: string
    images?: string[]     // http(s) URL；本期不存本地上传
  }
}
```

### 2.3 校验规则

- `label` trim 后非空且文档内唯一（enabled 项之间必须唯一；建议全部项唯一以免改启用踩坑）
- `open_case`：`case_id` 必填 + Case 仓储存在
- `list_cases_by_tag`：`tag` 必填
- `placeholder`：不要求 case/tag；`placeholder_text` 可选（空则用默认「暂未开放」类文案）
- `reply_media`：`reply.text` 与 `reply.images` **至少其一非空**；`images` 若有则每项须为绝对 `http`/`https` URL
- 保存为整份替换（PUT）；拒绝时不写库

### 2.4 默认种子

对齐现网 `MainMenuRows`：

| id | label | 动作 |
|----|-------|------|
| btn-image | 🖼 图片 | `list_cases_by_tag` tag=`image` |
| btn-video | 🎬 视频 | `placeholder` |
| btn-recharge | 💰 充值积分 | `placeholder` |
| btn-checkin | 📅 签到 | `placeholder` |
| btn-profile | 👤 个人中心 | `placeholder` |
| btn-help | 🆘 帮助 | `placeholder` |

空表或读失败：运行时回退内存种子（与上表一致），并打 error/warn 日志。

## 3. 包与依赖

```
internal/tgmenu/
  domain/          // MenuDocument, MenuItem, Validate, DefaultSeed
  application/     // Get, Replace
  infrastructure/persistence/  // GORM

internal/httpapi/tgmenu/   // GET/PUT → application
internal/channel/tg/       // 依赖 tgmenu 只读端口（GetMenu），禁止反向依赖
apps/admin-api             // 挂载路由；不 import channel/tg
apps/bot                   // 注入同一 DB 上的 Menu 仓储
web/admin                  // /tg-menu 页 + 侧栏 + i18n
```

依赖方向：`httpapi` / `channel/tg` → `tgmenu`；`tgmenu` → `catalog`（仅校验 case 存在，用窄接口）。

## 4. API

| 方法 | 路径 | 行为 |
|------|------|------|
| GET | `/api/v1/tg-menu` | 返回 `{ id, items, updated_at }`；无行则先种子再返回或直接返回种子表示（实现选「懒种子写入」或「读时合成」；推荐 **首次 GET/Bot 读时若空则 upsert 种子**） |
| PUT | `/api/v1/tg-menu` | body `{ items: MenuItem[] }`；校验通过后整份替换 |

错误：400 校验；404 不用于整份文档；500 存储失败。无鉴权（与现 admin-api 一致）。

## 5. Bot 运行时

1. **构建键盘**：`GetMenu` → 过滤 `enabled` → 按 `(row,col)` 生成 `ReplyKeyboardMarkup`（`IsPersistent: true` 保持现语义）。
2. **点击匹配**：用户文本 == 某项 `label` → 取该项 `action` 分发：
   - `list_cases_by_tag` → 现有 `showImageCases` 类逻辑泛化为按 tag 列表
   - `open_case` → 现有 Case 预览/开始路径（与 inline 入口对齐）
   - `placeholder` → 回复 `placeholder_text` 或默认文案
   - `reply_media` → 若有 text 先发文本；再对每个 image URL `SendPhoto`（或等价）；单张失败记日志继续其余；全部媒体失败且无 text 则回一句错误提示
3. **刷新**：每次构建键盘读仓储；可选进程内 TTL≤5s 缓存。不要求热推送。
4. **删除硬编码布局唯一性**：`menu.go` 常量可保留作种子文案来源或测试夹具，运行时不得只靠常量拼键盘。

## 6. 管理控制台

- 侧栏：「TG 菜单」（中英 i18n），建议放在 Case 附近
- 页面：单页编辑整份 `items`（表格：label/row/col/enabled/action + 条件字段）
- `reply_media`：textarea + 图片 URL 多行输入（增删行）
- `open_case`：Case 下拉（`listCases`）
- 保存 → PUT；错误展示 ErrorBanner；成功 toast
- 无前端 mock

## 7. 测试策略

| 层 | 内容 |
|----|------|
| domain | Validate 各动作；种子结构；label 冲突 |
| persistence | upsert 种子；Replace 后 Get 一致 |
| httpapi | GET/PUT 200；非法 case_id / 空 reply_media → 400 |
| channel/tg | 表驱动：给定 Menu → 键盘行；点击 label → 期望调用（list/open/placeholder/reply_media）；图片 URL 失败不崩 |
| 手工 | 改文案与 reply_media 后 `/start` 与点击符合预期 |

## 8. 迁移与回滚

1. AutoMigrate `tg_menu_configs`；首次读空写入种子  
2. Bot/admin 同时切读库  
3. 回滚应用版本即可；表可留  

## 9. 架构文档

实现后更新 `docs/architecture/data-model.md`（新表）；必要时 runtime 一句「主菜单来自 tg_menu_configs」。

## 10. Spec 对齐

Canonical：`docs/openspec/changes/tg-menu-config/specs/**`（含 reply_media Spec Patch）。
