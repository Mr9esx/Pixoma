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

  it('条件节点常驻展开编辑器、支持拖动排序、画布加分支', () => {
    expect(CANVAS).toContain('add-branch')
    expect(CANVAS).toContain('RuleEditor')
    expect(RULE_EDITOR).toContain('nodrag')
    expect(CANVAS).toContain('onNodeDragStop')
    expect(CANVAS).toContain('reorderByY')
    expect(CANVAS).toContain('draggable')
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

  it('Case 列表与画布融合为整体编辑器，点击切换流程（不再拖入）', () => {
    expect(PROTOTYPE).toContain('Case 列表')
    expect(PROTOTYPE).toContain('data-case-item')
    expect(PROTOTYPE).toContain('onClick={() => loadCase(c.id)}')
    expect(PROTOTYPE).toContain('点击切换 Case 流程')
    expect(PROTOTYPE).not.toContain('draggable')
    expect(PROTOTYPE).not.toContain('dataTransfer.setData')
    expect(PROTOTYPE).not.toContain('onDragOver')
    const MOCK = read('mock-data.ts')
    expect(MOCK).toContain('mockCases')
    expect(MOCK).toContain('id: 1002')
    expect(MOCK).toContain('id: 1003')
    expect(CANVAS).toContain('caseName')
  })

  it('规则与 Topic 手动连线：onConnect 更新 topic，删除连线取消投递', () => {
    expect(CANVAS).toContain('nodesConnectable')
    expect(CANVAS).toContain('isValidConnection')
    expect(CANVAS).toContain('onConnect={handleConnect}')
    expect(CANVAS).toContain('onEdgesDelete')
    expect(CANVAS).toContain('deletable: false')
    expect(RULE_EDITOR).not.toContain('onTopicChange')
    expect(RULE_EDITOR).toContain('未连接 · 从右侧圆点拖出连线到 Topic')
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
})
