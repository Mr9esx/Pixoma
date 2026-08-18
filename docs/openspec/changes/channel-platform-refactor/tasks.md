## 1. 渠道领域与持久化

- [x] 1.1 新增 `internal/channel` 领域：Channel 实体（id/platform/name/credential/enabled/时间戳）与 Repository 端口
- [x] 1.2 GORM 实现 `channels` 表与 CRUD（含凭证加密/解密与 masked 回显）
- [ ] 1.3 凭证密钥注入：AES-GCM 密钥经 env 注入（开发可退化并告警）；无 env / `platform_settings` token 迁移
- [x] 1.4 渠道应用服务：Create/List/Get/Update/Delete，校验平台合法、凭证非空、删除/停用语义

## 2. 菜单模型中立化（tgmenu → menu）

- [x] 2.1 重命名 `internal/tgmenu` → `internal/menu` 并更新全部 import（domain/application/infrastructure/httpapi）
- [x] 2.2 领域模型去 TG 字段：`row/col` → 有序列表 `order`；`intro_text` 上限移出领域校验；`bot_id` → `channel_id`；移除 `list_cases_by_tag` kind 与 `tag` 字段
- [x] 2.3 校验规则更新：同层 label 唯一、深度上限、kind 关联约束保留；去除 row/col 与 tag 相关检查
- [x] 2.4 新建 `channel_menus` / `channel_menu_items` / `channel_menu_item_cases` / `channel_menu_item_extras`；删除旧 `tg_menus*` 与 legacy JSON 表模型（无存量数据，不做迁移）
- [x] 2.5 删除 `MigrateFromLegacyIfNeeded` 等旧数据迁移逻辑（无存量兼容需求）
- [x] 2.6 默认种子按渠道生成：图片文件夹入口 + 挂载图片 Case；空渠道可读

## 3. 会话寻址渠道化

- [x] 3.1 `sharedkernel.ChatID` 由 int64 改为渠道命名 string（`tg:` 前缀）并新增 ChannelAddr 辅助类型
- [ ] 3.2 `sessions` 新表直接使用 `channel_id` + `chat_external_id`（无存量迁移）；task/notify 事件载荷同步
- [ ] 3.3 notify 事件载荷 ChatID 字段切换并保持 JSON 兼容
- [ ] 3.4 TG 适配器入口/出口完成 chat_id ↔ `tg:xxx` 映射

## 4. 身份渠道化

- [ ] 4.1 新增 `user_external_identities`：`channel + external_id` 唯一约束；`users` 新模型不含 `tg_user_id` 列
- [ ] 4.2 upsert 按（channel, external_id）执行并刷新资料字段与 `last_seen_at`
- [ ] 4.3 用户查询/列表接口支持按渠道外部身份过滤（兼容旧 tg_user_id 查询）

## 5. 渠道端口契约与 TG 适配器重构

- [ ] 5.1 `internal/channel` 端口定义：EventPort / Messenger / MediaBridge / IdentityResolver
- [ ] 5.2 规范化事件与动作：文本/媒体/回调动作（OpenFolder/OpenCase/Back/StartCase/Confirm 等）
- [ ] 5.3 `channel/tg` 实现端口：回调前缀协议解析 → 规范化动作（64 字节编码留在 tg 内部）
- [ ] 5.4 TG 渲染下沉：ReplyKeyboard 优先读 extras 网格布局（缺省按 order 两列）、Inline、媒体上传与安全文件名、reply_media 图片；不抹平 TG 特色能力
- [ ] 5.5 适配器装配：从 Channel 凭证启动 bot（启用/禁用生命周期）

## 6. 运行时装配与通知投递

- [ ] 6.1 启动装配器：扫描启用渠道 → 启动对应适配器（独立 goroutine + graceful stop）
- [ ] 6.2 notifybridge 泛化到 channel 层：按渠道地址路由投递
- [ ] 6.3 通知幂等去重保留并按渠道隔离

## 7. 管理 API

- [ ] 7.1 `/api/v1/channels` CRUD（GET 列表 / POST / GET:id / PUT / DELETE）
- [ ] 7.2 `/api/v1/channels/{id}/menu` GET/PUT（渠道作用域校验，不存在渠道 404）
- [ ] 7.3 `/api/v1/cases/{id}/menu-placements` 返回含渠道标识的路径
- [ ] 7.4 移除旧 `/api/v1/tg-menu` 路由与 `internal/httpapi/tgmenu` 引用（无兼容需求）
- [ ] 7.5 admin-api host 挂载新渠道路由

## 8. 管理台改版

- [ ] 8.1 侧栏：「主键盘」→「渠道」；顺序为 Dashboard、实例、Case、渠道、Task、User、Session
- [ ] 8.2 渠道列表页 + 新建向导（选平台 + 名称 + Token）
- [ ] 8.3 渠道详情页：基本信息（masked token、启用/停用）tab
- [ ] 8.4 渠道菜单配置页：迁移原主键盘编辑器（排序控件；TG 渠道可在 extras 中配置根层网格布局）
- [ ] 8.5 i18n 中英文案：新增渠道相关文案，移除「主键盘」入口文案

## 9. 测试与回归

- [ ] 9.1 领域测试：order 排序、渠道作用域、去 row/col 后校验规则
- [ ] 9.2 持久化测试：渠道 CRUD、菜单迁移幂等、外部身份唯一
- [ ] 9.3 API 测试：channels CRUD、channel menu、placements 错误语义
- [ ] 9.4 适配器测试：回调动作翻译、chat_id 映射、通知路由与幂等
- [ ] 9.5 全量 `go test ./...` 通过 + TG 手工回归（六键主键盘、文件夹下钻、Case 流程、出图通知）
