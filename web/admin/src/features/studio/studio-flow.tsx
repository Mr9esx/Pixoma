import { useEffect, useMemo } from 'react'
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Node,
  type NodeChange,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Box, FileOutput, ListChecks, Workflow } from 'lucide-react'
import type {
  StudioFlowEdge,
  StudioFlowNode,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

type Props = {
  nodes: StudioFlowNode[]
  edges: StudioFlowEdge[]
  onAssetOpen?: (assetId: string) => void
  onPositionsChange?: (
    nodes: Array<{ id: string; position: { x: number; y: number }; sort_order: number }>
  ) => void
}

type FlowData = {
  title: string
  body?: string
  kind: StudioFlowNode['type']
  assetId?: string
  onAssetOpen?: (assetId: string) => void
}

type StudioReactNode = Node<FlowData, 'studio'>

const nodeTypes = { studio: StudioNode }

export function StudioFlow({ nodes: sourceNodes, edges: sourceEdges, onAssetOpen, onPositionsChange }: Props) {
  const initialNodes = useMemo(
    (): StudioReactNode[] =>
      sourceNodes.map((node, index) => ({
        id: node.id,
        type: 'studio',
        position:
          node.position.x !== 0 || node.position.y !== 0
            ? node.position
            : { x: 72 + index * 236, y: 128 + (index % 2) * 54 },
        data: {
          title: node.title,
          body: node.body,
          kind: node.type,
          assetId: node.asset_id,
          onAssetOpen,
        } satisfies FlowData,
      })),
    [sourceNodes, onAssetOpen]
  )
  const initialEdges = useMemo(
    () =>
      sourceEdges.map((edge) => ({
        id: edge.id,
        source: edge.source,
        target: edge.target,
        label: edge.label,
        markerEnd: { type: MarkerType.ArrowClosed },
        style: { stroke: 'var(--color-border)' },
        labelStyle: { fill: 'var(--color-muted-foreground)', fontSize: 11 },
      })),
    [sourceEdges]
  )
  const [nodes, setNodes, onNodesChange] = useNodesState<StudioReactNode>(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)

  useEffect(() => setNodes(initialNodes), [initialNodes, setNodes])
  useEffect(() => setEdges(initialEdges), [initialEdges, setEdges])

  const handleNodesChange = (changes: NodeChange<StudioReactNode>[]) => {
    onNodesChange(changes)
    if (!changes.some((change) => change.type === 'position' && !change.dragging)) return
    const nextNodes = nodes.map((node, index) => {
      const positionChange = changes.find(
        (change) => change.type === 'position' && change.id === node.id && change.position
      )
      return {
        id: node.id,
        position: positionChange?.type === 'position' && positionChange.position ? positionChange.position : node.position,
        sort_order: sourceNodes[index]?.sort_order ?? index,
      }
    })
    onPositionsChange?.(nextNodes)
  }

  if (sourceNodes.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center px-8 text-center'>
        <span className='mb-4 flex size-11 items-center justify-center rounded-xl bg-muted'>
          <Workflow className='size-5 text-muted-foreground' />
        </span>
        <p className='text-sm font-medium'>资产路线还没有节点</p>
        <p className='mt-1 max-w-xs text-xs leading-5 text-muted-foreground'>
          和 Agent 对话后，计划、操作和产出会按时间顺序出现在这里。你可以拖动节点重新组织路线。
        </p>
      </div>
    )
  }

  return (
    <div className='h-full min-h-0 bg-muted/20'>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={handleNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        fitViewOptions={{ padding: 0.22, maxZoom: 1 }}
        minZoom={0.35}
        maxZoom={1.5}
        proOptions={{ hideAttribution: true }}
      >
        <Background color='var(--color-border)' gap={20} size={1} />
        <Controls
          showInteractive={false}
          className='overflow-hidden rounded-lg border bg-popover'
        />
      </ReactFlow>
    </div>
  )
}

function StudioNode({ data, selected }: NodeProps) {
  const value = data as FlowData
  const Icon =
    value.kind === 'stage'
      ? ListChecks
      : value.kind === 'operation'
        ? Workflow
        : value.kind === 'asset'
          ? FileOutput
          : Box
  return (
    <button
      type='button'
      onDoubleClick={() =>
        value.assetId ? value.onAssetOpen?.(value.assetId) : undefined
      }
      className={cn(
        'w-52 rounded-xl border bg-card p-3 text-left transition-colors',
        selected ? 'border-ring ring-3 ring-ring/15' : 'hover:border-ring/60'
      )}
    >
      <Handle type='target' position={Position.Left} className='!bg-muted-foreground' />
      <div className='flex items-start gap-2.5'>
        <span className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted'>
          <Icon className='size-4 text-muted-foreground' />
        </span>
        <div className='min-w-0 flex-1'>
          <div className='mb-1 flex items-center gap-2'>
            <p className='truncate text-sm font-medium'>{value.title}</p>
            {value.kind === 'asset' ? (
              <Badge variant='secondary' className='px-1.5 text-[10px]'>
                资产
              </Badge>
            ) : null}
          </div>
          {value.body ? (
            <p className='line-clamp-2 text-xs leading-5 text-muted-foreground'>
              {value.body}
            </p>
          ) : null}
        </div>
      </div>
      <Handle type='source' position={Position.Right} className='!bg-muted-foreground' />
    </button>
  )
}
