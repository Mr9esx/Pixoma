## 1. Menu 领域与持久化

- [ ] 1.1 新增 Menu 领域模型（项 id、文案、顺序、启用、动作类型、case_id/tag、reply_media 的 reply 等）与仓储接口
- [ ] 1.2 实现 SQLite/GORM 持久化 + 迁移；空库默认种子对齐现网主菜单（图片=list_cases_by_tag:image，其余 placeholder）
- [ ] 1.3 校验：open_case 需存在 case_id；reply_media 至少 text 或 images；images 须 http(s)；文案唯一；非法配置拒绝写入
- [ ] 1.4 领域/持久化单测（种子、更新、校验失败、reply_media）

## 2. Bot 运行时接入

- [ ] 2.1 `channel/tg` 构建主 ReplyKeyboard 改为读取 Menu 仓储（失败回退种子）；去掉对硬编码布局的唯一依赖
- [ ] 2.2 点击路由：open_case → 既有 Case 工作流；list_cases_by_tag → 既有列表；placeholder → 提示文案；reply_media → 发文本与按 URL 发图（单张失败不崩）
- [ ] 2.3 组合根注入 Menu 仓储；补充/调整 TG 相关测试

## 3. admin-api

- [ ] 3.1 实现 `httpapi` Menu handler：GET/PUT（或等价）`/api/v1/tg-menu`
- [ ] 3.2 挂载到 admin-api；保持不依赖 `channel/tg`
- [ ] 3.3 handler 单测（合法更新、非法 case_id、读回一致）

## 4. 管理控制台

- [ ] 4.1 侧栏增加「TG 菜单」入口 + 中英 i18n
- [ ] 4.2 Menu 管理页：列表/编辑/排序/动作与 Case 选择、tag、placeholder、reply_media（文本+图片 URL）；调用 admin-api
- [ ] 4.3 空态/错误/保存反馈与现有资源页一致；README 联调补充 Menu

## 5. 文档与验收

- [ ] 5.1 同步 `docs/architecture/data-model.md`（及必要时 runtime/overview）中 Menu 表与主链路说明
- [ ] 5.2 联调验收：改文案/绑 Case 后 TG `/start` 键盘与入口行为符合预期；Network 仅打 admin-api
