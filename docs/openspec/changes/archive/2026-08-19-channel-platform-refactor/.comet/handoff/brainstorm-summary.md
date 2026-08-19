# Brainstorm Summary（已确认）

- Change: channel-platform-refactor
- Date: 2026-08-18

## 已确认事实（来自 open 阶段）

- 管理台以「渠道」为一级实体：选平台（TG）+ 填 Bot Token + 启停；无「主键盘」顶级模块
- 菜单收进渠道详情；领域模型平台中立（无 row/col、无 TG 长度约束、无 list_cases_by_tag/tag）
- 差异数据走渠道扩展表 `channel_menu_item_extras`（channel_id + menu_item_id + extra_type + JSON），不做按平台分菜单表
- 不抹平 TG 特色能力：TG 网格布局（row/col）等存 extras，适配器完整保留
- 无存量数据：不迁移旧表/旧字段/旧 API；`users` 无 tg_user_id；`sessions` 用 channel_id + chat_external_id；凭证仅存 channels.credential_ciphertext（AES-GCM，masked 回显）
- ChatID 由 int64 改为渠道命名 string（`tg:<chat_id>`）；notify 按渠道投递
- 运行时端口：EventPort / Messenger / MediaBridge / IdentityResolver；channel/tg 重构为端口实现
- 当前 TG 运行时：long polling（`go b.Start(ctx)`），启动时一次装配，token 为空则不启动

## 待确认（候选）

- ~~渠道启停/凭证更新的生效方式~~ → **已确认：进程内热生效**（装配器监听渠道表，动态启动/停止适配器；凭证更新重建 bot）
- ~~TG 接入模式~~ → **已确认：保持 long polling**（每渠道一个长轮询 goroutine；不引入 webhook）
- ~~extras 在本期管理台的编辑深度~~ → **已确认：管理台完整编辑所有 extras**（接受编辑器复杂度增加；TG 网格布局等差异数据可视化编辑）
- ~~渠道删除/停用语义~~ → **已确认：禁用为主、删除受限**（删除仅允许已禁用且无活跃会话/任务引用；或软删除）

## Spec Patch 候选

- channel-menu-config：新增「渠道差异数据（extras）管理」验收场景（按渠道保存/读取 extras；无效 extras 忽略；TG 网格布局可经管理台编辑）
- channel-management：新增「渠道变更热生效」与「删除受限」验收场景
- channel-runtime-ports：补充「渠道变更驱动适配器启停」场景（热生效）

## 深度技术方案（用户已确认 2026-08-18）

### 运行时装配器（热生效）

- `internal/channel/runtime`：启动时扫描启用渠道并启动各渠道适配器；随后监听渠道表变更（短 TTL 轮询，如 5s），增量处理 enable/disable/凭证变更
- 生命周期：每渠道一个 long polling goroutine；凭证变更 → 停旧 bot、建新 bot；进程退出优雅停止全部
- 事件统一：适配器入站 → 规范化动作（OpenFolder/OpenCase/Back/StartCase/Confirm/Exit/Skip/Continue…）→ `botapp.Facade`

### 端口契约

- `Outbound`：SendText / SendMenu（根层入口）/ SendList（目录与按钮列表）/ SendMedia（图片/文件）
- `MediaBridge`：TG file_id ↔ blob（未来飞书 image_key ↔ blob）
- `IdentityResolver`：(channel_id, external_user_id) → 内部 user（upsert 资料）
- `ChannelAddr`：channel_id + chat_external_id；内存/事件用 `tg:<chat_id>` 字符串，落库拆两列

### 数据与 API

- 表：channels / channel_menus / channel_menu_items(order) / channel_menu_item_cases / channel_menu_item_extras / user_external_identities；sessions 用 channel_id + chat_external_id
- 凭证 AES-GCM（复用 bootstrap enc key），masked 回显
- API：/api/v1/channels CRUD（删除受限）、/api/v1/channels/{id}/menu、placements；移除 /api/v1/tg-menu
- 管理台：渠道列表/新建向导/详情（基本信息 + 菜单配置 + extras 编辑）

### 关键取舍与风险

- 热生效复杂度集中在装配器生命周期 → 用可注入 fake 适配器单测 start/stop/restart
- extras 全量可视化增加编辑器复杂度 → 按渠道平台动态渲染差异字段，中立字段编辑器保持简单
- 删除受限 → 数据完整性与误删保护优先

### 测试策略

- 装配器：渠道变更驱动启停/重建（fake 适配器）
- 适配器：回调翻译、chat 映射、媒体、extras 消费
- 领域/持久化：extras 隔离与校验、删除限制、外部身份唯一
- API：channels/menu/placements 错误语义
- 手工：TG 六键/文件夹/Case/出图；热生效（改 token 不重启）
