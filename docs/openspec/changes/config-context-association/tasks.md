## 1. Case 路由编辑

- [ ] 1.1 Case 编辑/详情页新增「处理流程」节：嵌入 `TaskFlowEditor`，本地维护 routing 草稿
- [ ] 1.2 保存：`PATCH /cases/{id}` 写 routing；未导入 workflow 也可编辑；保存后回读一致
- [ ] 1.3 合同测试：处理流程节、保存载荷、未导入 workflow 可编辑

## 2. 关联上下文面板

- [ ] 2.1 统一 `ContextLinks` 组件（上游/下游引用 + 就绪状态 + 跳转），复用四模块详情页
- [ ] 2.2 Case 详情：路由 Topic + 订阅节点 + 菜单挂载
- [ ] 2.3 Topic 详情：路由 Case（复用 CountCaseRefs）+ 订阅节点 + 就绪状态
- [ ] 2.4 节点详情：订阅 Topic + 关联 Case 摘要
- [ ] 2.5 菜单挂载摘要：MenuCardEditor 挂载 Case 显示 routing 就绪与可执行节点
- [ ] 2.6 合同测试：关联面板结构、就绪徽标、跳转

## 3. 主键盘树管理页

- [ ] 3.1 新增 `/tg-menu` 路由与树形编辑页（文件夹/子项/挂载 Case/placeholder/reply_media）
- [ ] 3.2 保存走菜单 API；失败展示错误；刷新后树保持
- [ ] 3.3 合同测试：树编辑、保存成功/失败

## 4. 验证与文档

- [ ] 4.1 `pnpm tsc -b` + `pnpm vitest run`；`go build ./...` + `go test ./...`
- [ ] 4.2 冒烟：Case 保存 routing → Topic 详情关联 → 菜单挂载摘要一致
- [ ] 4.3 管理配置指引补充关联链路说明
