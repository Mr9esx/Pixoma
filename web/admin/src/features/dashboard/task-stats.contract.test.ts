import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('dashboard task stats section', () => {
  it('renders a daily bar chart with a range calendar', () => {
    const section = read('task-stats-section.tsx')
    expect(section).toMatch(/data-testid=['"]task-daily-chart-card['"]/)
    expect(section).toMatch(/ChartContainer/)
    expect(section).toMatch(/BarChart/)
    expect(section).toMatch(/mode=['"]range['"]/)
    expect(section).toMatch(/Calendar/)
    expect(section).toMatch(/DAILY_PRESETS/)
  })

  it('replaces the sample-based tasks card with the stats section', () => {
    const page = read('dashboard-page.tsx')
    expect(page).toMatch(/TaskStatsSection/)
    expect(page).not.toMatch(/listTasks/)
    expect(page).not.toMatch(/dashboard-tasks-card/)
  })

  it('provides zh/en copy for daily stats', () => {
    const zh = read('../../lib/i18n/locales/zh.json')
    const en = read('../../lib/i18n/locales/en.json')
    for (const key of [
      'taskDailyTitle',
      'range7d',
      'range30d',
      'range90d',
      'processed',
      'statsSuccessRate',
      'statsErrorTop',
      'statsEdgeLoad',
    ]) {
      expect(zh).toContain(`"${key}"`)
      expect(en).toContain(`"${key}"`)
    }
  })
})
