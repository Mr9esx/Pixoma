import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const graph = readFileSync(join(here, 'link-graph.tsx'), 'utf8')
const node = readFileSync(join(here, 'topology-node.tsx'), 'utf8')

describe('link graph contract', () => {
  it('is a read-only xyflow canvas with stable node types', () => {
    expect(graph).toContain("from '@xyflow/react'")
    expect(graph).toContain('@xyflow/react/dist/style.css')
    expect(graph).toContain("data-testid='link-graph'")
    expect(graph).toContain('nodesDraggable={false}')
    expect(graph).toContain('nodesConnectable={false}')
    expect(graph).toContain('deleteKeyCode={null}')
    expect(graph).toContain('Controls')
    expect(graph).not.toContain('onConnect')
    expect(graph).toContain('nodeTypes')
    expect(node).toContain('StatusDot')
    expect(node).toContain("health !== 'pending'")
    expect(node).toContain('nodrag')
    expect(node).toContain('topology.openDetail')
  })
})
