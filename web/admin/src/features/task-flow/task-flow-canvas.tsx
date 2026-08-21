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
import { Hand, Maximize, ZoomIn, ZoomOut } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AttributeDescriptor, EdgePresence, EdgeRecord, RoutingConfig, RoutingRule, TopicRecord } from './types'
import { addRule, buildBindingGraph, moveRule, nodePosition, removeRule, reorderByY, type FlowGraph } from './lib/graph'
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

type CaseData = { label: string; sourceHandles: string[]; ruleCount: number; readOnly: boolean; onAddBranch: () => void }
type TopicData = { label: string; targetHandles: string[]; sourceHandles: string[]; binding: TopicBinding }
type EdgeData = { label: string; targetHandles: string[]; online: boolean }
type BranchData = {
  rule: RoutingRule
  index: number
  total: number
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  readOnly: boolean
  targetHandles: string[]
  sourceHandles: string[]
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
    <div className='relative w-44 rounded-lg border border-border bg-background px-3 py-3 text-sm font-medium shadow-none'>
      <div className='handles sources'>
        {d.sourceHandles.map((h) => (
          <Handle key={h} id={h} type='source' position={Position.Right} />
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
    <div data-branch-node className='relative w-[560px] rounded-lg border-2 border-border bg-background shadow-none'>
      <div className='handles targets'>
        {d.targetHandles.map((h) => (
          <Handle key={h} id={h} type='target' position={Position.Left} />
        ))}
      </div>
      <div className='handles sources'>
        {d.sourceHandles.map((h) => (
          <Handle key={h} id={h} type='source' position={Position.Right} />
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
        dashed ? 'border-dashed border-border bg-muted/30' : 'border-border',
      )}
    >
      <div className='handles targets'>
        {d.targetHandles.map((h) => (
          <Handle key={h} id={h} type='target' position={Position.Left} />
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
        d.online ? 'border-border' : 'border-dashed border-border bg-muted/30',
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
  freePos: Record<number, { x: number; y: number }>,
): Node[] {
  const branchCount = graph.nodes.filter((n) => n.kind === 'condition-branch').length
  return graph.nodes.map((model) => {
    const index = model.ruleIndex ?? 0
    const hasRuleIndex = model.ruleIndex !== undefined
    const slotPos = nodePosition(model, branchCount)
    const base = {
      id: model.id,
      type: model.kind,
      position: hasRuleIndex && freePos[index] ? { ...slotPos, y: freePos[index].y } : slotPos,
      draggable: !readOnly && model.kind === 'condition-branch',
      selectable: false,
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
          onAddBranch,
        } satisfies CaseData,
      }
    }
    if (model.kind === 'topic-target') {
      const topicKey = model.topic ?? model.label
      return {
        ...base,
        data: {
          label: topicName(topics, topicKey),
          targetHandles: [topicTargetHandle(topicKey)],
          sourceHandles: [topicSourceHandle(topicKey)],
          binding: bindingByTopic.get(topicKey) ?? { topic: topicKey, status: 'unbound', boundEdges: [], onlineEdges: [] },
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
      } satisfies EdgeData,
    }
  })
}

function buildEdges(graph: FlowGraph): Edge[] {
  return graph.edges.map((e) => {
    const dashedStyle = e.dashed ? { strokeDasharray: '6 4', stroke: 'var(--muted-foreground)' } : undefined
    if (e.source === CASE_NODE_ID) {
      const isDefault = e.target === DEFAULT_NODE_ID
      const ruleIndex = isDefault ? NaN : Number(e.target.slice('branch-'.length))
      return {
        id: e.id,
        source: e.source,
        sourceHandle: CASE_SOURCE_HANDLE,
        target: e.target,
        targetHandle: isDefault ? DEFAULT_TARGET_HANDLE : branchTargetHandle(ruleIndex),
        style: dashedStyle,
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
      style: dashedStyle,
    }
  })
}

function CanvasInner({ routing, topics, attributes, edges: edgeRecords, presence, defaultTopicKey, readOnly, caseName, onChange }: CanvasProps) {
  const { getNodes, setViewport, zoomIn, zoomOut } = useReactFlow()
  const flowRef = useRef<HTMLDivElement>(null)
  const { bump, fitAfterNextLayout, draggingRef } = useElkLayout(flowRef)
  const rules = routing?.rules ?? []
  const graph = useMemo(
    () => buildBindingGraph(routing, defaultTopicKey, edgeRecords, presence),
    [routing, defaultTopicKey, edgeRecords, presence],
  )
  const bindingByTopic = useMemo(() => {
    const map = new Map<string, TopicBinding>()
    for (const binding of topicBindings(edgeRecords, presence)) map.set(binding.topic, binding)
    return map
  }, [edgeRecords, presence])
  const [freePos, setFreePos] = useState<Record<number, { x: number; y: number }>>({})
  const [panMode, setPanMode] = useState(false)
  const rulesLenRef = useRef(rules.length)
  const firstGraphRef = useRef(true)

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  const handleAddBranch = useCallback(() => {
    onChange(addRule(routing))
    fitAfterNextLayout()
    bump()
  }, [routing, topics, onChange, fitAfterNextLayout, bump])

  // 输入变化时重建节点/边；拖拽期间 React Flow 自己持有位置，不会被覆盖。
  useEffect(() => {
    if (rulesLenRef.current !== rules.length) {
      rulesLenRef.current = rules.length
      setFreePos({}) // 增删规则后回到 ELK 自动布局
    }
    setNodes(buildNodes(graph, rules, topics, attributes, bindingByTopic, readOnly, caseName, handleAddBranch, onChange, freePos))
    setEdges(buildEdges(graph))
  }, [graph, rules, topics, attributes, bindingByTopic, readOnly, caseName, handleAddBranch, onChange, setNodes, setEdges, freePos])

  // 规则/图结构变化 → 防抖触发 ELK 重排（打字期间不跳动）。
  useEffect(() => {
    if (firstGraphRef.current) {
      firstGraphRef.current = false
      return
    }
    bump()
  }, [graph, bump])

  // 节点实测尺寸变化（如条件行数增减）→ 重排。
  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes)
      if (changes.some((c) => c.type === 'dimensions')) bump()
    },
    [onNodesChange, bump],
  )

  const handleNodeDragStart = useCallback(() => {
    draggingRef.current = true
  }, [draggingRef])

  const handleNodeDragStop = useCallback(
    (_: unknown, node: Node) => {
      draggingRef.current = false
      if (node.type !== 'condition-branch') return
      const live = new Map<string, number>()
      for (const n of getNodes()) {
        if (n.type === 'condition-branch') live.set(n.id, n.position.y)
      }
      const oldRules = routing?.rules ?? []
      const positions = oldRules.map((_, i) => ({
        id: `branch-${i}`,
        y: live.get(`branch-${i}`) ?? nodePosition({ id: `branch-${i}`, kind: 'condition-branch', ruleIndex: i, label: '' }, oldRules.length).y,
      }))
      const next = reorderByY({ rules: oldRules }, positions)
      const yById = new Map(positions.map((p) => [p.id, p.y]))
      const nextPos: Record<number, { x: number; y: number }> = {}
      next.rules.forEach((rule, k) => {
        const oldIndex = oldRules.indexOf(rule)
        nextPos[k] = { x: 300, y: yById.get(`branch-${oldIndex}`) ?? 80 + oldIndex * 176 }
      })
      setFreePos(nextPos)
      onChange(next)
    },
    [getNodes, onChange, routing],
  )

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
      onChange({ rules: rules.map((r, i) => (i === index ? { ...r, topic } : r)) })
    },
    [rules, onChange, defaultTopicKey],
  )

  const handleEdgesDelete = useCallback(
    (deleted: Edge[]) => {
      const indices = new Set<number>()
      for (const edge of deleted) {
        if (edge.id.startsWith('e-branch-')) indices.add(Number(edge.id.slice('e-branch-'.length)))
      }
      if (indices.size === 0) return
      onChange({ rules: rules.map((r, i) => (indices.has(i) ? { ...r, topic: undefined } : r)) })
    },
    [rules, onChange],
  )

  const handleFitView = useCallback(() => {
    const el = flowRef.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const vp = computeFitViewport(getNodes(), rect.width, rect.height)
    void setViewport(vp, { duration: 200 })
  }, [flowRef, getNodes, setViewport])

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
      panOnDrag={panMode ? true : [1, 2]}
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
      proOptions={{ hideAttribution: true }}
    >
      <style>{`
        .rf-task-flow {
          --xy-edge-stroke: var(--muted-foreground);
          --xy-edge-stroke-width: 2.5;
          --xy-connectionline-stroke: var(--muted-foreground);
          --xy-connectionline-stroke-width: 2.5;
        }
        .rf-task-flow .react-flow__node { cursor: default; }
        .rf-task-flow .react-flow__node.dragging { cursor: grabbing; }
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
          className={controlBtn + (panMode ? ' bg-primary/10 text-primary' : '')}
          aria-label='平移模式'
          title='手型：按住左键拖动平移（默认中键/右键平移）'
          onClick={() => setPanMode((v) => !v)}
        >
          <Hand className='size-3.5' />
        </button>
      </div>
      {!readOnly && (
        <div className='pointer-events-none absolute top-4 left-1/2 z-10 -translate-x-1/2 rounded-md border border-border bg-background px-3 py-1.5 text-xs text-muted-foreground shadow-none'>
          从规则右侧圆点拖线到 Topic 完成投递（选中连线按 Delete 取消）；点 Case 卡片上的「+ 添加分支」新增规则；滚轮缩放、中键/右键/手型平移
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
