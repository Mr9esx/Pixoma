# Brainstorm Summary

- Change: dashboard-workbench-refactor
- Date: 2026-08-28

## 确认的技术方案

- 贡献图（kibo-ui contribution-graph）跟随全局时间范围控件（近 7/30/90 天 + 自定义），数据来自 `/stats/tasks/daily` 的 `days[].processed`。
- 欢迎卡采用 `@kokonutui/mouse-effect-card`：保留鼠标点阵动效，但改为全宽工作台横幅；内容为左侧"欢迎回来，{昵称}"（`/setup/me` 的 `nickname || username`）+ 右侧两个快捷入口（新建工作流 / 添加节点）；去掉默认居中促销排版、固定高度、品牌文案与 `bg-white` 光晕，改用语义令牌、无投影 border 分层。
- 页面为双栏：左栏数据大盘，右栏"需要关注的信息"。右栏采用分区列表：节点异常 / 未启用工作流 / 失败任务 Top，每节紧凑列表 + 状态点，条目可点击跳转，空态友好提示。
- 数据大盘左栏区块：全宽贡献图 → 三张概览卡（节点情况、平均负载、负载最高节点 Top5）→ 工作流使用热度 Top + 任务耗时（左右）→ 任务状态分布 + 错误 Top5（左右）。
- 全局时间范围 state 上提到工作台根，区间型区块联动刷新；节点在线/生效等实时信息不受时间范围影响。
- 去掉旧"集群实时负载图"、算力池汇总卡、每节点任务量卡，避免与新概览卡重复。

## 关键取舍与风险

- kokonutui 组件硬编码样式与设计体系冲突 → 落文件后覆盖为语义令牌、无投影、全宽、内容自适应。
- 贡献图 `level` 需手动分档 → 纯函数按 count 生成 0–4 档并配单测。
- 右栏失败任务与左栏错误 Top5 数据源重叠 → 共用 query/queryKey 避免重复请求。
- 窄屏下贡献图可能横向溢出 → `@container` + `overflow-x-auto`。
- 去掉旧集群负载图 → 由"平均负载 + 负载 Top5"覆盖负载维度。

## 测试策略

- 纯函数单测：`dailyToActivity`（count → level 分档）、Top 排序、失败隔离。
- 组件/契约测试覆盖新布局（欢迎卡、双栏、右栏分区、贡献图、概览卡）。
- 时间范围联动、失败隔离（单区块 API 失败不影响其它区块）为验收重点。
- 前端项目 `lint`、`build`、`test` 通过；按 pixoma-design-system 第 11 节 10 条验收。

## Spec Patch

无（open 阶段 delta spec 已覆盖需求，无需回写）。
