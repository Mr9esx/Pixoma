---
change: link-health-visibility
role: technical-design
canonical_spec: openspec
---

# 链路健康提示与下一步行动 深度技术设计

## 1. 目标

为 Case / Topic / 计算节点（Edge）三类实体提供统一的依赖健康提示，降低 B 端运营的理解门槛：用户打开详情页就能回答三个问题——**现在能不能用、为什么、下一步做什么、具体怎么修**。

一期落地：**Header 告警**（异常时在页面头部出现 Alert 摘要）+ 下方健康检查明细（断点：原因 + 配置/运行分类 + 行动入口 + 处理指引；引用列表）。**不做 React Flow 全链路图（用户决策，二期评估）。**

## 2. 现状盘点

### 已有（已合入主干的 config-context-association）

| 需求 | 现状 | 位置 |
|---|---|---|
| Case 被哪些入口引用 | ✅ `MenuPlacementsSection` 表格 | `features/cases/sections/menu-placements.tsx` |
| Case 处理流程（routing）编辑 | ✅ `TaskFlowEditor` | `features/task-flow/task-flow-editor.tsx` |
| Case 整链健康结论 | ✅ `ConfigChain`（入口→工作流→投放→执行） | `features/config-context/config-chain.tsx` |
| Topic 被哪些 Case 引用 / 订阅节点就绪 | ✅ 数量 + 前 5 明细 | `features/topics/topic-detail-panel.tsx` |
| 节点订阅 Topic / 在线状态 | ✅ 列表与状态标签 | `features/edges/`（`presence-tags`、`list-health`） |

### 缺口

1. **节点视角**：详情页没有「是否被 Topic 绑定」的提示与「被哪些 Topic 引用」列表。
2. **引用粒度**：Topic/Case 只有数量级摘要，没有完整引用列表。
3. **断点缺行动**：既有 `ConfigChain` 只给结论与明细，没有「配置问题/运行问题」分类，也没有「下一步去哪修」的行动入口。
4. **空态引导**：无入口 / 无引用的实体缺少「现在该做什么」的引导。

## 3. 技术决策

- **不新增后端端点**：沿用 config-context-association「前端组合」模式。`listCases` 已返回完整 `routing`，`listEdges` 已返回 `subscribe_topics`，`listPresence` 已返回在线状态，`getCaseMenuPlacements` 已返回入口挂载。
- **新建 `features/link-health/`** 收敛新逻辑；`ConfigChain` 保留（线性健康结论），`LinkHealthAlert` 负责页面头部告警摘要，`LinkHealthSection` 负责下方健康检查明细（断点行动 + 处理指引 + 引用列表）。
- **断点携带行动与处理指引**：每个 `HealthBreakpoint` 带 `fix: 'config' | 'runtime'`（配置问题 vs 运行问题）、`action: { to, key }`（去向 + i18n 文案 key）与 `guide`（人话处理步骤，i18n key）。用户看到断点即看到「下一步去哪」和「具体怎么处理」。
- **i18n 双语**：新命名空间 `linkHealth`，zh/en 成对，locale 合同测试强制成对。
- **测试**：纯函数走 vitest node 单测；组件/页面走文本合同测试（与 `config-chain.contract.test.ts` 同模式）；新测试文件登记进 `vitest.config.ts` include。
- **一期不引入 `@xyflow/react`**；全链路图在 Phase 2 独立 plan 评估（届时用确定性分层坐标 + `fitView`/`Controls`，不引入 task-flow 编辑器布局 hook）。

## 4. 数据流

```text
页面既有 queries（cases/edges/presence/placements）
  → features/link-health/lib/references.ts（引用 + 健康 + 断点分类/行动）
  → features/link-health/link-health-section.tsx（结论 + 断点行动 + 引用列表）
```

## 5. 组件契约（Phase 1）

