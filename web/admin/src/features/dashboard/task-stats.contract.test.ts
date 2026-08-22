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
    expect(section).toMatch(/ComposedChart/)
    expect(section).toMatch(/stackId=/)
    expect(section).toMatch(/PieChart/)
  })

  it('organizes the dashboard into three sections with a global range', () => {
    const page = read('dashboard-page.tsx')
    expect(page).toMatch(/RealtimeStatusSection/)
    expect(page).toMatch(/TaskStatsSection/)
    expect(page).toMatch(/CaseAnalysisSection/)
    expect(page).toMatch(/TaskRangePicker/)
    expect(page).toMatch(/fullAccuracyNote/)
    expect(page).not.toMatch(/listTasks/)
    expect(page).not.toMatch(/listCases/)
    expect(page).not.toMatch(/sampleNote/)
  })

  it('uses a fleet chart and case scatter', () => {
    const realtime = read('realtime-status-section.tsx')
    const cases = read('case-analysis-section.tsx')
    expect(realtime).toMatch(/listFleetStats/)
    expect(realtime).toMatch(/BarChart/)
    expect(cases).toMatch(/ScatterChart/)
    expect(cases).toMatch(/listTaskCaseTopStats/)
  })

  it('provides zh/en copy for the dashboard', () => {
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
      'fullAccuracyNote',
      'realtimeStatus',
      'taskPerformance',
      'businessAnalysis',
      'queueExecTitle',
      'statusDonutTitle',
      'perNodeLoadTitle',
      'caseScatterTitle',
      'caseTopTitle',
    ]) {
      expect(zh).toContain(`"${key}"`)
      expect(en).toContain(`"${key}"`)
    }
  })
})
