import type { JSONContent } from '@tiptap/react'
import type { StudioComposerPart } from '@/lib/api/studio'

// 让引用后的光标停在文本位置，保持与普通文字相同的高度。
export const STUDIO_REFERENCE_CARET = '\u200B'

export type StudioComposerValue = {
  text: string
  parts: StudioComposerPart[]
  selectedSkillIds: string[]
  selectedAssets: { assetId: string; assetVersionId: string }[]
}

export function serializeStudioComposer(content: JSONContent): StudioComposerValue {
  const parts: StudioComposerPart[] = []
  const selectedSkillIds: string[] = []
  const selectedAssets: StudioComposerValue['selectedAssets'] = []
  const appendText = (text: string) => {
    if (!text) return
    const last = parts[parts.length - 1]
    if (last?.type === 'text') last.text += text
    else parts.push({ type: 'text', text })
  }

  for (const [paragraphIndex, paragraph] of (content.content ?? []).entries()) {
    if (paragraph.type !== 'paragraph') throw new Error('Unsupported composer content')
    if (paragraphIndex > 0) appendText('\n')
    let afterReference = false
    for (const node of paragraph.content ?? []) {
      if (node.type === 'text') {
        const text = node.text ?? ''
        appendText(afterReference ? text.replace(STUDIO_REFERENCE_CARET, '') : text)
        afterReference = false
        continue
      }
      if (node.type === 'hardBreak') {
        appendText('\n')
        afterReference = false
        continue
      }
      if (node.type !== 'studioReference') throw new Error('Unsupported composer node')
      const { kind, id, label, versionId } = node.attrs ?? {}
      if (typeof id !== 'string' || !id || typeof label !== 'string' || !label) {
        throw new Error('Invalid composer reference')
      }
      if (kind === 'skill') {
        parts.push({ type: 'skill_ref', skill_id: id, name: label })
        if (!selectedSkillIds.includes(id)) selectedSkillIds.push(id)
      } else if (kind === 'asset' && typeof versionId === 'string' && versionId) {
        parts.push({ type: 'asset_ref', asset_id: id, asset_version_id: versionId, name: label })
        if (!selectedAssets.some((asset) => asset.assetId === id)) {
          selectedAssets.push({ assetId: id, assetVersionId: versionId })
        }
      } else {
        throw new Error('Invalid composer reference')
      }
      afterReference = true
    }
  }

  return {
    text: parts.map((part) => {
      if (part.type === 'text') return part.text
      if (part.type === 'skill_ref') return `「${part.name}」Skill`
      return `「${part.name}」资产`
    }).join(''),
    parts,
    selectedSkillIds,
    selectedAssets,
  }
}
