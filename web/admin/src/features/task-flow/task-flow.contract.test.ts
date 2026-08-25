import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const read = (p: string) => readFileSync(join(here, p), 'utf8')

const CANVAS = read('task-flow-canvas.tsx')
const CONDITION_FORM = read('condition-form.tsx')
const RULE_EDITOR = read('rule-editor.tsx')
const PROTOTYPE = read('task-flow-prototype.tsx')
const EDITOR = read('task-flow-editor.tsx')
const TYPES = read('types.ts')
const BINDING = read('lib/topic-binding.ts')

describe('task-flow prototype contract', () => {
  it('画布使用 React Flow 且拓扑语义完整（起始/分支/目标/默认回退）', () => {
    expect(CANVAS).toContain('@xyflow/react')
    expect(CANVAS).toContain('case-start')
    expect(CANVAS).toContain('condition-branch')
    expect(CANVAS).toContain('topic-target')
    expect(CANVAS).toContain('default-topic')
    expect(CANVAS).toContain('strokeDasharray')
  })

  it('条件节点常驻展开编辑器、支持自由拖动、画布加分支', () => {
    expect(CANVAS).toContain('add-branch')
    expect(CANVAS).toContain('RuleEditor')
    expect(RULE_EDITOR).toContain('nodrag')
    expect(CANVAS).toContain('onNodeDragStop')
    expect(CANVAS).toContain('draggable: !readOnly')
    expect(CANVAS).toContain('handleNodeDragStop')
  })

  it('ELK 自动布局：多句柄端口 + FIXED_ORDER，边按 sourceHandle/targetHandle 接线', () => {
    expect(CANVAS).toContain('useElkLayout')
    expect(CANVAS).toContain('sourceHandle')
    expect(CANVAS).toContain('targetHandle')
    expect(CANVAS).toContain('useUpdateNodeInternals')
    expect(CANVAS).toContain('handles sources')
    expect(CANVAS).toContain('handles targets')
    const layout = read('lib/elk-layout.ts')
    expect(layout).toContain("from 'elkjs/lib/elk.bundled.js'")
    expect(layout).toContain('elk.direction')
    expect(layout).toContain('org.eclipse.elk.portConstraints')
    expect(layout).toContain('FIXED_ORDER')
    expect(layout).toContain('layerConstraint')
    expect(layout).toContain("'LAST'")
    expect(layout).toContain("side: 'WEST'")
    expect(layout).toContain("side: 'EAST'")
  })

  it('条件表单由属性目录驱动，不硬编码属性 key', () => {
    expect(CONDITION_FORM).toContain('attributes.map')
    expect(CONDITION_FORM).not.toContain("'user.is_premium'")
    expect(CONDITION_FORM).not.toContain('"user.is_premium"')
  })

  it('原型展示真实数据结构字段（rules/when/field/op/value/topic/effective_topics）', () => {
    expect(PROTOTYPE).toContain('JSON.stringify(value, null, 2)')
    expect(PROTOTYPE).toContain('routing/attributes')
    expect(TYPES).toContain('field: string')
    expect(TYPES).toContain('op: ConditionOp')
    expect(TYPES).toContain('effective_topics')
    const MOCK = read('mock-data.ts')
    expect(MOCK).toContain('effective_topics')
  })

  it('整体编辑器：只读/编辑切换与 Case 卡片添加分支入口存在', () => {
    expect(PROTOTYPE).toContain('查看模式')
    expect(PROTOTYPE).toContain('编辑模式')
    expect(CANVAS).toContain('+ 添加分支')
  })

  it('编辑器支持预览模式：只读画布、隐藏顶栏与 Topic 池', () => {
    expect(EDITOR).toContain('preview?: boolean')
    expect(EDITOR).toContain('readOnly || preview')
    expect(EDITOR).toContain('!preview')
    expect(EDITOR).toContain('TaskFlowCanvas')
  })

  it('顶栏 Case 下拉切换流程，右侧 Topic 池可拖入画布（孤立 Topic 保留在画布）', () => {
    // Case 下拉：Select 组件列出所有 Case，切换即加载对应 routing。
    expect(PROTOTYPE).toContain('Select')
    expect(PROTOTYPE).toContain('caseId')
    expect(PROTOTYPE).toContain('loadCase')
    expect(PROTOTYPE).toContain('mockCases')
    // 右侧 Topic 池：抖入可拖拽 Topic 项，样式与数据（mockTopics/binding）驱动；
    // header 与左侧顶栏同高水平对齐（h-14），无多余说明文案。
    expect(PROTOTYPE).toContain('TaskFlowEditor')
    expect(EDITOR).toContain('data-topic-pool-item')
    expect(EDITOR).toContain('application/pixoma-topic')
    expect(EDITOR).toContain('onDragStart')
    expect(EDITOR).toContain('bindingByTopic')
    expect(EDITOR).toContain('flex h-14 items-center border-b border-border px-3')
    expect(PROTOTYPE).not.toContain('拖入画布放置为孤立 Topic')
    const MOCK = read('mock-data.ts')
    expect(MOCK).toContain('mockCases')
    expect(MOCK).toContain('id: 1002')
    expect(MOCK).toContain('id: 1003')
    expect(CANVAS).toContain('caseName')
  })

  it('改连/删线/删规则后，失去最后引用的 Topic 保留为孤立节点（不消失、位置不跳）', () => {
    // 路由变更统一入口 handleRoutingChange：diff 前后引用集合，orphaned Topic 收进孤立集合。
    expect(CANVAS).toContain('handleRoutingChange')
    expect(CANVAS).toContain('orphaned')
    expect(CANVAS).toContain('t !== defaultTopicKey')
    // 孤立前的画布位置沿用（从 getNodes 取当前坐标），节点不跳位。
    expect(CANVAS).toContain('current.find((n) => n.id === `topic-${t}`)')
    // handleConnect/handleEdgesDelete 均走 handleRoutingChange，三个路径全覆盖。
    expect(CANVAS).toContain('handleRoutingChange({ rules: rules.map((r, i) => (i === index ? { ...r, topic } : r)) })')
    expect(CANVAS).toContain('handleRoutingChange({ rules: rules.map((r, i) => (indices.has(i) ? { ...r, topic: undefined } : r)) })')
    // buildNodes 的 onChange 也走 handleRoutingChange（删规则分支同样保留 Topic）。
    expect(CANVAS).toContain('handleAddBranch, handleRoutingChange,')
  })

  it('孤立 Topic 保留在画布：用户临时调整/稍后连线，不随 rules 消失；规则连用/删线自动切换孤立态', () => {
    const GRAPH = read('lib/graph.ts')
    // 图模型支持 isolatedTopics：孤立 Topic 节点加入图，无边。
    expect(GRAPH).toContain('isolatedTopics')
    expect(GRAPH).toContain('孤立 Topic：未被规则引用、但用户拖入画布的 Topic，作为独立节点保留')
    // 画布维护 isolatedTopics 状态；拖入时记录位置；有效孤立集合 = 拖入集合 - 已被规则引用。
    expect(CANVAS).toContain('isolatedTopics')
    expect(CANVAS).toContain('setIsolatedTopics')
    expect(CANVAS).toContain('screenToFlowPosition')
    expect(CANVAS).toContain('application/pixoma-topic')
    expect(CANVAS).toContain('effectiveIsolated')
    expect(CANVAS).toContain('isolatedTopics.filter((key) => !rules.some((r) => r.topic === key))')
    // 孤立 Topic 样式：虚线边框（isolated=true）。
    expect(CANVAS).toContain('isolated')
    expect(CANVAS).toContain('border-dashed')
  })

  it('规则与 Topic 手动连线：onConnect 更新 topic，删除连线取消投递', () => {
    expect(CANVAS).toContain('nodesConnectable')
    expect(CANVAS).toContain('isValidConnection')
    expect(CANVAS).toContain('onConnect={handleConnect}')
    expect(CANVAS).toContain('onEdgesDelete')
    expect(CANVAS).toContain('deletable: false')
    expect(RULE_EDITOR).not.toContain('onTopicChange')
    expect(RULE_EDITOR).toContain('未连接 · 从右侧圆点拖出连线到调度通道')
    expect(RULE_EDITOR).toContain('已连接 →')
    const T = read('types.ts')
    expect(T).toContain('topic?: string')
  })

  it('Topic 关联计算节点：effective_topics 绑定 + 心跳在线双重校验', () => {
    expect(TYPES).toContain('effective_topics')
    expect(TYPES).toContain('edge_online')
    expect(TYPES).toContain('comfy_running')
    expect(BINDING).toContain('effective_topics')
    expect(BINDING).toContain('edge_online')
    expect(BINDING).toContain('onlineEdges.length > 0')
    expect(CANVAS).toContain('boundEdges')
    expect(CANVAS).toContain('onlineEdges')
    expect(CANVAS).toContain('已绑定，无在线 agent')
    expect(CANVAS).toContain('未绑定计算节点')
    expect(PROTOTYPE).toContain('edges={mockEdges}')
    expect(PROTOTYPE).toContain('presence={mockPresence}')
    const MOCK = read('mock-data.ts')
    expect(MOCK).toContain('mockPresence')
    expect(MOCK).toContain("edge_online: false")
  })

  it('计算节点是独立 Node，通过连线与 Topic 建立绑定', () => {
    expect(CANVAS).toContain("'edge-node': EdgeNode")
    expect(CANVAS).toContain('在线 · 可拉取')
    expect(CANVAS).toContain('topicSourceHandle')
    expect(CANVAS).toContain('edgeTargetHandle')
    const GRAPH = read('lib/graph.ts')
    expect(GRAPH).toContain('buildBindingGraph')
    expect(GRAPH).toContain("'edge-node'")
    expect(GRAPH).toContain('e-bind-')
    expect(GRAPH).toContain('isEdgeOnline')
  })

  it('保存前校验：条件/目标 Topic 合法，错误定位到分支卡片并阻止保存', () => {
    const VALIDATE = read('lib/validate.ts')
    expect(VALIDATE).toContain('validateRouting')
    expect(VALIDATE).toContain('topic-missing')
    expect(VALIDATE).toContain('topic-disabled')
    expect(VALIDATE).toContain('condition-unknown-field')
    expect(VALIDATE).toContain('invalidIndexes')
    // 错误定位：画布分支卡片红框 + 错误条；保存按钮受校验门控。
    expect(CANVAS).toContain('validateRouting')
    expect(CANVAS).toContain('issueByIndex')
    expect(CANVAS).toContain('data-rule-issue')
    expect(CANVAS).toContain('border-destructive')
    expect(CANVAS).toContain('修复前不可保存')
    expect(PROTOTYPE).toContain('validateRouting')
    expect(PROTOTYPE).toContain('data-save-button')
    expect(EDITOR).toContain('data-save-status')
    expect(EDITOR).toContain('校验通过，可保存')
    expect(EDITOR).toContain('条规则未通过校验')
  })

  it('节点细边框 + 点击 active 态 border 变 primary', () => {
    // 需求：branch border 变细（border 而非 border-2）；点击后 border 变 primary。
    expect(CANVAS).not.toContain('rounded-lg border-2 bg-background')
    expect(CANVAS).toContain('d.active')
    expect(CANVAS).toContain('border-primary')
    expect(CANVAS).toContain('selectedNodeId')
    expect(CANVAS).toContain('onNodeClick={handleNodeClick}')
    expect(CANVAS).toContain('onPaneClick={handlePaneClick}')
  })

  it('Case→规则连线为固定语义连线：隐藏端口圆点、不可交互；拖动自由摆放，ELK 仅手动触发', () => {
    // 问题 1：Case/规则左侧端口挂载 fixed-handle（isConnectable=false），CSS 隐藏圆点、禁止交互；
    // 仅规则右侧源端口可拖线到 Topic。
    expect(CANVAS).toContain('edgesReconnectable={false}')
    expect(CANVAS).toContain('deletable: false')
    expect(CANVAS).toContain('focusable: false')
    expect(CANVAS).toContain('fixed-handle')
    expect(CANVAS).toContain('isConnectable={false}')
    expect(CANVAS).toContain('不可从这里拉线')
    expect(CANVAS).toContain('不可从这里拖入')
    // 问题 2：默认即平移/选取——panOnDrag 常开，左键空白拖画布、左键节点拖节点；
    // 不再有独立手型按钮/pan-mode 状态。
    expect(CANVAS).toContain('panOnDrag')
    expect(CANVAS).toContain('nodesDraggable={!readOnly}')
    expect(CANVAS).not.toContain('pan-mode')
    expect(CANVAS).not.toContain('panMode')
    expect(CANVAS).not.toContain('平移模式')
    // 可拖动的线（规则→Topic 连线 + 拖线过程线 + 可交互圆点）统一为 lab 主题色。
    expect(CANVAS).toContain('CONNECTABLE_EDGE_COLOR')
    expect(CANVAS).toContain('lab(75.0771% -60.7313 19.4147)')
    expect(CANVAS).toContain('--xy-connectionline-stroke: lab(75.0771% -60.7313 19.4147)')
    expect(CANVAS).toContain('link-handle')
    expect(CANVAS).toContain('.link-handle::after')
    // 画布顶部无说明文案（只保留校验错误条）。
    expect(CANVAS).not.toContain('拖动节点自由摆放；左下「魔棒」')
    // 问题 3：所有节点可自由拖动（draggable: !readOnly）；ELK 只在「自动整理」按钮时重排，
    // 平时拖动绝不触发全图重排（无 skipNextBumpRef / 无自动 bump / 规则数组稳定为 useMemo）。
    expect(CANVAS).toContain('draggable: !readOnly')
    expect(CANVAS).toContain('useMemo(() => routing?.rules ?? [], [routing])')
    expect(CANVAS).toContain('freePos[model.id]')
    expect(CANVAS).toContain('自动整理')
    expect(CANVAS).toContain('Wand2')
    expect(CANVAS).not.toContain('skipNextBumpRef')
    expect(CANVAS).not.toContain('reorderByY')
    expect(CANVAS).not.toContain("'dimensions') bump()")
  })
})
