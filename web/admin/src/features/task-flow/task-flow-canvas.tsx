import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Background,
  Handle,
  Position,
  ReactFlow,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  useReactFlow,
  useUpdateNodeInternals,
  type Connection,
  type Edge,
  type Node,
  type NodeChange,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { CircleAlert, Maximize, Wand2, ZoomIn, ZoomOut } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AttributeDescriptor, EdgePresence, EdgeRecord, RoutingConfig, RoutingRule, TopicRecord } from './types'
import { addRule, buildBindingGraph, moveRule, nodePosition, removeRule, type FlowGraph } from './lib/graph'
import {
  branchSourceHandle,
  branchTargetHandle,
  CASE_NODE_ID,
  CASE_SOURCE_HANDLE,
  DEFAULT_NODE_ID,
  DEFAULT_SOURCE_HANDLE,
  DEFAULT_TARGET_HANDLE,
  edgeTargetHandle,
  computeFitViewport,
  topicSourceHandle,
  topicTargetHandle,
} from './lib/elk-layout'
import { useElkLayout } from './lib/use-elk-layout'
import { validateRouting, type RuleIssue } from './lib/validate'
import { RuleEditor } from './rule-editor'
import { topicBindings, type TopicBinding, type TopicStatus } from './lib/topic-binding'

type CanvasProps = {
  routing: RoutingConfig | undefined
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  edges: EdgeRecord[]
  presence: EdgePresence[]
  defaultTopicKey: string
  readOnly: boolean
  caseName: string
  onChange: (next: RoutingConfig) => void
}

const topicName = (topics: TopicRecord[], key: string) => topics.find((t) => t.key === key)?.name ?? key

type CaseData = { label: string; sourceHandles: string[]; ruleCount: number; readOnly: boolean; active?: boolean; onAddBranch: () => void }
type TopicData = {
  label: string
  targetHandles: string[]
  sourceHandles: string[]
  binding: TopicBinding
  /** 未被任何规则引用的孤立 Topic（用户从右侧池拖入，待连线）。 */
  isolated?: boolean
  /** 点击后的 active 态（border 变 primary）。 */
  active?: boolean
}
type EdgeData = { label: string; targetHandles: string[]; online: boolean; active?: boolean }
type BranchData = {
  rule: RoutingRule
  index: number
  total: number
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  readOnly: boolean
  targetHandles: string[]
  sourceHandles: string[]
  /** 保存前校验问题（有错时卡片红框 + 顶部错误条）。 */
  issue?: RuleIssue
  /** 点击后的 active 态（border 变 primary）。 */
  active?: boolean
  onConditionChange: (when: RoutingRule['when']) => void
  onMove: (dir: -1 | 1) => void
  onRemove: () => void
}

/** Case 起始节点：右侧按规则顺序渲染 N+1 个源端口（规则 + 默认回退），卡片内提供「+ 添加分支」。 */
const CaseNode = memo(function CaseNode({ id, data }: NodeProps) {
  const d = data as CaseData
  const updateNodeInternals = useUpdateNodeInternals()
  useEffect(() => {
    updateNodeInternals(id)
  }, [id, updateNodeInternals, d.sourceHandles.length])
  return (
    <div
      className={cn(
        'relative w-44 rounded-lg border bg-background px-3 py-3 text-sm font-medium shadow-none',
        d.active ? 'border-primary' : 'border-border',
      )}
    >
      <div className='handles sources'>
        {d.sourceHandles.map((h) => (
          // Case → 规则是固定语义连线：端口仅作连线锚点，隐藏圆点、不可交互（不可从这里拉线）。
          <Handle key={h} id={h} type='source' position={Position.Right} isConnectable={false} className='fixed-handle' />
        ))}
      </div>
      <div className='text-center'>
        {d.label}
        <div className='mt-0.5 text-[10px] font-normal text-muted-foreground'>
          {d.ruleCount} 条分流 · 默认回退
        </div>
      </div>
      <button
        type='button'
        data-add-branch
        disabled={d.readOnly}
        onClick={d.onAddBranch}
        className='mt-2 w-full rounded-md border border-dashed border-border bg-background px-2 py-1.5 text-xs font-normal text-muted-foreground transition-colors hover:border-primary/60 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50'
      >
        + 添加分支
      </button>
    </div>
  )
})

