import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('config chain association', () => {
  it('ConfigChain 提供整链健康、环节高亮、结论与去配置', () => {
    const chain = read('config-chain.tsx')
    expect(chain).toContain("data-testid='config-chain'")
    expect(chain).toContain('ChainHop')
    expect(chain).toContain('ChainDetail')
    expect(chain).toContain('configChain.goConfig')
    expect(chain).toContain('configChain.blockedHops')
    expect(chain).toContain('<Link')
  })

  it('Case 详情处理流程编辑器保留；ConfigChain 已由状态与关联取代', () => {
    const section = read('case-context-section.tsx')
    expect(section).toContain('TaskFlowEditor')
    expect(section).toContain('patchCase')
    expect(section).not.toContain('ConfigChain')
    expect(section).toContain('LinkHealthSection')
    const panel = read('../cases/detail-panel.tsx')
    expect(panel).toContain('CaseContextSection')
  })

  it('Topic 详情接入链路视图', () => {
    const panel = read('../topics/topic-detail-panel.tsx')
    expect(panel).toContain('ConfigChain')
    expect(panel).toContain('listCases')
    expect(panel).toContain('configChain.topicBlocked')
  })

  it('i18n 成对', () => {
    const zh = JSON.parse(read('../../lib/i18n/locales/zh.json'))
    const en = JSON.parse(read('../../lib/i18n/locales/en.json'))
    for (const k of Object.keys(zh.configChain)) {
      expect(en.configChain[k]).toBeTruthy()
    }
  })
})
