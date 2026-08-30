import { describe, expect, it } from 'vitest'
import {
  clearQuickConfigSession,
  loadQuickConfigSession,
  SESSION_KEY,
  saveQuickConfigSession,
  type QuickConfigSession,
} from './session'

function fakeStorage(): Storage {
  const map = new Map<string, string>()
  return {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => void map.set(k, v),
    removeItem: (k: string) => void map.delete(k),
    clear: () => map.clear(),
    key: (i: number) => [...map.keys()][i] ?? null,
    get length() {
      return map.size
    },
  }
}

const session: QuickConfigSession = {
  caseId: null,
  step: 2,
  caseDraft: { id: 0, name: 'demo' },
  topicKey: 'default',
  topicDraft: null,
  selectedEdgeId: 'gpu-1',
  schemaVersion: 3,
  updatedAt: '2026-08-30T10:00:00.000Z',
}

describe('saveQuickConfigSession / loadQuickConfigSession', () => {
  it('保存后可完整读回 v3 草稿', () => {
    const storage = fakeStorage()
    saveQuickConfigSession(storage, session)
    expect(loadQuickConfigSession(storage)).toEqual(session)
  })

  it('schemaVersion 2 的旧会话被清空并返回 null', () => {
    const storage = fakeStorage()
    storage.setItem(
      SESSION_KEY,
      JSON.stringify({
        caseId: 12,
        mode: 'existing',
        step: 1,
        schemaVersion: 2,
        updatedAt: '2026-08-21T10:00:00.000Z',
      })
    )
    expect(loadQuickConfigSession(storage)).toBeNull()
    expect(storage.getItem(SESSION_KEY)).toBeNull()
  })

  it('无会话时返回 null', () => {
    expect(loadQuickConfigSession(fakeStorage())).toBeNull()
  })

  it('数据损坏时返回 null 而不是抛错', () => {
    const storage = fakeStorage()
    storage.setItem(SESSION_KEY, '{not-json')
    expect(loadQuickConfigSession(storage)).toBeNull()
  })
})

describe('clearQuickConfigSession', () => {
  it('清除会话键', () => {
    const storage = fakeStorage()
    saveQuickConfigSession(storage, session)
    clearQuickConfigSession(storage)
    expect(storage.getItem(SESSION_KEY)).toBeNull()
  })
})
