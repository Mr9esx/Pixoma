import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('config context association', () => {
  it('ContextLinks 提供分组引用、就绪徽标与跳转', () => {
    const links = read('context-links.tsx')
    expect(links).toContain("data-testid='context-links'")
    expect(links).toContain('ContextGroup')
    expect(links).toContain('configContext.ready')
    expect(links).toContain('configContext.warn')
    expect(links).toContain('<Link')
  })

  it('Case 详情嵌入处理流程编辑器并保存 routing', () => {
    const section = read('case-context-section.tsx')
    expect(section).toContain('TaskFlowEditor')
    expect(section).toContain('patchCase')
    expect(section).toContain('ContextLinks')
    const panel = read('../cases/detail-panel.tsx')
    expect(panel).toContain('CaseContextSection')
    expect(panel).toContain('sectionProcessing')
  })

  it('Topic 详情展示关联上下文', () => {
    const panel = read('../topics/topic-detail-panel.tsx')
    expect(panel).toContain('ContextLinks')
    expect(panel).toContain('listCases')
    expect(panel).toContain('configContext.relatedCases')
  })

  it('i18n 成对', () => {
    const zh = JSON.parse(read('../../lib/i18n/locales/zh.json'))
    const en = JSON.parse(read('../../lib/i18n/locales/en.json'))
    for (const k of Object.keys(zh.configContext)) {
      expect(en.configContext[k]).toBeTruthy()
    }
  })
})
