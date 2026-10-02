import type { JSONContent } from '@tiptap/react'
import type { StudioComposerPart } from '@/lib/api/studio'

// 让引用后的光标停在文本位置，保持与普通文字相同的高度。
export const STUDIO_REFERENCE_CARET = '\u200B'
export const STUDIO_REFERENCE_BEFORE_CARET = '\u2060'

export type StudioComposerValue = {
  text: string
  parts: StudioComposerPart[]
  selectedSkillIds: string[]
  selectedAssets: { assetId: string; assetVersionId: string }[]
}

export function studioComposerContent(parts: StudioComposerPart[]): JSONContent {
  const content: JSONContent[] = []
  for (const part of parts) {
    if (part.type === 'text') {
      for (const [index, text] of part.text.split('\n').entries()) {
        if (index > 0) content.push({ type: 'hardBreak' })
        if (text) content.push({ type: 'text', text })
      }
      continue
    }
    const reference = part.type === 'skill_ref'
      ? { kind: 'skill', id: part.skill_id, label: part.name }
      : part.type === 'workflow_ref'
        ? { kind: 'workflow', id: part.workflow_id, label: part.name }
        : { kind: 'asset', id: part.asset_id, label: part.name, versionId: part.asset_version_id }
    content.push({ type: 'text', text: STUDIO_REFERENCE_BEFORE_CARET })
    content.push({ type: 'studioReference', attrs: reference })
    content.push({ type: 'text', text: STUDIO_REFERENCE_CARET })
  }
  return { type: 'doc', content: [{ type: 'paragraph', content }] }
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
        const text = (node.text ?? '').replaceAll(STUDIO_REFERENCE_BEFORE_CARET, '')
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
      } else if (kind === 'workflow') {
        parts.push({ type: 'workflow_ref', workflow_id: id, name: label })
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
      if (part.type === 'skill_ref') return `「${part.name}」技能`
      if (part.type === 'workflow_ref') return `「${part.name}」工作流`
      return `「${part.name}」资产`
    }).join(''),
    parts,
    selectedSkillIds,
    selectedAssets,
  }
}
