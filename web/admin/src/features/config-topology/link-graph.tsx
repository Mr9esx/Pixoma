import { useMemo, useState } from 'react'
import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
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

function layout(graph: TopologyGraph, selectedId: string | null): TopologyNodeType[] {
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
        data: { ...n, highlighted, dimmed },
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

function FlowCanvas({ graph }: { graph: TopologyGraph }) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const nodes = useMemo(() => layout(graph, selectedId), [graph, selectedId])
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
      </ReactFlow>
    </div>
  )
}

export function LinkGraph({ graph }: { graph: TopologyGraph }) {
  return (
    <div data-testid='link-graph' className='h-full min-h-0 w-full'>
      <ReactFlowProvider>
        <FlowCanvas graph={graph} />
      </ReactFlowProvider>
    </div>
  )
}
