import { useMemo, useState } from 'react'
import { Download } from 'lucide-react'
import { toPng } from 'html-to-image'
import {
  Background,
  Controls,
  Panel,
  getNodesBounds,
  getViewportForBounds,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { TopologyGraph } from './lib/build-link-graph'
import { pathThrough } from './lib/path-through'
import { TopologyNode, type TopologyNodeType } from './topology-node'

const COL: Record<string, number> = {
  platform: 0,
  case: 1,
  topic: 2,
  edge: 3,
}
const COL_W = 220
const ROW_H = 88

const nodeTypes = { topology: TopologyNode }

function layout(
  graph: TopologyGraph,
  selectedId: string | null,
  focusId?: string | null
): TopologyNodeType[] {
  const hit = selectedId ? pathThrough(graph, selectedId) : null
  const grouped = new Map<string, typeof graph.nodes>()
  for (const n of graph.nodes) {
    const list = grouped.get(n.kind) ?? []
    list.push(n)
    grouped.set(n.kind, list)
  }
  const nodes: TopologyNodeType[] = []
  for (const kind of ['platform', 'case', 'topic', 'edge']) {
    const col = COL[kind] ?? 0
    const list = [...(grouped.get(kind) ?? [])].sort((a, b) =>
      a.name.localeCompare(b.name)
    )
    list.forEach((n, i) => {
      const highlighted = hit ? hit.nodes.has(n.id) : false
      const dimmed = Boolean(hit) && !highlighted
      nodes.push({
        id: n.id,
        type: 'topology',
        position: { x: col * COL_W, y: i * ROW_H },
        data: { ...n, highlighted, dimmed, isFocus: n.id === focusId },
        selectable: true,
        draggable: false,
      })
    })
  }
  return nodes
}

function layoutEdges(graph: TopologyGraph, selectedId: string | null): Edge[] {
  const hit = selectedId ? pathThrough(graph, selectedId) : null
  return graph.edges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    style: {
      opacity: hit && !hit.edges.has(e.id) ? 0.2 : 1,
    },
  }))
}

function ExportButton() {
  const { getNodes } = useReactFlow()

  async function exportPng() {
    const nodes = getNodes()
    if (nodes.length === 0) return
    const bounds = getNodesBounds(nodes)
    const imageWidth = Math.ceil(bounds.width + 160)
    const imageHeight = Math.ceil(bounds.height + 160)
    const viewport = getViewportForBounds(bounds, imageWidth, imageHeight, 0.3, 2, 0.1)

    const viewportEl = document.querySelector(
      '.react-flow__viewport'
    ) as HTMLElement | null
    if (!viewportEl) return

    const isDark = document.documentElement.classList.contains('dark')
    const bg = isDark ? '#1c1c1e' : getComputedStyle(document.body).backgroundColor
    const dotColor = isDark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.08)'

    const dataUrl = await toPng(viewportEl, {
      backgroundColor: bg,
      width: imageWidth,
      height: imageHeight,
      style: {
        width: `${imageWidth}px`,
        height: `${imageHeight}px`,
        transform: `translate(${viewport.x}px, ${viewport.y}px) scale(${viewport.zoom})`,
        backgroundImage: `radial-gradient(circle, ${dotColor} 1px, transparent 1px)`,
        backgroundSize: '20px 20px',
      },
      pixelRatio: 2,
    })
    const a = document.createElement('a')
    a.href = dataUrl
    a.download = 'topology.png'
    a.click()
  }

  return (
    <Panel position='bottom-right'>
      <button
        type='button'
        className='react-flow__controls-button'
        title='导出图片'
        onClick={exportPng}
      >
        <Download className='size-4' />
      </button>
    </Panel>
  )
}

function FlowCanvas({
  graph,
  focusId,
}: {
  graph: TopologyGraph
  focusId?: string | null
}) {
  const [selectedId, setSelectedId] = useState<string | null>(focusId ?? null)
  const nodes = useMemo(
    () => layout(graph, selectedId, focusId),
    [graph, selectedId, focusId]
  )
  const edges = useMemo(
    () => layoutEdges(graph, selectedId),
    [graph, selectedId]
  )
  return (
    <div className='h-full min-h-0 w-full'>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => setSelectedId(node.id)}
        nodesDraggable={false}
        nodesConnectable={false}
        deleteKeyCode={null}
        fitView
        minZoom={0.3}
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        <Controls showInteractive={false} />
        <ExportButton />
      </ReactFlow>
    </div>
  )
}

export function LinkGraph({
  graph,
  focusId,
}: {
  graph: TopologyGraph
  focusId?: string | null
}) {
  return (
    <div data-testid='link-graph' className='h-full min-h-0 w-full'>
      <ReactFlowProvider>
        <FlowCanvas graph={graph} focusId={focusId} />
      </ReactFlowProvider>
    </div>
  )
}
