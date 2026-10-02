import { useEffect, useMemo, useState } from 'react'
import {
  addEdge,
  Handle,
  MarkerType,
  Panel,
  Position,
  useUpdateNodeInternals,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge as ReactFlowEdge,
  type Node,
  type NodeChange,
  type NodeProps,
} from '@xyflow/react'
import { Box, FileOutput, ListChecks, Plus, Workflow } from 'lucide-react'
import type {
  StudioFlowEdge,
  StudioFlowNode,
  StudioWorkflowExecution,
} from '@/lib/api/studio'
import { cn } from '@/lib/utils'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Canvas } from '@/components/ai-elements/canvas'
import { Controls } from '@/components/ai-elements/controls'
import { Edge } from '@/components/ai-elements/edge'
import {
  Node as FlowNode,
  NodeDescription,
  NodeHeader,
  NodeTitle,
} from '@/components/ai-elements/node'
import { StatusDot } from '@/components/status-dot'

type Props = {
  nodes: StudioFlowNode[]
  edges: StudioFlowEdge[]
  background?: boolean
  workflowExecutions?: StudioWorkflowExecution[]
  onAssetOpen?: (assetId: string) => void
  onPositionsChange?: (
    nodes: Array<{
      id: string
      position: { x: number; y: number }
      sort_order: number
    }>
  ) => Promise<unknown>
  onNodeCreate?: (input: {
    type: 'stage' | 'plan' | 'operation'
    title: string
    body?: string
    position: { x: number; y: number }
  }) => Promise<unknown>
  onNodeDelete?: (nodeId: string) => Promise<unknown>
  onEdgeCreate?: (input: {
    source: string
    target: string
    source_output_key?: string
    target_input_key?: string
    label?: string
  }) => Promise<unknown>
  onEdgeDelete?: (edgeId: string) => Promise<unknown>
}

type FlowData = {
  title: string
  body?: string
  kind: StudioFlowNode['type']
  assetId?: string
  assetVersion?: number
  onAssetOpen?: (assetId: string) => void
  workflowExecution?: StudioWorkflowExecution
  inputAssets: Array<{ key: string; name: string }>
  outputs: NonNullable<StudioFlowNode['outputs']>
}

type StudioReactNode = Node<FlowData, 'studio'>

const nodeTypes = { studio: StudioNode }
const edgeTypes = { animated: Edge.Animated }
const fieldTypeLabels: Record<string, string> = { string: '文本', text: '文本', enum: '选项', number: '数字', boolean: '布尔值', image: '图片', video: '视频', audio: '音频', file: '文件', asset: '资产' }