/** 条件节点：始终展开的编辑器卡片，左侧入端口、右侧出端口，标题栏可拖动排序。 */
const BranchNode = memo(function BranchNode({ data }: NodeProps) {
  const d = data as BranchData
  return (
    <div
      data-branch-node
      className={cn(
        'relative w-[560px] rounded-lg border bg-background shadow-none',
        d.issue ? 'border-destructive/70' : d.active ? 'border-primary' : 'border-border',
      )}
    >
      {d.issue && (
        <div
          data-rule-issue
          className='nodrag flex items-center gap-1.5 rounded-t-md border-b border-destructive/40 bg-destructive/10 px-4 py-1.5 text-xs font-medium text-destructive'
        >
          <CircleAlert className='size-3.5 shrink-0' />
          <span className='truncate'>{d.issue.message}</span>
        </div>
      )}
      <div className='handles targets'>
        {d.targetHandles.map((h) => (
          // 入端口仅承接 Case 固定连线：隐藏圆点、不可交互（不可从这里拖入）。
          <Handle key={h} id={h} type='target' position={Position.Left} isConnectable={false} className='fixed-handle' />
        ))}
      </div>
      <div className='handles sources'>
        {d.sourceHandles.map((h) => (
          // 出端口是唯一可交互源：从规则右侧圆点拖线到 Topic；link-handle 圆点用连线同色。
          <Handle key={h} id={h} type='source' position={Position.Right} className='link-handle' />
        ))}
      </div>
      <RuleEditor
        rule={d.rule}
        index={d.index}
        total={d.total}
        topics={d.topics}
        attributes={d.attributes}
        onConditionChange={d.onConditionChange}
        onMove={d.onMove}
        onRemove={d.onRemove}
        dragHandle={!d.readOnly}
      />
    </div>
  )
})

const STATUS_DOT: Record<TopicStatus, string> = {
  ready: 'bg-emerald-500',
  'bound-offline': 'bg-amber-500',
  unbound: 'bg-rose-400',
}

const STATUS_TEXT: Record<TopicStatus, string> = {
  ready: '可消费',
  'bound-offline': '已绑定，无在线 agent',
  unbound: '未绑定计算节点',
}

function TopicCard({ d, dashed }: { d: TopicData; dashed?: boolean }) {
  return (
    <div
      className={cn(
        'relative w-52 rounded-lg border bg-background px-3 py-2.5 shadow-none',
        dashed || d.isolated ? 'border-dashed' : 'border-solid',
        d.active ? 'border-primary' : dashed || d.isolated ? 'border-border bg-muted/20' : 'border-border',
      )}
    >
      <div className='handles targets'>
        {d.targetHandles.map((h) => (
          // 可交互入端口（接收规则连线）：link-handle 圆点用连线同色。
          <Handle key={h} id={h} type='target' position={Position.Left} className='link-handle' />
        ))}
      </div>
      <div className='handles sources'>
        {d.sourceHandles.map((h) => (
          <Handle key={h} id={h} type='source' position={Position.Right} />
        ))}
      </div>
      <div className='flex items-center gap-1.5'>
        <span className={cn('size-2 shrink-0 rounded-full', STATUS_DOT[d.binding.status])} />
        <span className='truncate text-sm font-medium'>{d.label}</span>
      </div>
      <div className='mt-1 text-[10px] text-muted-foreground'>
        {d.binding.status === 'ready'
          ? `${d.binding.onlineEdges.length} 台在线可消费`
          : STATUS_TEXT[d.binding.status]}
      </div>
    </div>
  )
}

const TopicNode = memo(function TopicNode({ data }: NodeProps) {
  const d = data as TopicData
  return <TopicCard d={d} />
})

const DefaultTopicNode = memo(function DefaultTopicNode({ data }: NodeProps) {
  const d = data as TopicData
  return <TopicCard d={d} dashed />
})

