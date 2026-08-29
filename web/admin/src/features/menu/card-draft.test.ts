import { describe, expect, it } from 'vitest'
import { buildNewCardDraft, validateNewCardDraft } from './card-draft'

describe('validateNewCardDraft', () => {
  it('空 name 报错', () => {
    const result = validateNewCardDraft({ name: '', text: 'hi' })
    expect(result.ok).toBe(false)
    expect(result.errors).toContain('name')
  })

  it('只有空白字符的 name 报错', () => {
    const result = validateNewCardDraft({ name: '   ', text: 'hi' })
    expect(result.ok).toBe(false)
    expect(result.errors).toContain('name')
  })

  it('name 和 text 都没内容时报 name 错（text 可选）', () => {
    const result = validateNewCardDraft({ name: '', text: '' })
    expect(result.ok).toBe(false)
    expect(result.errors).toContain('name')
  })

  it('合法 name 通过验证', () => {
    const result = validateNewCardDraft({ name: '开始', text: '选风格' })
    expect(result.ok).toBe(true)
    expect(result.errors).toEqual([])
  })
})

describe('buildNewCardDraft', () => {
  it('构造 Card 草稿：trim name，生成 id', () => {
    const draft = buildNewCardDraft({ name: '  开始  ', text: 'hi' })
    expect(draft.id).toBeTruthy()
    expect(draft.name).toBe('开始')
    expect(draft.text).toBe('hi')
    expect(draft.media).toEqual([])
    expect(draft.buttons).toEqual([])
  })

  it('空 text 也合法（可后续编辑）', () => {
    const draft = buildNewCardDraft({ name: '开始', text: '' })
    expect(draft.text).toBe('')
  })

  it('id 唯一性（多次调用得到不同 id）', () => {
    const a = buildNewCardDraft({ name: 'a', text: '' })
    const b = buildNewCardDraft({ name: 'a', text: '' })
    expect(a.id).not.toBe(b.id)
  })

  it('validate + build 链路：先验证合法后 build 得到 Card', () => {
    const input = { name: '开始', text: 'hi' }
    const ok = validateNewCardDraft(input)
    expect(ok.ok).toBe(true)
    const card = buildNewCardDraft(input)
    expect(card.name).toBe('开始')
  })

  it('非法时 build 抛错', () => {
    expect(() => buildNewCardDraft({ name: '', text: '' })).toThrow()
  })
})
