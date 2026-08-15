## Why

Telegram 主菜单（ReplyKeyboard）原先硬编码；首期已落地扁平可配置 Menu。产品确认需要**竞品式文件夹导航**（根键盘 → 📁 下钻 → Case），并用 **DB 关系**维护菜单↔Case，以便 Case 详情反查、并为多 Bot 预留结构。本期仍**单 Bot 运行**，不做多 Bot 切换与 Token 后台。

## What Changes

- 将扁平 `tg_menu_configs` 升级为关系表树：`tg_menus` / `tg_menu_items`（`parent_id`）/ `tg_menu_item_cases`
- Bot：根 ReplyKeyboard；folder → Inline 子层 + 挂载 Case + 返回；其余动作语义保留
- admin-api：树形 GET/PUT Menu；Case 菜单挂载反查 API
- `web/admin`：「主键盘」树编辑；Case 详情只读展示挂载路径
- **非目标**：多 Bot 切换 UI、Token CRUD、用 Case.categories 当文件夹、管理端图片上传、充值等真业务、前端 mock

## Capabilities

### New Capabilities
- `tg-menu`: 树形 Menu 领域/持久化、默认种子、Case 反查
- `tg-menu-admin-api`: 管理端树形 Menu HTTP API 与 placements

### Modified Capabilities
- `channel-tg`: 主键盘与文件夹下钻来自 Menu 树
- `admin-web-shell`: 侧栏「主键盘」
- `admin-resource-pages`: 主键盘树编辑 + Case 详情挂载展示

## Impact

- 代码：`internal/tgmenu`、`channel/tg`、`httpapi`、`web/admin`、admin-api 挂载
- 数据：新三表；旧 JSON 表迁移后停写
- 配置：Token 仍 yaml/env
- 文档：architecture data-model / runtime / bounded-contexts
