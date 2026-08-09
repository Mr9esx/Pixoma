## Context

参见 `proposal.md`。首期已实现扁平 Menu JSON；产品确认升级为**树形主键盘**（文件夹下钻 Case）+ **菜单项↔Case 关联表**（Case 详情可反查），并为多 Bot **预留 `bot_id`**（本期仍单 Bot 运行）。

## Goals / Non-Goals

**Goals:**
- 关系表：`tg_menus` / `tg_menu_items`（`parent_id`）/ `tg_menu_item_cases`
- Bot：根 ReplyKeyboard；folder → Inline 子层 + Case + 返回
- admin：主键盘树编辑；Case 详情展示挂载路径
- 从旧 `tg_menu_configs` 迁移/种子

**Non-Goals:**
- 多 Bot 切换 UI、Token 后台
- 用 Case.categories 当文件夹
- 管理端图片上传
- 充值等真业务、前端 mock

## Decisions

1. **关系表树（方案 A）** — 否决 JSON 双写与 categories 当文件夹。  
2. **`bot_id` 预留、运行单 Bot** — 菜单 `bot_id=default`。  
3. **Case 反查在详情页** — `GET .../cases/{id}/menu-placements`。  
4. **「图片」种子为 folder** — 挂载 image Case；`list_cases_by_tag` 仅兼容。  
5. **包边界不变** — admin-api 不依赖 `channel/tg`。  
6. **继续本 change** — 扁平实现升级，不另开 change。

## Risks / Trade-offs

- [迁移] 旧 JSON → 新表需一次性脚本/启动迁移。  
- [callback 64 字节] 短前缀 + item id。  
- [深度] 校验限制最大深度。  
- [无鉴权] 与现 admin-api 一致，仅内网。

## Migration Plan

1. AutoMigrate 新表；若新表空：从种子或旧 `tg_menu_configs` 导入。  
2. Bot/admin 切读新模型。  
3. 停止写旧表。  
4. 回滚：回退应用版本；新表可留。

## Open Questions

- （已关闭）文件夹来源、多 Bot 深度、Case 反查位置、是否新开 change
