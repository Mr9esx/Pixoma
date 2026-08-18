# Build Review Notes — channel-platform-refactor

执行方式：executing-plans（用户 2026-08-19 明确「不用 subagent，当前会话执行」，覆盖原 subagent-driven 配置）
审查范围：`bbe08c6`..HEAD（226 文件 / +14378 / -4891）

## CRITICAL（已修复）

1. **菜单项 ID 跨渠道主键冲突**：`channel_menu_items.ID` 为全局主键，而种子/编辑器使用固定 ID（`btn-image` 等）。第二个渠道初始化即冲突，多渠道无法共存。已改为 `(channel_id, id)` 复合主键；`channel_menu_item_cases` 同步加 `channel_id`；placements 按 `channel+id` 解析；新增 `TestReplaceTree_SameSeedAcrossChannels` 回归。

## IMPORTANT（已修复）

2. **中立领域残留 TG 长度上限**：计划要求「中立核心无 TG 长度约束」，但领域仍保留 `MaxIntroTextLen=3500`。已删除领域限制，改由 TG 适配器 messenger 按平台上限截断（文本 4096 / caption 1024 字符）。
3. **putExtras 归一化失效**：`for _, extra := range extras` 值拷贝导致 `MenuItemID` 写入丢失。改为索引循环，并以 map key 为准；补测试。
4. **停用/删除渠道后 notify 残留**：`tgBotWrapper.Stop` 未注销 registry，通知会继续路由到已停止的 bot。已加 `unset`。
5. **装配器 Stop 失败仍重建**：`restartLocked` 在 Stop 出错时继续 Create，可能双 bot 并存。改为记录 error 并等待退避重试。

## ACCEPTED（记录接受原因）

- **admin-api 无鉴权**：全量管理 API（含既有 cases/users/tasks）一致行为，入口已输出「do not expose to the public internet」警告；非本 change 回归，接入网关/鉴权列为后续独立事项。
- **装配器持有锁期间执行 Start/Stop**：会短暂阻塞其他渠道 reconcile；当前渠道数量级可接受，后续可改为每渠道独立 goroutine。
- **`notified` 去重 map 无上限**：随任务数增长；TG bot 长生命周期下可接受，后续可加 LRU 上限。
- **`SendMenu` fallback 种子用 "default" 渠道**：仅在菜单读取失败时兜底，不影响正常路径。
- **TG 回调 64 字节限制**：`cpf:<folder>:<case>` 在超长 ID 下可能超限，属 TG 平台固有限制，适配层是正确位置；长 ID 场景需后续数据映射方案。
