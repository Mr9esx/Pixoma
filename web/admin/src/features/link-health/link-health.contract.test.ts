import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('link health visibility', () => {
  it('LinkHealthAlert 在 Header 展示告警摘要（仅异常时）', () => {
    const alert = read('link-health-alert.tsx')
    expect(alert).toContain("data-testid='link-health-alert'")
    expect(alert).toContain("role='alert'")
    expect(alert).toContain('linkHealth.alertTitle')
    expect(alert).toContain('linkHealth.alertSummary')
    expect(alert).toContain('linkHealth.alertViewDetails')
    expect(alert).toContain('anchorTo')
    expect(alert).toContain("health.state === 'ok'")
  })

  it('LinkHealthSection 提供断点（分类 + 行动 + 指引）与引用列表', () => {
    const section = read('link-health-section.tsx')
    expect(section).toContain("data-testid='link-health-section'")
    expect(section).toContain("data-testid='link-health-breakpoints'")
    expect(section).toContain('SectionHead')
    expect(section).toContain('linkHealth.sectionHint')
    expect(section).toContain('divide-y')
    expect(section).toContain('linkHealth.fixConfig')
    expect(section).toContain('linkHealth.fixRuntime')
    expect(section).toContain('b.action.to')
    expect(section).toContain('b.guide')
    expect(section).toContain('linkHealth.howToHandle')
    expect(section).toContain("data-testid='link-health-reference-list'")
  })

  it('i18n linkHealth 命名空间 zh/en 成对', () => {
    const zh = JSON.parse(read('../../lib/i18n/locales/zh.json'))
    const en = JSON.parse(read('../../lib/i18n/locales/en.json'))
    for (const k of Object.keys(zh.linkHealth)) {
      expect(en.linkHealth[k]).toBeTruthy()
    }
  })

  it('Edge 详情接入绑定提示与行动', () => {
    const panel = read('../edges/detail-panel.tsx')
    expect(panel).toContain('LinkHealthSection')
    expect(panel).toContain('LinkHealthAlert')
    expect(panel).toContain('edgeReferences')
    expect(panel).toContain('listCases')
  })

  it('Edge 详情 hook 顺序合规（edgeRefs useMemo 必须在 loading 早退之前）', () => {
    const panel = read('../edges/detail-panel.tsx')
    const memoAt = panel.indexOf('const edgeRefs = useMemo')
    const earlyReturnAt = panel.indexOf('if (detailQuery.isLoading')
    expect(memoAt).toBeGreaterThan(-1)
    expect(earlyReturnAt).toBeGreaterThan(-1)
    expect(memoAt).toBeLessThan(earlyReturnAt)
  })

  it('Topic 详情接入引用列表与行动', () => {
    const panel = read('../topics/topic-detail-panel.tsx')
    expect(panel).toContain('LinkHealthSection')
    expect(panel).toContain('LinkHealthAlert')
    expect(panel).toContain('topicReferences')
    expect(panel).toContain('listPresence')
  })

  it('Case 详情接入可达性提示与行动', () => {
    const section = read('../config-context/case-context-section.tsx')
    expect(section).toContain('LinkHealthSection')
    expect(section).toContain('LinkHealthAlert')
    expect(section).toContain('caseReferences')
    expect(section).toContain('linkHealth.title')
  })
})
