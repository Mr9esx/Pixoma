import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const node = readFileSync(join(here, 'topology-node.tsx'), 'utf8')

describe('topology node contract', () => {
  it('renders per-kind lucide icons, kind labels, and focus border', () => {
    expect(node).toContain("from 'lucide-react'")
    expect(node).toContain('Radio')
    expect(node).toContain('Boxes')
    expect(node).toContain('Tags')
    expect(node).toContain('Server')
    expect(node).toContain('topology.kindPlatform')
    expect(node).toContain('topology.kindCase')
    expect(node).toContain('topology.kindTopic')
    expect(node).toContain('topology.kindEdge')
    expect(node).toContain('isFocus')
    expect(node).toContain('border-primary')
    expect(node).toContain('StatusDot')
    expect(node).not.toContain("health !== 'pending'")
  })
})
