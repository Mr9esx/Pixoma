import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('dashboard workbench contract', () => {
  it('removes the top title/description and renders a welcome card', () => {
    const page = read('dashboard-page.tsx')
    expect(page).not.toMatch(/dashboard\.title/)
    expect(page).not.toMatch(/fullAccuracyNote/)
    expect(page).toContain('WorkbenchWelcomeCard')
  })

  it('renders a welcome card region with quick entry buttons', () => {
    const card = read('workbench-welcome-card.tsx')
    expect(card).toContain("data-testid='workbench-welcome'")
    expect(card).toContain("data-testid='workbench-quick-create-case'")
    expect(card).toContain("data-testid='workbench-quick-manage-edges'")
    expect(card).toContain('fetchCurrentUser')
  })

  it('organizes into a data board with the welcome card on the right', () => {
    const page = read('dashboard-page.tsx')
    expect(page).toMatch(/grid/)
    expect(page).toContain('WorkbenchDataBoard')
    expect(page).toContain('WorkbenchWelcomeCard')
    expect(page).toMatch(/data-testid='workbench-grid'/)
  })

  it('renders the left data board blocks', () => {
    const board = read('workbench-data-board.tsx')
    expect(board).toContain('WorkbenchContribution')
    expect(board).toContain('WorkbenchOverviewCards')
    expect(board).toContain('WorkbenchChartPairs')
    expect(board).toContain('TaskRangePicker')
    expect(board).toContain("data-testid='workbench-data-board'")
  })

  it('renders the overview cards and chart pairs regions', () => {
    const overview = read('workbench-overview-cards.tsx')
    const charts = read('workbench-chart-pairs.tsx')
    expect(overview).toContain("data-testid='workbench-node-overview'")
    expect(overview).toContain("data-testid='workbench-avg-load'")
    expect(overview).toContain("data-testid='workbench-top-load'")
    expect(charts).toContain("data-testid='workbench-workflow-top'")
    expect(charts).toContain("data-testid='workbench-task-duration'")
    expect(charts).toContain("data-testid='workbench-status-distribution'")
    expect(charts).toContain("data-testid='workbench-error-top'")
  })

  it('keeps the heatmap on the full year and overview cards live', () => {
    const contribution = read('workbench-contribution.tsx')
    const overview = read('workbench-overview-cards.tsx')
    expect(contribution).toContain('yearStart')
    expect(contribution).not.toContain('range.from')
    expect(overview).not.toContain('range.from')
  })
  it('uses admin-api query helpers and semantic chart colors', () => {
    const board = read('workbench-data-board.tsx')
    const charts = read('workbench-chart-pairs.tsx')
    expect(board).not.toMatch(/fetch\(["']?\/api\/v1/)
    expect(charts).toContain('var(--color-')
  })
})