```ts
// features/link-health/lib/references.ts
type HealthState = 'ok' | 'warn' | 'bad'
interface ReferenceItem { id: string; name: string; state: HealthState; to: string }
interface HealthBreakpoint {
  stage: 'entry' | 'workflow' | 'topic' | 'node'
  fix: 'config' | 'runtime'          // 配置问题（改配置可修） vs 运行问题（环境/在线）
  key: string                         // i18n 断点文案 key
  params?: Record<string, string>
  action: { to: string; key: string } // 下一步：去向 + 行动文案 key
  guide: string                       // 怎么处理：人话步骤（i18n key）
}
interface EntityHealth { state: HealthState; breakpoints: HealthBreakpoint[] }

function caseRoutingTopics(routing: RoutingConfig | undefined): string[]
function edgeIsReady(edge: EdgeLike, presence: EdgePresence[]): boolean
function topicReferences(key: string, input: HealthInput): { cases: ReferenceItem[]; edges: ReferenceItem[]; health: EntityHealth }
function edgeReferences(id: string, input: HealthInput): { topics: ReferenceItem[]; cases: ReferenceItem[]; health: EntityHealth }
function caseReferences(caseId: number, input: HealthInput): { menuEntries: ReferenceItem[]; topics: ReferenceItem[]; health: EntityHealth }

// features/link-health/link-health-section.tsx
function LinkHealthSection(props: {
  title: string
  health: EntityHealth
  upstream: { title: string; items: ReferenceItem[] }
  downstream: { title: string; items: ReferenceItem[] }
}): JSX.Element

// features/link-health/link-health-alert.tsx（页面 Header 告警，仅异常时渲染）
function LinkHealthAlert(props: {
  name: string
  health: EntityHealth
  anchorTo: string // 锚点，例如 '#link-health-section'
}): JSX.Element | null
```

### 页面结构

```text
详情页 Header（标题 + 状态标签）
  ├─ LinkHealthAlert（仅 health.state !== 'ok' 时出现；含「查看处理指引 ↓」锚点）
详情页正文
  └─ LinkHealthSection（id='link-health-section'：断点明细 + 引用列表）
```

## 6. 断点规则与行动映射

| 视角 | 断点 | fix | 行动（to / key） | 处理指引（guide key） |
|---|---|---|---|---|
| Case | 无菜单入口 | config | `/channels` / actionAddEntry | guideNoMenuEntry |
| Case | Topic 无在线节点 | runtime | `/edges` / actionManageNodes | guideTopicNoReadyNode |
| Topic | 无 Case 引用 | config | `/cases` / actionConfigureRouting | guideNoCaseRoutes |
| Topic | 无节点订阅 | config | `/edges` / actionBindTopic | guideNoEdgeSubscribers |
| Topic | 订阅节点全离线 | runtime | `/edges` / actionManageNodes | guideSubscribersOffline |
| 节点 | 未订阅 Topic | config | `/edges/{id}` / actionEditNode | guideNoTopicBinding |
| 节点 | 无可达 Case | config | `/cases` / actionConfigureRouting | guideNoCaseReachable |
| 节点 | 未就绪 | runtime | `/edges/{id}` / actionCheckNode | guideEdgeNotReady |

## 7. 阶段划分

- **Phase 1（本 plan）**：健康提示 + 断点分类/行动 + 引用列表（三页接入）。
- **Phase 2（独立 plan）**：React Flow 全链路图（暂缓，用户决策）、**内联一键修复**（复用既有编辑表单 mutation：启用节点 / 绑定 Topic / 配置路由）、入口视角、处理流程升级、变更影响分析。
- **Phase 3（独立 plan）**：孤儿实体统计、定时巡检与告警、健康传播汇总、流程版本管理、全局搜索。

## 8. 风险

- **数据量**：全量拉取 `listCases/listEdges`，实体量大时列表页变慢；Phase 2 评估加后端引用端点（`GET /api/v1/topics/{key}/references` 等）作为兜底。
- **「处理流程直接引用节点」假设未确认**：一致性检查（Phase 2）依赖该语义，开工前需产品确认。
- **工作区脏**：存在未提交的 case 删除守卫改动；接入任务需按文件拆分提交，不夹带。
