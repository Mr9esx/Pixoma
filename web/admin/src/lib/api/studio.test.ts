import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createStudioSession,
  getStudioSession,
  listStudioSessions,
  sendStudioMessage,
} from './studio'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Studio API', () => {
  it('lists Studio sessions with pagination', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'session-1', title: '雨夜侦探' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await listStudioSessions({ limit: 30, offset: 0 })

    expect(result[0].id).toBe('session-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/studio/sessions?limit=30&offset=0',
      expect.anything()
    )
  })

  it('creates a session before mounting the AG-UI runtime', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'session-1', title: '新对话' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await createStudioSession()

    expect(result.id).toBe('session-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/studio/sessions',
      expect.objectContaining({ method: 'POST' })
    )
  })

  it('loads a complete session snapshot', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          session: { id: 'session-1' },
          messages: [],
          assets: [],
          flow: { nodes: [], edges: [] },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await getStudioSession('session-1')

    expect(result.flow.nodes).toEqual([])
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/studio/sessions/session-1',
      expect.anything()
    )
  })

  it('sends composer model and permission selections with the message', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          session: { id: 'session-1' },
          message: { id: 'message-1' },
          run: { id: 'run-1' },
        }),
        { status: 202, headers: { 'Content-Type': 'application/json' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    await sendStudioMessage({
      sessionId: 'session-1',
      text: '生成分镜',
      modelConfigId: 'model-1',
      permissionMode: 'request_approval',
    })

    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/messages'
    )
    expect(JSON.parse(String(init.body))).toEqual({
      session_id: 'session-1',
      text: '生成分镜',
      model_config_id: 'model-1',
      permission_mode: 'request_approval',
    })
  })
})

