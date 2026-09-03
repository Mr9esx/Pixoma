import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const card = readFileSync(join(here, 'topology-card.tsx'), 'utf8')
const source = readFileSync(join(here, 'use-topology-source.ts'), 'utf8')
const board = readFileSync(
  join(here, '../dashboard/workbench-data-board.tsx'),
  'utf8'
)

describe('topology card contract', () => {
  it('sits at the top of the workbench data board', () => {
    expect(board).toContain('TopologyCard')
    expect(board.indexOf('<TopologyCard')).toBeLessThan(
      board.indexOf('<WorkbenchContribution')
    )
  })

  it('isolates load errors and does not POST channel checks', () => {
    expect(card).toContain("data-testid='topology-card'")
    expect(card).toContain('h-[360px]')
    expect(card).toContain('ErrorBanner')
    expect(source).not.toContain('checkChannelReachability')
    expect(source).toContain('getQueryData')
  })
})
