## Context

参见 `proposal.md`。当前单 Bot：`telegram_bot_token` / `TG_BOT_TOKEN`；主菜单硬编码于 `internal/channel/tg/menu.go`；Case 为全局目录；admin-api 与 `web/admin` 已具备 Case 等资源管理，但无 Menu。本期**不做**多 Bot / `bot_id` 租户。

## Goals / Non-Goals

**Goals:**
- Menu 落库 + 默认种子
- Bot 读配置生成主键盘并路由动作（含 Case 绑定）
- admin-api + 控制台可编辑 Menu

**Non-Goals:**
- 多 Bot / 租户字段改造
- Bot Token 后台管理
- 充值/签到等业务实现（占位即可）
- 前端 mock

## Decisions

1. **单租户 Menu 文档**  
   采用「一份当前 Menu 配置」（单行文档或版本表取最新）而非多 Bot 作用域。理由：明确降难度；后续若做多 Bot 再加 `bot_id`。  
   备选：YAML 文件 — 不利于后台编辑与校验，否决。

2. **动作模型**  
   - `open_case`：必填 `case_id` → 现有 Case 工作流  
   - `list_cases_by_tag`：必填 `tag`（种子「图片」=`image`）  
   - `placeholder`：可选提示文案  
   - `reply_media`：`reply.text` / `reply.images[]`（http(s) URL）至少其一；点击后发文字与图片；本期不做管理端上传  

3. **包边界**  
   - 领域+持久化：新建小模块（如 `internal/tgmenu`）或挂在 catalog 邻域；**禁止** admin-api 依赖 `channel/tg`。  
   - `channel/tg` 只依赖 menu 读取端口。  
   - HTTP：`internal/httpapi/tgmenu` 挂到 admin-api。

4. **刷新策略**  
   Adapter 每次构建主键盘时读仓储（或进程内短缓存 ≤ 几秒）。不要求热推送。

5. **控制台 IA**  
   侧栏增加「TG 菜单」；页用列表+编辑（可 Master–Detail 或单页表单）。Case 选择复用 `listCases`。

## Risks / Trade-offs

- [点击文案匹配脆弱] → 菜单项保留稳定 `id`；路由优先用 id/callback，ReplyKeyboard 文案变更时用「文案→项」映射表并在保存时校验唯一文案。  
- [与现网「图片→tag 列表」差异] → 种子保留 `list_cases_by_tag`；纯 `open_case` 不够表达列表时以该动作为准。  
- [空/坏配置导致 Bot 无键盘] → 读取失败回退默认种子并打日志。  
- [无鉴权写 Menu] → 与现 admin-api 一致，仅内网；文档警示。

## Migration Plan

1. 建表 + 默认种子 upsert（对齐现 `MainMenuRows`）。  
2. Adapter 切读配置；保留短期兼容或直接替换硬编码布局。  
3. admin-api + 控制台上线后用 UI 改绑定验证。  
4. 回滚：关读库开关或回退镜像；表可保留。

## Open Questions

- （已关闭）多 Bot：本期不做  
- （已关闭）Token 后台管理：本期不做  
- `list_cases_by_tag` 是否保留：做，用于兼容「图片」入口
