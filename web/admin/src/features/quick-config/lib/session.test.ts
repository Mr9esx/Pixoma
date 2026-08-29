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
  caseId: 12,
  mode: 'existing',
  step: 1,
  caseDraft: { id: 12, name: '动漫图像生成' },
  routing: { rules: [{ when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' }] },
  pendingEntries: [{ channelId: 'ch-a', label: '开始生成' }],
  schemaVersion: 2,
  updatedAt: '2026-08-21T10:00:00.000Z',
}

describe('saveQuickConfigSession / loadQuickConfigSession', () => {
  it('保存后可完整读回', () => {
    const storage = fakeStorage()
    saveQuickConfigSession(storage, session)
    expect(loadQuickConfigSession(storage)).toEqual(session)
    expect(storage.getItem(SESSION_KEY)).toContain('"caseId":12')
  })

  it('旧版会话缺少草稿字段时也能读回并给默认值', () => {
    const storage = fakeStorage()
    storage.setItem(
      SESSION_KEY,
      JSON.stringify({
        caseId: 7,
        mode: 'create',
        step: 0,
        schemaVersion: 2,
        updatedAt: '2026-08-21T09:00:00.000Z',
      })
    )
    expect(loadQuickConfigSession(storage)).toEqual({
      caseId: 7,
      mode: 'create',
      step: 0,
      caseDraft: null,
      routing: undefined,
      pendingEntries: [],
      schemaVersion: 2,
      updatedAt: '2026-08-21T09:00:00.000Z',
    })
  })

  it('新建草稿会话（caseId 为 null）可保存读回', () => {
    const storage = fakeStorage()
    const draftSession: QuickConfigSession = {
      caseId: null,
      mode: 'create',
      step: 1,
      caseDraft: { id: 0, name: '未命名工作流' },
      routing: undefined,
      pendingEntries: [],
      schemaVersion: 2,
      updatedAt: '2026-08-21T11:00:00.000Z',
    }
    saveQuickConfigSession(storage, draftSession)
    expect(loadQuickConfigSession(storage)).toEqual(draftSession)
  })

  it('无会话时返回 null', () => {
    expect(loadQuickConfigSession(fakeStorage())).toBeNull()
  })

  it('数据损坏时返回 null 而不是抛错', () => {
    const storage = fakeStorage()
    storage.setItem(SESSION_KEY, '{not-json')
    expect(loadQuickConfigSession(storage)).toBeNull()
  })

  it('结构不合法时返回 null', () => {
    const storage = fakeStorage()
    storage.setItem(SESSION_KEY, JSON.stringify({ caseId: 'x' }))
    expect(loadQuickConfigSession(storage)).toBeNull()
  })

  it('无 schemaVersion 的旧版会话被清空并返回 null', () => {
    const storage = fakeStorage()
    storage.setItem(
      SESSION_KEY,
      JSON.stringify({
        caseId: 7,
        mode: 'create',
        step: 1,
        updatedAt: '2026-08-21T09:00:00.000Z',
      })
    )
    expect(loadQuickConfigSession(storage)).toBeNull()
    expect(storage.getItem(SESSION_KEY)).toBeNull()
  })

  it('保存的会话带有 schemaVersion 2 并可读回', () => {
    const storage = fakeStorage()
    saveQuickConfigSession(storage, session)
    expect(loadQuickConfigSession(storage)).toEqual(session)
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
