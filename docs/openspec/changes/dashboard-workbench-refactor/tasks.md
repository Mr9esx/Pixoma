## 1. 组件与基础设施

- [ ] 1.1 将 `@kokonutui/mouse-effect-card` 落地到 `web/admin/src/components/kokonutui/mouse-effect-card.tsx`，改为使用语义令牌、全宽、内容自适应，去掉默认品牌文案/CTA 与固定高度，保留鼠标点阵动效。
- [ ] 1.2 将 `kibo-ui/contribution-graph` 落地到 `web/admin/src/components/kibo-ui/contribution-graph/index.tsx`，确认 `date-fns` 依赖可用。
- [ ] 1.3 为 contribution-graph 编写 `dailyToActivity`（`listTaskDailyStats` days → `{date,count,level}`，count 四分位分档 0–4）纯函数及单测。
- [ ] 1.4 新增工作台 i18n key（zh/en）：欢迎卡标题、快捷入口、数据大盘卡片标题、占位卡文案、贡献图全年说明、任务量单位。

## 2. 欢迎卡与根布局

- [ ] 2.1 重构 `web/admin/src/routes/_app/index.tsx` / `dashboard-page.tsx`：移除顶部 `title`/`desc` 区块，改为主数据区 + 右侧欢迎卡（无顶部全宽欢迎卡）。
- [ ] 2.2 欢迎卡接入 `/setup/me`（`fetchCurrentUser`），展示 `nickname || username`（左对齐、垂直居中，无多余副标题）；提供「新建工作流」「添加/管理节点」两个快捷入口并正确跳转。
- [ ] 2.3 将时间范围 state（复用 `TaskRangePicker`）上提到工作台根，控件渲染在数据大盘底部，仅驱动区间图表。

## 3. 数据大盘左栏

- [ ] 3.1 新增 `workbench-contribution`：全宽 contribution-graph 展示本年度（1/1-今天）每日任务量，不受全局时间范围影响。
- [ ] 3.2 新增节点概览卡（总数/启用数，来自 `listEdges`）。
- [ ] 3.3 新增平均负载卡（`listFleetStats` 的 avg_cpu/mem/gpu）。
- [ ] 3.4 新增负载最高节点 Top5 卡（`listFleetStats.nodes` 按 cpu 降序取 top5）。
- [ ] 3.5 新增「工作流使用热度 Top + 任务耗时」左右布局（summary 样式：标题 + 关键指标数字 + 图表；任务耗时用渐变面积图）。
- [ ] 3.6 新增「任务状态分布 + 错误 Top5」左右布局（`daily.summary` / `listTaskErrorStats` limit 5）。
- [ ] 3.7 每个区块保持独立 `useQuery` 与 `LoadingSkeleton`/`ErrorBanner` 失败隔离。

## 4. 右栏欢迎卡

- [ ] 4.1 右栏渲染欢迎卡（代替占位卡）：欢迎语 + 两个快捷入口，左对齐、垂直居中，无多余描述文案。
- [ ] 4.2 移除原 `workbench-attention` 组件及占位卡逻辑，避免残留引用；不再接入关注信息数据。

## 5. 旧区块清理与样式

- [ ] 5.1 移除旧 `realtime-status-section` 的「集群实时负载图」bar chart、算力池汇总卡、每节点任务量卡；保留节点/在线相关逻辑并迁移到新概览卡。
- [ ] 5.2 按 `pixoma-design-system` 核对：只使用语义令牌与 `color-mix`、表面无投影、accent 每屏至多两次、图表用 chart-1…5、卡片 border 分层。
- [ ] 5.3 确保文案走 i18n（默认中文），无禁用词/感叹号/emoji。

## 6. 测试与验收

- [ ] 6.1 调整/新增 `dashboard` 相关组件与契约测试，覆盖新布局、贡献图映射、失败隔离、时间范围联动。
- [ ] 6.2 运行该前端项目的 `lint`、`build`、`test`，确保通过。
- [ ] 6.3 按 `pixoma-design-system` DESIGN.md 第 11 节 10 条验收清单过查工作台页。
