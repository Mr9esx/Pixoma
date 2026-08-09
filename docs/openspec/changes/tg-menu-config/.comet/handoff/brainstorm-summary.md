# Brainstorm Summary

- Change: tg-menu-config
- Date: 2026-08-09
- Status: **已确认**

## 确认的技术方案

- 持久化方案 **A**：`tg_menu_configs` 单行 `id=default`，`items` JSON
- 动作：`open_case` / `list_cases_by_tag` / `placeholder` / **`reply_media`**（文字 + 图片 URL）
- API：`GET/PUT /api/v1/tg-menu`；包 `internal/tgmenu` + `httpapi/tgmenu`；admin-api 不依赖 channel/tg
- Bot：每次构建键盘读库（可短缓存）；文案唯一；失败回退种子
- 后台：TG 菜单页整份编辑；reply_media 配 text + images URL 列表
- 图片：**URL 粘贴**，本期不做上传/对象存储
- 单 Bot；不做 Token 后台、多租户

## 关键取舍与风险

- 坏图链：日志 + 尽量仍发文字
- ReplyKeyboard 文案匹配依赖唯一 label
- 无鉴权写接口（与现 admin 一致）

## 测试策略

- 领域校验（含 reply_media 至少 text 或 images 其一）
- HTTP GET/PUT；TG messenger mock 断言文本/图片投递
- 控制台联调

## Spec Patch

- `tg-menu`、`channel-tg`、`tg-menu-admin-api`、`admin-resource-pages` 补充 reply_media 场景
