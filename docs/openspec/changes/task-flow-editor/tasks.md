## 1. 依赖与入口

- [ ] 1.1 `web/admin` 引入 `@xyflow/react` 并按需导入，`pnpm` 构建通过、体积可接受
- [ ] 1.2 侧栏新增 Topic 菜单项（Dashboard、实例、Case、渠道、Task、User、Session、Topic 顺序）与路由
- [ ] 1.3 新增页面/区块的中英 i18n 文案（画布、条件表单、Topic 管理、节点订阅）成对补齐

## 2. 条件表单渲染器

- [ ] 2.1 条件目录数据层：TanStack Query 拉取 `/api/v1/routing/attributes` 并按属性 key 索引
- [ ] 2.2 schema → 控件映射：enum→Select、boolean→Switch、number→NumberInput、string→TextInput，未知类型回退文本输入
- [ ] 2.3 `and`/`or` 嵌套组合编辑与回读，保存输出协议 JSON
- [ ] 2.4 单测/契约测试：枚举渲染、未注册属性不可选、新增属性自动出现、组合规则回读等价

## 3. 任务分流画布

- [ ] 3.1 图模型与自动布局：Case 起始节点 + 条件分支 + Topic 目标节点 + 默认 Topic 虚线回退（只读）
- [ ] 3.2 编辑交互：新增/删除条件分支、选择目标 Topic、调整分支顺序（显式 order 数组为真相源）
- [ ] 3.3 序列化：画布 → `routing.rules`（有序）；回读：Case `routing` → 画布；空配置显示默认回退
- [x] 3.4 保存前校验：条件完整合法、目标 Topic 存在且启用；错误定位到对应分支
- [ ] 3.5 只读/编辑模式切换与契约测试（画布区存在、非法配置阻止保存）

## 4. 页面集成

- [ ] 4.1 Case 创建/编辑表单新增「任务分流」区块，`buildPayload` 统一携带 `routing`；详情页只读画布展示
- [ ] 4.2 计算节点详情页新增订阅 Topic 编辑（多选，空 = 默认 Topic），保存走节点更新 API
- [ ] 4.3 Topic 管理页：列表、新建、编辑、启用/禁用（数据来自 `/api/v1/topics`），`default` 无删除路径

## 5. 质量与联调

- [ ] 5.1 全量前端测试（vitest）与构建通过，既有 Case 工作流编辑器（导入/输入输出绑定）不被破坏
- [ ] 5.2 与 `topic-routing` 联调冒烟：创建 Topic → 画布配置规则 → 保存 Case → 节点订阅 → 只读回读
- [ ] 5.3 文档同步（管理后台入口说明、与 topic-routing 的契约依赖）
