import { useMemo, useState } from 'react'
import {
  Background,
  Controls,
  ReactFlow,
  type Edge,
  type Node,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { nodeLabel } from '../lib/node-catalog'

type Props = {
  record: CaseRecord
}

type RawNode = {
  class_type?: unknown
  inputs?: unknown
}

function workflowNodes(record: CaseRecord): Record<string, RawNode> {
  return (record.bindings.workflow ?? {}) as Record<string, RawNode>
}

function connectedInputs(node: RawNode): Array<[string, number]> {
  const out: Array<[string, number]> = []
  const inputs = node.inputs
  if (!inputs || typeof inputs !== 'object') return out
  for (const value of Object.values(inputs as Record<string, unknown>)) {
    if (Array.isArray(value) && typeof value[0] === 'string') {
      out.push([value[0], typeof value[1] === 'number' ? value[1] : 0])
    }
  }
  return out
}

export function WorkflowGraphPreview({ record }: Props) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<'simple' | 'full'>('simple')

  const { nodes, edges } = useMemo(() => {
    const workflow = workflowNodes(record)
    const inputBound = new Set(record.bindings.inputs.map((b) => b.node_id))
    const outputBound = new Set(record.bindings.outputs.map((b) => b.node_id))
    const inputKeyByNode = new Map<string, string>()
    for (const b of record.bindings.inputs) {
      if (!inputKeyByNode.has(b.node_id)) inputKeyByNode.set(b.node_id, b.key)
    }
    const outputKeyByNode = new Map<string, string>()
    for (const b of record.bindings.outputs) {
      if (!outputKeyByNode.has(b.node_id)) outputKeyByNode.set(b.node_id, b.key)
    }

    if (mode === 'simple') {
      const inputIds = [...inputBound]
      const outputIds = [...outputBound]
      const nodes: Node[] = inputIds.map((id, i) => ({
        id: `in-${id}`,
        position: { x: 0, y: i * 110 },
        data: {
          label: `${inputKeyByNode.get(id) ?? id}\n${nodeLabel(
            String(workflow[id]?.class_type ?? id)
          )}`,
        },
        style: simpleStyle('input'),
      }))
      outputIds.forEach((id, i) => {
        nodes.push({
          id: `out-${id}`,
          position: { x: 380, y: i * 110 },
          data: {
            label: `${outputKeyByNode.get(id) ?? id}\n${nodeLabel(
              String(workflow[id]?.class_type ?? id)
            )}`,
          },
          style: simpleStyle('output'),
        })
      })
      const edges: Edge[] = []
      for (const src of inputIds) {
        for (const dst of outputIds) {
          edges.push({
            id: `e-${src}-${dst}`,
            source: `in-${src}`,
            target: `out-${dst}`,
            markerEnd: { type: 'arrowclosed' },
          })
        }
      }
      return { nodes, edges }
    }

    // full mode
    const ids = Object.keys(workflow)
    const layer = new Map<string, number>(ids.map((id) => [id, 0]))
    const incoming = new Map<string, string[]>()
    for (const id of ids) incoming.set(id, [])
    for (const id of ids) {
      for (const [src] of connectedInputs(workflow[id])) {
        if (!workflow[src]) continue
        incoming.get(id)?.push(src)
      }
    }
    for (let pass = 0; pass < ids.length; pass += 1) {
      let changed = false
      for (const id of ids) {
        for (const src of incoming.get(id) ?? []) {
          const next = (layer.get(src) ?? 0) + 1
          if (next > (layer.get(id) ?? 0)) {
            layer.set(id, next)
            changed = true
          }
        }
      }
      if (!changed) break
    }
    const rows = new Map<number, number>()
    const nodes: Node[] = ids.map((id) => {
      const l = layer.get(id) ?? 0
      const y = (rows.get(l) ?? 0) * 120
      rows.set(l, (rows.get(l) ?? 0) + 1)
      const isInput = inputBound.has(id)
      const isOutput = outputBound.has(id)
      const border =
        isInput && isOutput
          ? '#a855f7'
          : isInput
            ? '#f59e0b'
            : isOutput
              ? '#10b981'
              : '#cbd5e1'
      return {
        id,
        position: { x: l * 280, y },
        data: {
          label: `${nodeLabel(String(workflow[id]?.class_type ?? id))}\n${id}`,
        },
        style: { border: `2px solid ${border}`, borderRadius: 8, padding: 8 },
      }
    })
    const edges: Edge[] = []
    for (const id of ids) {
      for (const [src, slot] of connectedInputs(workflow[id])) {
        if (!workflow[src]) continue
        edges.push({
          id: `e-${src}-${id}-${slot}`,
          source: src,
          target: id,
          markerEnd: { type: 'arrowclosed' },
        })
      }
    }
    return { nodes, edges }
  }, [mode, record])

  return (
    <div className='space-y-3' data-testid='case-workflow-graph-preview'>
      <div className='flex flex-wrap items-center gap-2'>
        <div className='flex overflow-hidden rounded-md border'>
          <button
            type='button'
            onClick={() => setMode('simple')}
            className={`px-3 py-1.5 text-xs ${
              mode === 'simple'
                ? 'bg-primary text-primary-foreground'
                : 'bg-background hover:bg-accent'
            }`}
          >
            {t('cases.graphSimple')}
          </button>
          <button
            type='button'
            onClick={() => setMode('full')}
            className={`px-3 py-1.5 text-xs ${
              mode === 'full'
                ? 'bg-primary text-primary-foreground'
                : 'bg-background hover:bg-accent'
            }`}
          >
            {t('cases.graphFull')}
          </button>
        </div>
        {mode === 'full' ? (
          <div className='flex items-center gap-3 text-xs text-muted-foreground'>
            <span className='flex items-center gap-1.5'>
              <i className='inline-block size-2.5 rounded-sm border-2 border-amber-500' />
              {t('cases.graphInputNode')}
            </span>
            <span className='flex items-center gap-1.5'>
              <i className='inline-block size-2.5 rounded-sm border-2 border-emerald-500' />
              {t('cases.graphOutputNode')}
            </span>
          </div>
        ) : null}
      </div>
      <div className='h-[360px] rounded-md border'>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          nodesDraggable={false}
          proOptions={{ hideAttribution: true }}
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  )
}

function simpleStyle(kind: 'input' | 'output') {
  return {
    border: `2px solid ${kind === 'input' ? '#f59e0b' : '#10b981'}`,
    borderRadius: 8,
    padding: 8,
  }
}