/** 计算节点（edge）：独立节点，通过绑定边与 Topic 相连。 */
const EdgeNode = memo(function EdgeNode({ data }: NodeProps) {
  const d = data as EdgeData
  return (
    <div
      className={cn(
        'relative w-40 rounded-lg border bg-background px-3 py-2.5 text-sm shadow-none',
        d.active ? 'border-primary' : d.online ? 'border-border' : 'border-dashed border-border bg-muted/30',
      )}
    >
      <div className='handles targets'>
        {d.targetHandles.map((h) => (
          <Handle key={h} id={h} type='target' position={Position.Left} />
        ))}
      </div>
      <div className='flex items-center gap-1.5'>
        <span className={cn('size-2 shrink-0 rounded-full', d.online ? 'bg-emerald-500' : 'bg-muted-foreground/50')} />
        <span className='truncate text-sm font-medium'>{d.label}</span>
      </div>
      <div className='mt-0.5 text-[10px] text-muted-foreground'>{d.online ? '在线 · 可拉取' : '离线'}</div>
    </div>
  )
})

const nodeTypes = {
  'case-start': CaseNode,
  'condition-branch': BranchNode,
  'topic-target': TopicNode,
  'default-topic': DefaultTopicNode,
  'edge-node': EdgeNode,
}

function buildNodes(
  graph: FlowGraph,
  rules: RoutingRule[],
  topics: TopicRecord[],
  attributes: AttributeDescriptor[],
  bindingByTopic: Map<string, TopicBinding>,
  readOnly: boolean,
  caseName: string,
  onAddBranch: () => void,
  onChange: (next: RoutingConfig) => void,
  freePos: Record<string, { x: number; y: number }>,
  issueByIndex: Map<number, RuleIssue>,
  /** 当前点击选中的节点 id（active 态 border 变 primary）。 */
  selectedNodeId: string | null,
): Node[] {
  const branchCount = graph.nodes.filter((n) => n.kind === 'condition-branch').length
  return graph.nodes.map((model) => {
    const index = model.ruleIndex ?? 0
    const slotPos = nodePosition(model, branchCount)
    const active = model.id === selectedNodeId
    const base = {
      id: model.id,
      type: model.kind,
      position: freePos[model.id] ?? slotPos,
      // 自由摆放：所有节点都可拖动（只读模式除外）；ELK 仅在用户点击「自动整理」时重排。
      draggable: !readOnly,
      selectable: !readOnly,
    }
    if (model.kind === 'condition-branch') {
      const rule = rules[index]
      return {
        ...base,
        data: {
          rule,
          index,
          total: rules.length,
          topics,
          attributes,
          readOnly,
          targetHandles: [branchTargetHandle(index)],
          sourceHandles: [branchSourceHandle(index)],
          issue: issueByIndex.get(index),
          active,
          onConditionChange: (when: RoutingRule['when']) =>
            onChange({ rules: rules.map((r, i) => (i === index ? { ...r, when } : r)) }),
          onMove: (dir: -1 | 1) => onChange(moveRule({ rules }, index, dir)),
          onRemove: () => onChange(removeRule({ rules }, index)),
        } satisfies BranchData,
      }
    }
    if (model.kind === 'case-start') {
      return {
        ...base,
        // React Flow v12 会给不可拖/不可选的节点内联 pointer-events:none，这里放开让卡片内按钮可点。
        style: { pointerEvents: 'all' },
        data: {
          label: caseName,
          sourceHandles: [CASE_SOURCE_HANDLE],
          ruleCount: rules.length,
          readOnly,
          active,
          onAddBranch,
        } satisfies CaseData,
      }
    }
    if (model.kind === 'topic-target') {
      const topicKey = model.topic ?? model.label
      // 未被任何规则引用的孤立 Topic（用户从右侧池拖入，待连线）。
      const isolated = !rules.some((r) => r.topic === topicKey)
      return {
        ...base,
        data: {
          label: topicName(topics, topicKey),
          targetHandles: [topicTargetHandle(topicKey)],
          sourceHandles: [topicSourceHandle(topicKey)],
          binding: bindingByTopic.get(topicKey) ?? { topic: topicKey, status: 'unbound', boundEdges: [], onlineEdges: [] },
          isolated,
          active,
        } satisfies TopicData,
      }
    }
    if (model.kind === 'default-topic') {
      const defaultKey = model.topic ?? model.label
      return {
        ...base,
        data: {
          label: topicName(topics, defaultKey),
          targetHandles: [DEFAULT_TARGET_HANDLE],
          sourceHandles: [DEFAULT_SOURCE_HANDLE],
          binding: bindingByTopic.get(defaultKey) ?? { topic: defaultKey, status: 'unbound', boundEdges: [], onlineEdges: [] },
          active,
        } satisfies TopicData,
      }
    }
    // edge-node
    const edgeId = model.edgeId ?? model.id.slice('edge-'.length)
    return {
      ...base,
      data: {
        label: model.label,
        targetHandles: [edgeTargetHandle(edgeId)],
        online: Boolean(model.online),
        active,
      } satisfies EdgeData,
    }
  })
}

