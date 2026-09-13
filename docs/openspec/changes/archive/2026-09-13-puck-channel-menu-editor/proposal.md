## Why

现有渠道菜单编辑器是「能力地图 + 弹层改 Menu/Card」：运营看不到 Telegram 里真实的键盘和卡片长什么样，卡片还要单独成库、用 id 引用。点名的 shadcn-builder 是网页表单生成器（导出 React），和「键盘 / 卡片 / 动作」不是一类东西。改用可嵌入的 Puck 可视化编辑器，画布做成手机，组件就是我们的菜单积木；文档改成一棵嵌套树，保存后编译成 bot 能执行的键盘和卡片。旧数据和 API 允许整表作废。

## What Changes

- **编辑器**：渠道菜单编辑入口改为嵌入 Puck（`@measured/puck`）。左调色板、中**手机画布**、右属性。运营在手机里拖键盘格子、卡片消息、卡片按钮；画布即用户将看到的形态，不再做独立卡片库，也不再 vendor [shadcn-builder](https://github.com/iduspara/shadcn-builder)。
- **文档模型**：每个渠道一份嵌套树。根是主键盘（列数 + 按钮）；按钮的动作可以是现有能力（开工作流 / 发文字媒体 / 链接 / 复制 / 列任务），或「打开卡片」——卡片作为该按钮的子节点长在树上，卡片按钮同样可再嵌卡片。卡片不复用、没有孤儿卡片清单。
- **编译**：保存的真相源是这棵树（Puck JSON + 我们的组件类型）。运行时编译成 Telegram 主键盘与消息卡片（含 inline 按钮），bot 不直接解释 Puck。
- **BREAKING**：废除现行 `Menu` + `Card` 分表/分 API（含 `/cards` CRUD、卡片引用查询）。不迁移已有配置；渠道无文档时用新默认树。
- **写入方**：quick-config 等往菜单里塞「打开工作流」的入口，改为往新树根键盘追加按钮。
- **不拆 change**：Puck 组件配置就是 schema；没有编辑器的运行时、没有运行时的编辑器都不能单独交付。既有 `channel-menu-editor-refactor` 不在本 change 内归档，产品意图由本 change 取代。

### 非目标

- 不 vendor、不嵌入 shadcn-builder，不导出 React 代码。
- 不在 Telegram 里做表单逐步问答（Input/Select 填参）。
- 不做卡片库、卡片跨按钮复用。
- 不做旧 Menu/Card 兼容读、不做一次性搬家。
- 不把 GrapesJS / Craft.js / 纯 React Flow 当编辑内核。

## Capabilities

### New Capabilities

- `puck-channel-menu-editor`：管理端用 Puck 在手机画布上编辑渠道菜单树（组件调色板、拖拽、属性、保存校验）的行为规范。

### Modified Capabilities

- `channel-menu-config`：渠道菜单从「平台中立能力树 / 现行 Menu+Card 引用」改为「每渠道一棵嵌套树」；卡片是树节点而不是可复用资源。
- `tg-menu`：持久化改为单一菜单文档；无配置时提供新默认树；Case 反查改为沿树路径。
- `tg-menu-admin-api`：管理 API 改为整份文档 GET/PUT；**BREAKING** 去掉独立 cards 集合接口。
- `channel-menu-interaction`：用户路径改为「主键盘 → 树上嵌套的卡片消息」；取消「分组最多一层 / 必须走独立卡片库」对编辑结构的约束（主键盘仍由管理员显式配置）。

## Impact

- 前端 `web/admin`：`features/menu/*` 用 Puck 替换能力地图弹层编辑；新增 Puck config（键盘、菜单按钮、卡片、卡片按钮）与手机画布 root；`package.json` 增加 `@measured/puck`；quick-config 写菜单适配新树。
- 后端 `internal/menucard`（及 HTTP）：存储与校验改为树文档；删除或停用 cards CRUD；编译器产出 bot 使用的键盘/卡片 IR；默认菜单重写。
- Bot / channel TG 适配器：消费编译结果，不解析 Puck。
- 数据：旧菜单/卡片配置作废；无迁移。
- 规格：上列 1 个新 capability + 4 个 delta；`channel-menu-editor-refactor` 仍为独立未归档 change。
- 架构文档：触及菜单持久化真相源与 admin API，需同步 `docs/architecture/data-model.md`（及若拓扑描述过期则 `runtime.md` / `bounded-contexts.md`）。
