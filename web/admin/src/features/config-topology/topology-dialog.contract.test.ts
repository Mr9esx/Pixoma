import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const read = (rel: string) => readFileSync(join(here, rel), 'utf8')

const panels = [
  '../cases/detail-panel.tsx',
  '../channels/channel-detail-panel.tsx',
  '../topics/topic-detail-panel.tsx',
  '../edges/detail-panel.tsx',
]

describe('topology dialog contract', () => {
  it('adds an open button on four detail pages without removing link health', () => {
    for (const rel of panels) {
      const src = read(rel)
      expect(src, rel).toContain('TopologyOpenButton')
      expect(src, rel).toContain('link-health-section')
    }
  })

  it('dialog uses focused graph mode', () => {
    const dialog = read('topology-dialog.tsx')
    expect(dialog).toContain("data-testid='topology-open'")
    expect(dialog).toContain("type: 'focus'")
    expect(dialog).toContain('topology.title')
  })
})