/** 可拖动连线（规则→Topic）与拖线过程线的统一颜色（用户指定 lab 色值）。 */
const CONNECTABLE_EDGE_COLOR = 'lab(75.0771% -60.7313 19.4147)'

function buildEdges(graph: FlowGraph): Edge[] {
  return graph.edges.map((e) => {
    const dashedStyle = e.dashed ? { strokeDasharray: '6 4', stroke: 'var(--muted-foreground)' } : undefined
    if (e.source === CASE_NODE_ID) {
      // Case → 规则/默认回退：语义固定连线，不可拖动、不可删除（仅规则→Topic 才可编辑）。
      const isDefault = e.target === DEFAULT_NODE_ID
      const ruleIndex = isDefault ? NaN : Number(e.target.slice('branch-'.length))
      return {
        id: e.id,
        source: e.source,
        sourceHandle: CASE_SOURCE_HANDLE,
        target: e.target,
        targetHandle: isDefault ? DEFAULT_TARGET_HANDLE : branchTargetHandle(ruleIndex),
        style: dashedStyle,
        deletable: false,
        focusable: false,
      }
    }
    if (e.target.startsWith('edge-')) {
      // Topic → 计算节点（绑定关系）
      return {
        id: e.id,
        source: e.source,
        sourceHandle: e.source === DEFAULT_NODE_ID ? DEFAULT_SOURCE_HANDLE : topicSourceHandle(e.source.slice('topic-'.length)),
        target: e.target,
        targetHandle: edgeTargetHandle(e.target.slice('edge-'.length)),
        deletable: false, // 绑定关系由后台 effective_topics 决定，画布不可删
        focusable: false,
        style: dashedStyle,
      }
    }
    const ruleIndex = Number(e.source.slice('branch-'.length))
    const isDefaultTarget = e.target === DEFAULT_NODE_ID
    return {
      id: e.id,
      source: e.source,
      sourceHandle: branchSourceHandle(ruleIndex),
      target: e.target,
      targetHandle: isDefaultTarget ? DEFAULT_TARGET_HANDLE : topicTargetHandle(e.target.slice('topic-'.length)),
      // 可拖动的线（规则→Topic，唯一可编辑连线）统一用主题强调色。
      style: { stroke: CONNECTABLE_EDGE_COLOR },
    }
  })
}

