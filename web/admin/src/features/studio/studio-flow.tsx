import { useEffect, useMemo, useState } from 'react'
import {
  addEdge,
  Background,
  Controls,
  Handle,
  MarkerType,
  Panel,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Connection,
  type Node,
  type NodeChange,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Box, FileOutput, ListChecks, Plus, Workflow } from 'lucide-react'
import type {
  StudioFlowEdge,
  StudioFlowNode,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  nodes: StudioFlowNode[]
  edges: StudioFlowEdge[]
  onAssetOpen?: (assetId: string) => void
  onPositionsChange?: (
    nodes: Array<{ id: string; position: { x: number; y: number }; sort_order: number }>
  ) => void
  onNodeCreate?: (input: {
    type: 'stage' | 'plan' | 'operation'
    title: string
    body?: string
    position: { x: number; y: number }
  }) => Promise<unknown>
  onNodeDelete?: (nodeId: string) => void
  onEdgeCreate?: (input: { source: string; target: string; label?: string }) => Promise<unknown>
  onEdgeDelete?: (edgeId: string) => void
}

type FlowData = {
  title: string
  body?: string
  kind: StudioFlowNode['type']
  assetId?: string
	assetVersion?: number
  onAssetOpen?: (assetId: string) => void
}

type StudioReactNode = Node<FlowData, 'studio'>

const nodeTypes = { studio: StudioNode }

export function StudioFlow({
  nodes: sourceNodes,
  edges: sourceEdges,
  onAssetOpen,
  onPositionsChange,
  onNodeCreate,
  onNodeDelete,
  onEdgeCreate,
  onEdgeDelete,
}: Props) {
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
		  assetVersion: node.asset_version,
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

  const handleConnect = (connection: Connection) => {
    if (!connection.source || !connection.target) return
    if (edges.some((edge) => edge.source === connection.source && edge.target === connection.target)) return
    if (!onEdgeCreate) return
    void onEdgeCreate({ source: connection.source, target: connection.target }).then(() => {
      setEdges((current) =>
        addEdge(
          {
            ...connection,
            markerEnd: { type: MarkerType.ArrowClosed },
            style: { stroke: 'var(--color-border)' },
            labelStyle: { fill: 'var(--color-muted-foreground)', fontSize: 11 },
          },
          current
        )
      )
    })
  }

  if (sourceNodes.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center px-8 text-center'>
        <span className='mb-4 flex size-11 items-center justify-center rounded-xl bg-muted'>
          <Workflow className='size-5 text-muted-foreground' />
        </span>
        <p className='text-sm font-medium'>资产路线还没有节点</p>
        <p className='mt-1 max-w-xs text-xs leading-5 text-muted-foreground'>
          和 Agent 对话后，计划、操作和产出会按时间顺序出现在这里。你也可以先手动搭好 SOP。
        </p>
        {onNodeCreate ? (
          <CreateFlowNodeDialog onCreate={onNodeCreate} nodeCount={0} />
        ) : null}
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
        onNodesDelete={(deleted) => deleted.forEach((node) => onNodeDelete?.(node.id))}
        onEdgesDelete={(deleted) => deleted.forEach((edge) => onEdgeDelete?.(edge.id))}
        onConnect={handleConnect}
        deleteKeyCode={['Backspace', 'Delete']}
        fitView
        fitViewOptions={{ padding: 0.22, maxZoom: 1 }}
        minZoom={0.35}
        maxZoom={1.5}
        proOptions={{ hideAttribution: true }}
      >
        <Background color='var(--color-border)' gap={20} size={1} />
        <Panel position='top-left'>
          <div className='flex items-center gap-2 rounded-lg border bg-card p-1.5'>
            {onNodeCreate ? (
              <CreateFlowNodeDialog
                onCreate={onNodeCreate}
                nodeCount={sourceNodes.length}
              />
            ) : null}
            <span className='hidden px-1 text-xs text-muted-foreground 2xl:inline'>
              拖动节点、拖出连线；选中后按 Delete 移除
            </span>
          </div>
        </Panel>
        <Controls
          showInteractive={false}
          className='overflow-hidden rounded-lg border bg-popover'
        />
      </ReactFlow>
    </div>
  )
}

function CreateFlowNodeDialog({
  onCreate,
  nodeCount,
}: {
  onCreate: (input: {
    type: 'stage' | 'plan' | 'operation'
    title: string
    body?: string
    position: { x: number; y: number }
  }) => Promise<unknown>
  nodeCount: number
}) {
  const [open, setOpen] = useState(false)
  const [type, setType] = useState<'stage' | 'plan' | 'operation'>('plan')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const submit = async () => {
    if (!title.trim()) return
    setSubmitting(true)
    setError('')
    try {
      await onCreate({
        type,
        title: title.trim(),
        body: body.trim(),
        position: { x: 72 + nodeCount * 76, y: 120 + (nodeCount % 3) * 74 },
      })
      setOpen(false)
      setTitle('')
      setBody('')
    } catch {
      setError('节点创建失败，请稍后重试。')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant='ghost' size='sm' className='rounded-md'>
          <Plus />
          新增节点
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>新增 SOP 节点</DialogTitle>
          <DialogDescription>
            节点只调整当前 Session 的资产路线，不会改动已有资产或执行记录。
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-4 py-2'>
          <div className='space-y-2'>
            <Label>节点类型</Label>
            <div className='flex flex-wrap gap-2'>
              {([
                ['stage', '阶段'],
                ['plan', '计划'],
                ['operation', '操作'],
              ] as const).map(([value, label]) => (
                <Button
                  key={value}
                  type='button'
                  variant={type === value ? 'secondary' : 'outline'}
                  size='sm'
                  onClick={() => setType(value)}
                >
                  {label}
                </Button>
              ))}
            </div>
          </div>
          <div className='space-y-2'>
            <Label htmlFor='studio-flow-node-title'>节点名称</Label>
            <Input
              id='studio-flow-node-title'
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder='例如：确认本话色彩脚本'
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='studio-flow-node-description'>补充说明</Label>
            <Textarea
              id='studio-flow-node-description'
              value={body}
              onChange={(event) => setBody(event.target.value)}
              placeholder='写下这个步骤需要完成什么…'
              className='min-h-24'
            />
          </div>
          {error ? <p role='alert' className='text-sm text-destructive'>{error}</p> : null}
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => setOpen(false)}>取消</Button>
          <Button disabled={!title.trim() || submitting} onClick={() => void submit()}>
            {submitting ? '正在添加…' : '添加节点'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
                {value.assetVersion ? `v${value.assetVersion}` : '已固定'}
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
