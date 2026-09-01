import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const src = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'menu-capability-map.tsx'),
  'utf8'
)

describe('MenuCapabilityMap', () => {
  it('is read-only map with edit CTA and trail helpers', () => {
    expect(src).toContain("testId='menu-capability-map'")
    expect(src).toContain("data-testid='map-keyboard'")
    expect(src).toContain('MenuMapLayout')
    expect(src).toContain("data-testid='map-key'")
    expect(src).toContain("data-testid='edit-menu'")
    expect(src).toContain('trailFromItem')
    expect(src).toContain('trailFromOrphan')
    expect(src).toContain('pushTrailButton')
    expect(src).toContain('sliceTrail')
    expect(src).toContain('orphanCards')
    expect(src).toContain('workflowEntryCount')
    expect(src).toContain('actionOutcomeLabel')
    expect(src).toContain('actionIsBroken')
    expect(src).toContain('h-11')
    expect(src).not.toContain('PhoneSimulation')
    expect(src).not.toContain('tab-outline')
    expect(src).not.toContain('‹ 返回')
    expect(src).not.toContain('shadow-')
    expect(src).not.toContain('ActionForm')
    const layout = readFileSync(
      join(dirname(fileURLToPath(import.meta.url)), 'menu-map-layout.tsx'),
      'utf8'
    )
    expect(layout).toContain('min-[920px]:grid-cols-2')
    expect(layout).toContain("data-testid='map-path'")
    expect(layout).toContain('flex-1')
    expect(layout).toContain('min-h-0')
  })

  it('uses MenuMapLayout', () => {
    expect(src).toContain('MenuMapLayout')
  })

  it('renders workflow info card for bound open workflow steps', () => {
    expect(src).toContain(
      "import { WorkflowInfoCard } from './workflow-info-card'"
    )
    expect(src).toContain('WorkflowInfoCard')
    expect(src).toContain("step.action.type === 'open_workflow'")
  })
})
