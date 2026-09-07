import { describe, expect, it } from 'vitest'
import { formatDraftView } from './draft-value'
import { opsIdentity } from './identity'

describe('opsIdentity', () => {
  it('joins user and workflow for a scannable title', () => {
    expect(opsIdentity({ userLabel: 'alice', caseId: 12 })).toBe('alice · 12')
  })

  it('falls back to whichever side exists', () => {
    expect(opsIdentity({ userLabel: 'alice' })).toBe('alice')
    expect(opsIdentity({ caseId: 12 })).toBe('12')
    expect(opsIdentity({})).toBe('')
  })
})

describe('formatDraftView', () => {
  it('marks skipped entries separately from values', () => {
    expect(formatDraftView({ key: 'prompt', skipped: true })).toEqual({
      kind: 'skipped',
    })
  })

  it('prefers text, then number, then bool, then blob', () => {
    expect(formatDraftView({ key: 'a', text: 'hello' })).toEqual({
      kind: 'value',
      text: 'hello',
    })
    expect(formatDraftView({ key: 'b', number: 3 })).toEqual({
      kind: 'value',
      text: '3',
    })
    expect(formatDraftView({ key: 'c', bool: true })).toEqual({
      kind: 'value',
      text: 'true',
    })
    expect(formatDraftView({ key: 'd', blob: { w: 1 } })).toEqual({
      kind: 'value',
      text: '{"w":1}',
    })
  })
})
