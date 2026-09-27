import { describe, expect, it } from 'vitest'
import { serializeStudioComposer } from './studio-composer-content'

describe('serializeStudioComposer', () => {
  it('omits the caret spacer after an inline reference', () => {
    expect(serializeStudioComposer({
      type: 'doc',
      content: [{
        type: 'paragraph',
        content: [
          { type: 'studioReference', attrs: { kind: 'skill', id: 'skill-1', label: '分镜草稿' } },
          { type: 'text', text: '\u200B后续内容' },
        ],
      }],
    })).toEqual({
      text: '「分镜草稿」技能后续内容',
      parts: [
        { type: 'skill_ref', skill_id: 'skill-1', name: '分镜草稿' },
        { type: 'text', text: '后续内容' },
      ],
      selectedSkillIds: ['skill-1'],
      selectedAssets: [],
    })
  })

  it('keeps a typed slash as ordinary text', () => {
    expect(
      serializeStudioComposer({
        type: 'doc',
        content: [{ type: 'paragraph', content: [{ type: 'text', text: '/skill' }] }],
      })
    ).toEqual({
      text: '/skill',
      parts: [{ type: 'text', text: '/skill' }],
      selectedSkillIds: [],
      selectedAssets: [],
    })
  })

  it('preserves a zero-width space typed outside a reference', () => {
    expect(serializeStudioComposer({
      type: 'doc',
      content: [{ type: 'paragraph', content: [{ type: 'text', text: '甲\u200B乙' }] }],
    }).text).toBe('甲\u200B乙')
  })

  it('keeps a workflow reference in the message without submitting its inputs', () => {
    expect(serializeStudioComposer({
      type: 'doc',
      content: [{ type: 'paragraph', content: [
        { type: 'studioReference', attrs: { kind: 'workflow', id: '12', label: '角色三视图' } },
        { type: 'text', text: '\u200B需要哪些输入？' },
      ] }],
    })).toEqual({
      text: '「角色三视图」工作流需要哪些输入？',
      parts: [
        { type: 'workflow_ref', workflow_id: '12', name: '角色三视图' },
        { type: 'text', text: '需要哪些输入？' },
      ],
      selectedSkillIds: [],
      selectedAssets: [],
    })
  })

  it('preserves inline reference order and selects each resource once', () => {
    expect(
      serializeStudioComposer({
        type: 'doc',
        content: [
          {
            type: 'paragraph',
            content: [
              { type: 'text', text: '用 ' },
              { type: 'studioReference', attrs: { kind: 'skill', id: 'skill-1', label: '分镜草稿' } },
              { type: 'text', text: ' 处理 ' },
              { type: 'studioReference', attrs: { kind: 'asset', id: 'asset-1', versionId: 'version-2', label: '角色设定图' } },
              { type: 'text', text: '，再用 ' },
              { type: 'studioReference', attrs: { kind: 'skill', id: 'skill-1', label: '分镜草稿' } },
            ],
          },
        ],
      })
    ).toEqual({
      text: '用 「分镜草稿」技能 处理 「角色设定图」资产，再用 「分镜草稿」技能',
      parts: [
        { type: 'text', text: '用 ' },
        { type: 'skill_ref', skill_id: 'skill-1', name: '分镜草稿' },
        { type: 'text', text: ' 处理 ' },
        { type: 'asset_ref', asset_id: 'asset-1', asset_version_id: 'version-2', name: '角色设定图' },
        { type: 'text', text: '，再用 ' },
        { type: 'skill_ref', skill_id: 'skill-1', name: '分镜草稿' },
      ],
      selectedSkillIds: ['skill-1'],
      selectedAssets: [{ assetId: 'asset-1', assetVersionId: 'version-2' }],
    })
  })
})