export function StudioFlow({
  nodes: sourceNodes,
  edges: sourceEdges,
  background = true,
  workflowExecutions,
  onAssetOpen,
  onPositionsChange,
  onNodeCreate,
  onNodeDelete,
  onEdgeCreate,
  onEdgeDelete,
}: Props) {
  const initialNodes = useMemo((): StudioReactNode[] => {
    const executionByNode = new Map(
      workflowExecutions?.map((execution) => [
        execution.operation_node_id,
        execution,
      ]) ?? []
    )
    const nodeByID = new Map(sourceNodes.map((node) => [node.id, node]))
    return sourceNodes.map((node, index) => ({
      id: node.id,
      type: 'studio',
      deletable: !executionByNode.has(node.id),
      position:
        node.position.x !== 0 || node.position.y !== 0
          ? node.position
          : { x: 72 + index * 448, y: 128 },
      data: {
        title: node.title,
        body: node.body,
        kind: node.type,
        assetId: node.asset_id,
        assetVersion: node.asset_version,
        onAssetOpen,
        workflowExecution: executionByNode.get(node.id),
        inputAssets: sourceEdges.filter((edge) => edge.target === node.id).map((edge) => nodeByID.get(edge.source)).filter((source): source is StudioFlowNode => source?.type === 'asset').map((source) => ({ key: source.id, name: source.title })),
        outputs: node.outputs ?? [],
      } satisfies FlowData,
    }))
  }, [sourceNodes, sourceEdges, workflowExecutions, onAssetOpen])
  const initialEdges = useMemo(() => {
    const runningNodes = new Set(
      workflowExecutions
        ?.filter(
          (execution) =>
            execution.status === 'submitted' &&
            execution.task_status === 'running'
        )
        .map((execution) => execution.operation_node_id) ?? []
    )
    return sourceEdges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      sourceHandle: edge.source_output_key ? `out:${edge.source_output_key}` : undefined,
      targetHandle: edge.target_input_key ? `in:${edge.target_input_key}` : undefined,
      deletable: !edge.target_input_key,
      label: edge.label,
      type:
        !edge.label &&
        (runningNodes.has(edge.source) || runningNodes.has(edge.target))
          ? 'animated'
          : undefined,
      markerEnd: { type: MarkerType.ArrowClosed },
      style: { stroke: 'var(--color-border)' },
      labelStyle: { fill: 'var(--color-muted-foreground)', fontSize: 14 },
    }))
  }, [sourceEdges, workflowExecutions])
  const [nodes, setNodes, onNodesChange] =
    useNodesState<StudioReactNode>(initialNodes)
  const [edges, setEdges, onEdgesChange] =
    useEdgesState<ReactFlowEdge>(initialEdges)
  const [flowError, setFlowError] = useState('')
  const [savingPositions, setSavingPositions] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => setNodes(initialNodes), [initialNodes, setNodes])
  useEffect(() => setEdges(initialEdges), [initialEdges, setEdges])

  const handleNodesChange = (changes: NodeChange<StudioReactNode>[]) => {
    if (savingPositions && changes.some((change) => change.type === 'position'))
      return
    onNodesChange(changes)
    if (
      !changes.some((change) => change.type === 'position' && !change.dragging)
    )
      return
    if (!onPositionsChange) return
    const nextNodes = nodes.map((node, index) => {
      const positionChange = changes.find(
        (change) =>
          change.type === 'position' && change.id === node.id && change.position
      )
      return {
        id: node.id,
        position:
          positionChange?.type === 'position' && positionChange.position
            ? positionChange.position
            : node.position,
        sort_order: sourceNodes[index]?.sort_order ?? index,
      }
    })
    setFlowError('')
    setSavingPositions(true)
    void onPositionsChange(nextNodes)
      .catch(() => {
        setNodes(initialNodes)
        setFlowError('节点位置保存失败，已恢复原位置。')
      })
      .finally(() => setSavingPositions(false))
  }

  const handleConnect = (connection: Connection) => {
    if (
      !connection.source ||
      !connection.target ||
      connection.source === connection.target ||
      connecting
    )
      return
    if (
      edges.some(
        (edge) =>
          edge.source === connection.source && edge.target === connection.target && (edge.sourceHandle ?? null) === connection.sourceHandle && (edge.targetHandle ?? null) === connection.targetHandle
      )
    )
      return
    if (!onEdgeCreate) return
    setFlowError('')
    setConnecting(true)
    void onEdgeCreate({ source: connection.source, target: connection.target, source_output_key: connection.sourceHandle?.replace(/^out:/, ''), target_input_key: connection.targetHandle?.replace(/^in:/, '') })
      .then(() => {
        setEdges((current) =>
          addEdge(
            {
              ...connection,
              markerEnd: { type: MarkerType.ArrowClosed },
              style: { stroke: 'var(--color-border)' },
              labelStyle: {
                fill: 'var(--color-muted-foreground)',
                fontSize: 14,
              },
            },
            current
          )
        )
      })
      .catch(() => setFlowError('连线创建失败，重新连接节点。'))
      .finally(() => setConnecting(false))
  }

  return (
    <div className={cn('relative h-full min-h-0', background && 'bg-card')}>
      <Canvas<StudioReactNode, ReactFlowEdge>
        background={background}
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={handleNodesChange}
        onEdgesChange={onEdgesChange}
        onBeforeDelete={async ({
          nodes: deletingNodes,
          edges: deletingEdges,
        }) => {
          if (deleting) return false
          setFlowError('')
          setDeleting(true)
          try {
            for (const node of deletingNodes) {
              if (!onNodeDelete) return false
              await onNodeDelete(node.id)
            }
            const removedNodeIDs = new Set(deletingNodes.map((node) => node.id))
            for (const edge of deletingEdges) {
              if (
                removedNodeIDs.has(edge.source) ||
                removedNodeIDs.has(edge.target)
              )
                continue
              if (!onEdgeDelete) return false
              await onEdgeDelete(edge.id)
            }
            return true
          } catch {
            setFlowError(
              deletingNodes.length
                ? '节点删除失败，制作流程未全部更新。'
                : '连线删除失败，制作流程未更新。'
            )
            return false
          } finally {
            setDeleting(false)
          }
        }}
        onConnect={handleConnect}
        isValidConnection={(connection) =>
          connection.source !== connection.target &&
          !edges.some(
            (edge) =>
              edge.source === connection.source &&
              edge.target === connection.target && (edge.sourceHandle ?? null) === connection.sourceHandle && (edge.targetHandle ?? null) === connection.targetHandle
          )
        }
        nodesDraggable={!savingPositions}
        nodesConnectable={!connecting}
        deleteKeyCode={deleting ? null : ['Backspace', 'Delete']}
        fitView
        fitViewOptions={{ padding: 0.22, maxZoom: 1 }}
        minZoom={0.35}
        maxZoom={1.5}
        proOptions={{ hideAttribution: true }}
      >
        {flowError ? (
          <Panel position='top-right' style={{ top: 64 }}>
            <Alert
              role='alert'
              variant='destructive'
              className='max-w-xs bg-card'
            >
              <AlertTitle>{flowError}</AlertTitle>
            </Alert>
          </Panel>
        ) : null}
        {onNodeCreate ? (
          <Panel position='bottom-left'>
            <CreateFlowNodeDialog
              onCreate={onNodeCreate}
              nodeCount={sourceNodes.length}
            />
          </Panel>
        ) : null}
        <Controls
          position='bottom-right'
          showInteractive={false}
          className='overflow-hidden rounded-lg border bg-popover'
        />
      </Canvas>
      {sourceNodes.length === 0 ? (
        <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center px-8 text-center'>
          <p className='text-sm text-muted-foreground'>暂无制作流程</p>
        </div>
      ) : null}
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
        position: { x: 72 + nodeCount * 448, y: 128 },
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
      <IconButtonTooltip label='新增节点'>
        <DialogTrigger asChild>
          <Button
            variant='secondary'
            size='icon'
            className='size-11 rounded-full'
            aria-label='新增节点'
          >
            <Plus />
          </Button>
        </DialogTrigger>
      </IconButtonTooltip>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>新增流程节点</DialogTitle>
          <DialogDescription>
            节点只调整当前会话的制作流程，不会改动已有资产或运行记录。
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-4 py-2'>
          <div className='space-y-2'>
            <Label>节点类型</Label>
            <div className='flex flex-wrap gap-2'>
              {(
                [
                  ['stage', '阶段'],
                  ['plan', '计划'],
                  ['operation', '操作'],
                ] as const
              ).map(([value, label]) => (
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
          {error ? (
            <p role='alert' className='text-sm text-destructive'>
              {error}
            </p>
          ) : null}
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => setOpen(false)}>
            取消
          </Button>
          <Button
            disabled={!title.trim() || submitting}
            onClick={() => void submit()}
          >
            {submitting ? '正在添加…' : '添加节点'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function StudioNode({ id, data, selected }: NodeProps) {
  const value = data as FlowData
  const assetId = value.assetId
  const execution = value.workflowExecution
  const updateNodeInternals = useUpdateNodeInternals()
  useEffect(() => { updateNodeInternals(id) }, [id, value.outputs, execution?.input_fields, execution?.output_fields, updateNodeInternals])
  const statusLabel = execution
    ? {
        submitted:
          execution.task_status === 'pending'
            ? '待处理'
            : execution.task_status === 'queued'
              ? '排队中'
              : '执行中',
        succeeded: '成功',
        failed: '失败',
        cancelled: '已取消',
      }[execution.status]
    : undefined
  const body = execution?.status === 'failed' ? execution.error_message || '工作流失败。' : execution?.status === 'cancelled' ? '工作流已取消。' : execution ? undefined : value.body
  if (execution || value.outputs.length) {
    const inputFields = execution?.input_fields?.length ? execution.input_fields : value.inputAssets.map((asset) => ({ key: asset.key, type: 'asset', description: '输入资产' }))
    const outputFields = execution?.output_fields?.length ? execution.output_fields : value.outputs.map((output) => ({ key: output.key, type: output.type, description: undefined as string | undefined }))
    const outputRows = [
      ...outputFields.map((field) => ({ field, output: value.outputs.find((item) => item.key === field.key) })),
      ...value.outputs.filter((output) => !outputFields.some((field) => field.key === output.key)).map((output) => ({ field: { key: output.key, type: output.type, description: undefined as string | undefined }, output })),
    ]
    return <Card className={cn('relative w-96 gap-0 rounded-md bg-card py-0 shadow-none transition-colors', selected ? 'border-ring ring-3 ring-ring/15' : 'hover:border-ring/60')}>
      <Handle type='target' position={Position.Left} isConnectable={false} className='opacity-0' style={{ top: 24 }} />
      <Handle type='source' position={Position.Right} isConnectable={false} className='opacity-0' style={{ top: 24 }} />
      <CardHeader className='flex flex-row items-center gap-2 rounded-t-md border-b bg-muted/40 px-4 py-3'>
        <Workflow aria-hidden='true' className='size-4 shrink-0 text-muted-foreground' /><span className='min-w-0 flex-1 truncate text-sm font-semibold'>{value.title}</span>
        {statusLabel && execution ? <Badge variant='outline' className='gap-1.5 px-2 text-sm'><StatusDot label={statusLabel} state={execution.status === 'succeeded' ? 'ok' : execution.status === 'submitted' ? 'active' : 'warn'} />{statusLabel}</Badge> : null}
      </CardHeader>
      <CardContent className='flex flex-col gap-4 px-0 py-3'>
        <section aria-label='输入' className='flex flex-col gap-1'>
          <h3 className='px-4 text-sm font-medium text-muted-foreground'>输入</h3>
          {inputFields.length ? inputFields.map((field) => {
            const input = execution?.inputs?.find((item) => item.key === field.key)
            const display = input?.asset_name || input?.value || value.inputAssets.find((asset) => asset.key === field.key)?.name || (input ? '已提供' : '未提供')
            return <div key={field.key} className='relative flex min-h-11 items-center gap-2 px-4 py-1 text-sm'>
              <Handle id={`in:${field.key}`} type='target' position={Position.Left} isConnectable={Boolean(execution)} className='!size-11 !border-0 !bg-transparent before:absolute before:inset-4 before:rounded-full before:border-2 before:border-card before:bg-muted-foreground' style={{ left: -22, top: '50%' }} />
              <span className='min-w-0 flex-1 truncate font-medium' title={field.description || field.key}>{field.description || field.key}</span>
              <Badge variant='secondary' className='shrink-0 text-sm'>{fieldTypeLabels[field.type] || field.type}</Badge>
              <span className='max-w-32 truncate text-muted-foreground' title={display}>{display}</span>
            </div>
          }) : <p className='px-4 text-sm text-muted-foreground'>无输入记录</p>}
        </section>
        <section aria-label='输出' className='flex flex-col gap-1'>
          <h3 className='px-4 text-sm font-medium text-muted-foreground'>输出</h3>
          {outputRows.length ? outputRows.map(({ field, output }) => <div key={field.key} className='relative flex min-h-11 items-center gap-2 px-4 py-1 text-sm'>
            {output ? <Handle id={`out:${field.key}`} type='source' position={Position.Right} className='!size-11 !border-0 !bg-transparent before:absolute before:inset-4 before:rounded-full before:border-2 before:border-card before:bg-muted-foreground' style={{ right: -22, top: '50%' }} /> : null}
            <span className='min-w-0 flex-1 truncate font-medium' title={field.description || field.key}>{field.description || field.key}</span>
            <Badge variant='secondary' className='shrink-0 text-sm'>{fieldTypeLabels[field.type] || field.type}</Badge>
            {output ? <Button type='button' size='sm' variant='link' className='nodrag max-w-36 min-w-0 truncate px-0 text-sm' title={output.name} onClick={() => value.onAssetOpen?.(output.asset_id)}>{output.name}</Button> : <span className='text-sm text-muted-foreground'>待生成</span>}
          </div>) : <p className='px-4 text-sm text-muted-foreground'>暂无产物</p>}
        </section>
        {body ? <p className='px-4 text-sm text-muted-foreground'>{body}</p> : null}
      </CardContent>
    </Card>
  }
  const Icon =
    value.kind === 'stage'
      ? ListChecks
      : value.kind === 'operation'
        ? Workflow
        : value.kind === 'asset'
          ? FileOutput
          : Box
  return (
    <FlowNode
      handles={{ target: true, source: true }}
      onDoubleClick={assetId ? () => value.onAssetOpen?.(assetId) : undefined}
      className={cn(
        'w-64 transition-colors',
        selected ? 'border-ring ring-3 ring-ring/15' : 'hover:border-ring/60'
      )}
    >
      <NodeHeader className='gap-2 border-b bg-muted/40'>
        <div className='flex items-center gap-2'>
          <Icon className='size-4 shrink-0 text-muted-foreground' />
          <NodeTitle className='min-w-0 flex-1 truncate text-sm'>
            {assetId ? (
              <button
                type='button'
                className='max-w-full truncate text-left focus-visible:outline-2 focus-visible:outline-ring'
                onKeyDown={(event) => {
                  if (event.key !== 'Enter' && event.key !== ' ') return
                  event.preventDefault()
                  event.stopPropagation()
                  value.onAssetOpen?.(assetId)
                }}
              >
                {value.title}
              </button>
            ) : (
              value.title
            )}
          </NodeTitle>
          {value.kind === 'asset' ? (
            <Badge variant='secondary' className='px-2 text-sm'>
              {value.assetVersion ? `v${value.assetVersion}` : '已固定'}
            </Badge>
          ) : null}
        </div>
        {body ? (
          <NodeDescription className='line-clamp-3 text-sm leading-5'>
            {body}
          </NodeDescription>
        ) : null}
      </NodeHeader>
    </FlowNode>
  )
}
