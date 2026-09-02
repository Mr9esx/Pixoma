## 1. 领域树与校验

- [x] 1.1 用失败测试钉死 `MenuTree`：根按钮 ≤6、列数 1–6、打开卡片必须内嵌 card、卡片链深度 ≤8、非法动作/空文案/非法 URL/不存在的 workflow 拒绝
- [x] 1.2 实现嵌套树 domain（替代现行平铺 `Menu`+`Card` 引用），校验与默认树（打开工作流 + 帮助发文字）让测试通过
- [x] 1.3 用测试描述 `Compile(tree)`：根 → 键盘折行；`open_card` → 按节点 id 取卡片消息；其它动作保持现有语义
- [x] 1.4 实现编译器；Case placements 改为沿树收集路径，更新相关测试

## 2. 持久化与 HTTP

- [x] 2.1 用测试固定 GET/PUT `/api/v1/channels/{id}/menu` 读写整棵树；旧格式 JSON 视为无配置并返回默认树
- [x] 2.2 实现单文档存储；保存走 domain 校验；不迁移 `channel_cards`
- [x] 2.3 用测试固定原 `/cards` 与卡片引用接口不再作为管理契约（404/410 且不写库）
- [x] 2.4 实现并删除 admin/bot/livedemo 对卡片仓库的写路径；更新渠道删除、Case 删除清理
- [x] 2.5 更新 `docs/architecture/data-model.md`（及仍描述分表菜单的 `bounded-contexts.md` / `runtime.md` 段落）

## 3. Bot 运行时

- [x] 3.1 用测试固定：点键盘打开卡片发送子树卡片；卡片按钮再打开更深层卡片
- [x] 3.2 切换 TG 菜单运行时到编译结果（callback 用树节点 id）
- [x] 3.3 更新 livedemo seed 为新默认/示例树

## 4. Puck 编辑器

- [x] 4.1 添加 `@measured/puck`；用测试钉死 `puckData ↔ MenuTree` 往返（含嵌套卡片）
- [x] 4.2 实现映射与 Puck config（键盘、菜单按钮、卡片、卡片按钮；动作字段按类型切换）
- [x] 4.3 用契约/组件测试钉死：手机画布、调色板无表单字段、打开卡片挂子树、非法动作不能保存、未保存关闭有确认
- [x] 4.4 用 Puck 做整页编辑器（渠道详情「编辑菜单」进入；顶栏渠道名 + 返回）；overrides 走 shadcn + 设计令牌；保存 PUT 整棵树
- [x] 4.5 详情菜单区改为同一套组件的只读手机；删除能力地图与弹层编辑、卡片库 UI，以及 `listCards`/`createCard` 等调用

## 5. 其它写入方

- [x] 5.1 用测试固定 `addWorkflowMenuEntry` 向根键盘追加 `open_workflow`，满 6 个失败
- [x] 5.2 改 quick-config 写新树，不再构造旧 `Menu`/`Card`

## 6. 收口

- [x] 6.1 跑 menucard / channel tg / admin menu / quick-config 相关测试并修回归
- [ ] 6.2 确认 admin 菜单编辑主路径在浏览器里可拖、可保存、可重开（无浏览器工具时用契约测试 + 说明未跑到的部分）
