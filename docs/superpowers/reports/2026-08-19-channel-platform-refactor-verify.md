# 验证报告：channel-platform-refactor

日期：2026-08-19
验证模式：full（40 任务 / 8 delta specs / 213 变更文件，均超轻量阈值）
语言：zh-CN

## 1. 结论

**PASS** — 全部 7 项检查通过，无未决 CRITICAL/IMPORTANT 问题；验证过程中修复的发现已提交（`e139ee6`），非关键接受项记录于本报告与 `.comet/review-notes.md`。

## 2. 检查项

### 2.1 任务完成度 — PASS

- tasks.md：40/40 全部 `[x]`
- Superpowers plan：14/14 任务全部 `[x]`

### 2.2 实现符合 change design.md 高层决策 — PASS

| design.md 决策 | 实现位置 |
|---|---|
| 消息平台实体（平台/凭证/启停） | `internal/channel/domain` + `application` + `infrastructure/persistence` |
| 凭证 AES-GCM + masked 回显 | `internal/platform/crypto` + `domain/credential.go` |
| 菜单按消息平台作用域（order 替代 row/col） | `internal/menu`（`channel_menu_*` 表，`(channel_id, id)` 复合主键） |
| 会话地址消息平台化 | `sharedkernel.ChatID` + `ChannelAddr`；`sessions.channel_id + chat_external_id` |
| 端口契约 + TG 端口实现 | `internal/channel/ports` + `internal/channel/tg` |
| 热生效装配器 | `internal/channel/runtime`（5s watch、start/stop/restart、退避重试） |
| 通知按消息平台投递 + 幂等 | `runtime.NotifyRouter` + `tg.Adapter.notified` |
| 管理 API / 管理台 | `internal/httpapi/channels` + `channelmenu`；`web/admin/src/features/channels` |

### 2.3 实现符合 Design Doc（深度技术设计）— PASS（含 2 处记录偏差）

Design Doc 10 个章节的核心决策均已落地；偏差记录见 §5。

### 2.4 能力规格场景 — PASS

8 个 delta spec 的 Requirements/Scenarios 逐项核对：

| Spec | 核对结果 |
|---|---|
| channel-management（5 需求 11 场景） | 通过：创建/列表/详情/CRUD/非法平台/装配/热生效/受限删除；装配器测试覆盖 start/stop/restart/delete/失败退避/cancel |
| channel-menu-config（5 需求 12 场景） | 通过：消息平台隔离（含跨消息平台同 ID 回归测试）、中立模型无 TG 字段、extras 按消息平台存取与无效忽略、Case 挂载/反查、默认种子 |
| channel-runtime-ports（4 需求 7 场景） | 通过：端口契约、应用层无 SDK 依赖、身份消息平台化、通知按消息平台+幂等 |
| channel-tg（3 需求 6 场景） | 通过：Case 列表 Inline 按钮、完成发图、安全文件名（`photoUploadName`）、主键盘来自菜单配置、凭证装配/禁用停止 |
| tg-menu（3 需求 6 场景） | 通过：树持久化、挂载校验、默认种子、reply_media 校验（http(s) URL / 空回复拒绝） |
| tg-menu-admin-api（3 需求 6 场景） | 通过：GET/PUT 菜单、非法配置 400、placements 空/非空、仅经 admin-api |
| user-directory（2 需求 4 场景） | 通过：`user_external_identities` 唯一约束、upsert 建档与资料刷新 |
| admin-web-shell（3 需求 5 场景） | 通过：侧栏消息平台替代主键盘、顺序正确、中英文案 |

### 2.5 proposal 目标已满足 — PASS

消息平台一级实体、菜单收进消息平台详情、中立领域 + extras、运行时端口化、身份消息平台化、旧路径/旧表/旧字段物理删除，均已实现；无存量数据迁移要求满足。

### 2.6 delta spec 与 design doc 无矛盾 — PASS

handoff hash 变化（61bd8b → 23e0a9）来源于 brainstorming 阶段回写的 spec patch（extras 管理、热生效、删除受限场景）与 tasks 勾选；Design Doc 第 3/4/6/7/8 节已覆盖这些内容，无矛盾。

### 2.7 设计文档可定位 — PASS

`docs/superpowers/specs/2026-08-18-channel-platform-refactor-design.md` 存在且与当前 change 绑定（`.comet.yaml design_doc`）。