function CanvasInner({ routing, topics, attributes, edges: edgeRecords, presence, defaultTopicKey, readOnly, caseName, onChange }: CanvasProps) {
  const { getNodes, setViewport, zoomIn, zoomOut, screenToFlowPosition } = useReactFlow()
  const flowRef = useRef<HTMLDivElement>(null)
  const rules = useMemo(() => routing?.rules ?? [], [routing])
  // 画布上用户从右侧池拖入的孤立 Topic（待连线）——用户可能还会再删掉它们，所以原始集合持久保留。
  const [isolatedTopics, setIsolatedTopics] = useState<string[]>([])
  // 当前点击选中的节点 id（active 态 border 变 primary）。
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  // 有效孤立集合 = 原始拖入集合 - 已被规则引用的 Topic。规则连/删线时无需 setState：
  // 规则引用后自动不再是"孤立"，删线后又自动恢复"孤立"。
  const effectiveIsolated = useMemo(
    () => isolatedTopics.filter((key) => !rules.some((r) => r.topic === key)),
    [isolatedTopics, rules],
  )
  const graph = useMemo(
    () => buildBindingGraph(routing, defaultTopicKey, edgeRecords, presence, effectiveIsolated),
    [routing, defaultTopicKey, edgeRecords, presence, effectiveIsolated],
  )
  const bindingByTopic = useMemo(() => {
    const map = new Map<string, TopicBinding>()
    for (const binding of topicBindings(edgeRecords, presence)) map.set(binding.topic, binding)
    return map
  }, [edgeRecords, presence])
  const boundTopicKeys = useMemo(
    () => new Set(bindingByTopic.keys()),
    [bindingByTopic],
  )
  const validation = useMemo(
    () => validateRouting(routing, topics, attributes, boundTopicKeys),
    [routing, topics, attributes, boundTopicKeys],
  )
  const issueByIndex = useMemo(
    () => new Map(validation.issues.map((issue) => [issue.index, issue])),
    [validation],
  )
  const [freePos, setFreePos] = useState<Record<string, { x: number; y: number }>>({})
  const rulesLenRef = useRef(rules.length)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  // 路由变更统一入口：diff 前后被规则引用的 Topic 集合。
  // 任何失去「最后一条规则引用」的 Topic（拖线改连 / 删除连线 / 删除规则）都不消失，
  // 自动收进孤立集合、留在画布（位置沿用当前画布坐标），等用户再次连线或自行处置。
  const handleRoutingChange = useCallback(
    (next: RoutingConfig) => {
      const before = new Set(rules.map((r) => r.topic).filter((t): t is string => Boolean(t)))
      const after = new Set((next.rules ?? []).map((r) => r.topic).filter((t): t is string => Boolean(t)))
      const orphaned = [...before].filter((t) => !after.has(t) && t !== defaultTopicKey)
      if (orphaned.length > 0) {
        setIsolatedTopics((prev) => [...new Set([...prev, ...orphaned])])
        // 沿用孤立前的画布位置，节点不跳位。
        const current = getNodes()
        setFreePos((prev) => {
          const nextPos = { ...prev }
          for (const t of orphaned) {
            const node = current.find((n) => n.id === `topic-${t}`)
            if (node) nextPos[`topic-${t}`] = { x: node.position.x, y: node.position.y }
          }
          return nextPos
        })
      }
      onChange(next)
    },
    [rules, onChange, defaultTopicKey, getNodes],
  )

  const handleAddBranch = useCallback(() => {
    onChange(addRule(routing))
  }, [routing, onChange])

  // ELK 布局完成后把结果写回 freePos：下次输入变化重建节点时保持"上次布局/拖动"位置，
  // 不会跳回默认 slotPos。
  const handleLayouted = useCallback((layouted: Node[]) => {
    setFreePos((prev) => {
      const next: Record<string, { x: number; y: number }> = { ...prev }
      for (const n of layouted) next[n.id] = { x: n.position.x, y: n.position.y }
      return next
    })
  }, [])

  const { bump, fitAfterNextLayout, draggingRef } = useElkLayout(flowRef, handleLayouted)

  // 输入变化时重建节点/边；拖拽期间 React Flow 自己持有位置，不会被覆盖。
  // 增删规则（结构性变化）清空自由摆放位置并触发一次 ELK 重排 + fitView。
  useEffect(() => {
    if (rulesLenRef.current !== rules.length) {
      rulesLenRef.current = rules.length
      setFreePos({})
      fitAfterNextLayout()
      bump()
    }
    setNodes(buildNodes(graph, rules, topics, attributes, bindingByTopic, readOnly, caseName, handleAddBranch, handleRoutingChange, freePos, issueByIndex, selectedNodeId))
    setEdges(buildEdges(graph))
  }, [graph, rules, topics, attributes, bindingByTopic, readOnly, caseName, handleAddBranch, handleRoutingChange, setNodes, setEdges, freePos, issueByIndex, bump, fitAfterNextLayout, selectedNodeId])

  // 节点尺寸/位置变化仅同步给 React Flow；不再触发自动重排（ELK 由用户点「自动整理」触发）。
  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes)
    },
    [onNodesChange],
  )

  // 点击节点 → 选中（active 态）；点击空白 → 取消选中。
  const handleNodeClick = useCallback((_event: unknown, node: Node) => {
    setSelectedNodeId(node.id)
  }, [])
  const handlePaneClick = useCallback(() => {
    setSelectedNodeId(null)
  }, [])

  const handleNodeDragStart = useCallback(() => {
    draggingRef.current = true
  }, [draggingRef])

  // 拖动松手：记录所有节点当前位置到 freePos，之后输入变化时保持用户摆放；不重排规则。
  const handleNodeDragStop = useCallback(() => {
    draggingRef.current = false
    const nextPos: Record<string, { x: number; y: number }> = {}
    for (const n of getNodes()) nextPos[n.id] = { x: n.position.x, y: n.position.y }
    setFreePos(nextPos)
  }, [getNodes, draggingRef])

  const isValidConnection = useCallback((conn: Edge | Connection) => {
    const isBranchSource = Boolean(conn.source?.startsWith('branch-'))
    const isTopicTarget =
      conn.target === DEFAULT_NODE_ID || Boolean(conn.target?.startsWith('topic-'))
    return isBranchSource && isTopicTarget
  }, [])

  const handleConnect = useCallback(
    (conn: Connection) => {
      if (!conn.source?.startsWith('branch-')) return
      if (conn.target !== DEFAULT_NODE_ID && !conn.target?.startsWith('topic-')) return
      const index = Number(conn.source.slice('branch-'.length))
      const topic = conn.target === DEFAULT_NODE_ID ? defaultTopicKey : conn.target!.slice('topic-'.length)
      handleRoutingChange({ rules: rules.map((r, i) => (i === index ? { ...r, topic } : r)) })
    },
    [rules, handleRoutingChange, defaultTopicKey],
  )

  const handleEdgesDelete = useCallback(
    (deleted: Edge[]) => {
      const indices = new Set<number>()
      for (const edge of deleted) {
        if (edge.id.startsWith('e-branch-')) indices.add(Number(edge.id.slice('e-branch-'.length)))
      }
      if (indices.size === 0) return
      handleRoutingChange({ rules: rules.map((r, i) => (indices.has(i) ? { ...r, topic: undefined } : r)) })
    },
    [rules, handleRoutingChange],
  )

  const handleFitView = useCallback(() => {
    const el = flowRef.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const vp = computeFitViewport(getNodes(), rect.width, rect.height)
    void setViewport(vp, { duration: 200 })
  }, [flowRef, getNodes, setViewport])

  // 从右侧 Topic 池拖入画布：作为孤立 Topic 节点（无边），供用户临时调整/稍后连线。
  const handleDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const handleDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()
      const topicKey = event.dataTransfer.getData('application/pixoma-topic')
      if (!topicKey || topicKey === defaultTopicKey) return
      // 已在画布上（被规则引用或已孤立）则忽略。
      const alreadyOnCanvas = rules.some((r) => r.topic === topicKey) || isolatedTopics.includes(topicKey)
      if (alreadyOnCanvas) return
      const position = screenToFlowPosition({ x: event.clientX, y: event.clientY })
      setIsolatedTopics((prev) => [...prev, topicKey])
      setFreePos((prev) => ({ ...prev, [`topic-${topicKey}`]: position }))
    },
    [defaultTopicKey, rules, isolatedTopics, screenToFlowPosition],
  )

  const controlBtn = 'flex h-8 w-8 items-center justify-center text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground disabled:opacity-40'

  return (
    <ReactFlow
      ref={flowRef}
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      className='rf-task-flow h-full w-full'
      onNodesChange={handleNodesChange}
      onEdgesChange={onEdgesChange}
      nodesConnectable={!readOnly}
      nodesFocusable={!readOnly}
      edgesFocusable={!readOnly}
      // 默认即平移/选取（无需切手型按钮）：左键空白处拖动画布，左键节点上拖动节点。
      // nodesDraggable 常开，由 React Flow 自身区分"按下点在节点上 vs 空白处"。
      nodesDraggable={!readOnly}
      panOnDrag
      zoomOnScroll
      zoomOnPinch
      zoomOnDoubleClick
      connectionRadius={24}
      edgesReconnectable={false}
      isValidConnection={isValidConnection}
      onConnect={handleConnect}
      onEdgesDelete={handleEdgesDelete}
      onNodeDragStart={handleNodeDragStart}
      onNodeDragStop={handleNodeDragStop}
      // 点击节点 → active 态（border 变 primary）；空白处点击 → 取消选中。
      onNodeClick={handleNodeClick}
      onPaneClick={handlePaneClick}
      // 右侧 Topic 池拖入画布（onDrop/onDragOver 事件走 React DragEvent 协议）。
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      proOptions={{ hideAttribution: true }}
    >
      <style>{`
        .rf-task-flow {
          --xy-edge-stroke: var(--muted-foreground);
          --xy-edge-stroke-width: 2.5;
          /* 拖线过程线与可拖动的规则→Topic 连线统一为 lab(75.0771% -60.7313 19.4147)。 */
          --xy-connectionline-stroke: lab(75.0771% -60.7313 19.4147);
          --xy-connectionline-stroke-width: 2.5;
        }
        .rf-task-flow .react-flow__node { cursor: default; }
        /* 拖动性能：节点提升到合成层（拖拽期间只 GPU 平移，不触发整卡 repaint）。 */
        .rf-task-flow .react-flow__node { will-change: transform; }
        .rf-task-flow .react-flow__node.dragging { cursor: grabbing; z-index: 10 !important; }
        /* 拖拽期间禁用卡内控件 hover（Select/Input 复杂 hover 逐帧 layout 会卡）。 */
        .rf-task-flow .react-flow__node.dragging :is(button, input, [role='combobox']) {
          pointer-events: none;
        }
        .rf-task-flow .react-flow__edge { cursor: default; }
        /* 官方 elkjs-multiple-handles 示例的多句柄排布 */
        .rf-task-flow .handles {
          position: absolute;
          top: 0;
          height: 100%;
          display: flex;
          flex-direction: column;
          justify-content: space-around;
          pointer-events: none;
        }
        .rf-task-flow .handles.targets { left: 0; transform: translateX(-50%); }
        .rf-task-flow .handles.sources { right: 0; transform: translateX(50%); }
        .rf-task-flow .handles .react-flow__handle {
          position: relative;
          top: 0;
          left: 0;
          transform: none;
          width: 16px;
          height: 16px;
          min-width: 16px;
          min-height: 16px;
          border: none;
          background: transparent;
          border-radius: 9999px;
          pointer-events: all;
          transition: transform 120ms ease;
        }
        .rf-task-flow .handles .react-flow__handle:hover { transform: scale(1.2); }
        .rf-task-flow .handles .react-flow__handle::after {
          content: '';
          position: absolute;
          inset: 3px;
          border-radius: 9999px;
          background: var(--muted-foreground);
          border: 2px solid var(--background);
          pointer-events: none;
          transition: background-color 120ms ease;
        }
        .rf-task-flow .handles .react-flow__handle:hover::after {
          background: var(--primary);
        }
        /* 可拖动链接的圆点（规则右侧源端口、Topic 左侧入端口）：与连线同色。 */
        .rf-task-flow .handles .react-flow__handle.link-handle::after {
          background: lab(75.0771% -60.7313 19.4147);
        }
        .rf-task-flow .handles .react-flow__handle.link-handle:hover::after {
          background: lab(75.0771% -60.7313 19.4147);
        }
        /* 固定端口（Case→规则语义连线）：不可见、不可交互。 */
        .rf-task-flow .handles .react-flow__handle.fixed-handle {
          pointer-events: none;
        }
        .rf-task-flow .handles .react-flow__handle.fixed-handle::after {
          display: none;
        }
      `}</style>
      <Background gap={20} size={1} />
      <div className='absolute bottom-4 left-4 z-10 flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border bg-background shadow-sm'>
        <button type='button' className={controlBtn} aria-label='放大' onClick={() => void zoomIn({ duration: 200 })}>
          <ZoomIn className='size-3.5' />
        </button>
        <button type='button' className={controlBtn} aria-label='缩小' onClick={() => void zoomOut({ duration: 200 })}>
          <ZoomOut className='size-3.5' />
        </button>
        <button type='button' className={controlBtn} aria-label='适应视图' onClick={handleFitView}>
          <Maximize className='size-3.5' />
        </button>
        <button
          type='button'
          className={controlBtn}
          aria-label='自动整理'
          title='自动整理布局（ELK）：按规则顺序重排所有节点'
          onClick={() => {
            setFreePos({})
            fitAfterNextLayout()
            bump()
          }}
        >
          <Wand2 className='size-3.5' />
        </button>
      </div>
      {validation.issues.length > 0 && (
        <div
          data-validation-banner
          className='pointer-events-none absolute top-14 left-1/2 z-10 -translate-x-1/2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive'
        >
          {validation.issues.length} 条规则未通过校验，修复前不可保存（见红框分支）
        </div>
      )}
    </ReactFlow>
  )
}

export function TaskFlowCanvas(props: CanvasProps) {
  return (
    <ReactFlowProvider>
      <CanvasInner {...props} />
    </ReactFlowProvider>
  )
}