## 3. 验证证据

| 命令 | 结果 |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | 无告警 |
| `go test ./...` | 全部通过（无失败输出） |
| `cd web/admin && pnpm test` | 30 files / 115 tests 全过 |
| `cd web/admin && pnpm build` | tsc + vite build 成功 |
| `rg "tg_menu\|tg-menu\|tgMenu\|TgUserID\|telegram_token_cipher\|list_cases_by_tag" internal apps web/admin/src` | 无命中（含"主键盘"文案已清除） |

已记录：`comet state record-check … verify --exit-code 0`。

## 4. 审查发现处理

Build 阶段最终代码审查（executing-plans + review_mode=standard；按用户要求在当前会话执行，未派发 reviewer subagent）发现并修复：

- CRITICAL：菜单项 ID 全局主键冲突导致多消息平台无法共存 → 复合主键 `(channel_id, id)` + 链接表消息平台化 + 回归测试（`TestReplaceTree_SameSeedAcrossChannels`）
- IMPORTANT：中立领域残留 TG 3500 上限 → 移入 TG messenger 截断（4096/1024 字符）
- IMPORTANT：putExtras 值拷贝导致 menu_item_id 归一失效 → 索引循环 + 测试
- IMPORTANT：消息平台停用后 notify handler 未注销 → `registry.unset`
- IMPORTANT：装配器 Stop 失败仍重建可能双 bot → 记录 error 等待退避

## 5. 接受的非关键偏差（记录原因与影响范围）

1. **InboundEvent.Raw 字段未实现**（Design Doc 2.1）：TG 适配器以类型化入口（HandleText/HandleCallback/HandleUserMedia）替代 InboundEvent 统一流水线，功能等价；`ports.InboundEvent` 契约保留供后续平台使用。影响：无行为差异；多平台接入时可按需补充统一事件封装。
2. **Outbound 新增 SendMediaURL**（Design Doc 2.3 未列）：reply_media 图片 URL 直发需要，适配器内部实现，不影响契约消费方。
3. **admin-api 无鉴权**：与既有全部管理 API 一致（仅内网约定），非本 change 回归；入口已输出公开暴露警告。
4. **装配器持锁执行 Start/Stop**：会短暂阻塞同轮其他消息平台 reconcile，当前规模可接受。
5. **`notified` 去重 map 无上限**：随任务数增长，TG bot 生命周期内可接受，后续可加 LRU。
6. **SendMenu fallback 种子使用 "default" 消息平台**：仅菜单读取失败时兜底，不影响正常路径。
7. **TG 回调 64 字节限制**：`cpf:<folder>:<case>` 在超长 ID 下可能超限，属 TG 平台固有限制，适配层是正确位置；长 ID 场景需后续数据映射方案。

## 6. 工作区处理

验证输入为已提交区间 `bbe08c6..HEAD`；工作区中其余未提交改动属于其他 active change（edge-system-monitoring、admin-shadcn-base-migration、instance-presence-tags 等），按既有约定保留不处理，不影响本 change 验证。

## 追加：上线前回归发现的缺陷（2026-08-19 复核）

### 缺陷
`make dev` 在既有开发库（旧 `sessions` 表含 `chat_id` 与数据）上启动失败：
`appboot: db: automigrate: SQL logic error: Cannot add a NOT NULL column with default value NULL`（`ALTER TABLE sessions ADD channel_id text NOT NULL`）。

### 根因与修复
- GORM AutoMigrate 向已有行表添加 NOT NULL 新列无默认值，SQLite 拒绝
- 附带修正 `(channel_id, chat_external_id, status)` 唯一索引设计缺陷：同一会话可有多条 `submitted` 历史记录，唯一索引会在历史数据或后续写入时冲突，应仅为普通复合索引（单活跃会话由应用层保证）

修复提交 `11e8460`：`channel_id`/`chat_external_id` 加 `default:''`；`idx_chat_active` 改为普通复合索引。

### 复核证据
- 新增 `TestAutoMigrate_UpgradesLegacySessionsTable`：旧 schema（chat_id + 两条 submitted 行）AutoMigrate 成功
- 开发库副本验证：ALTER + 复合索引 + 存量行读取成功
- `go test ./...` 全绿（58 包）
