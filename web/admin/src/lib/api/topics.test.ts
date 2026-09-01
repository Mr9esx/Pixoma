import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createTopic,
  deleteTopic,
  getTopicStats,
  getTopic,
  listTopics,
  updateTopic,
} from './topics'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('topics API', () => {
  it('listTopics GETs /api/v1/topics', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ key: 'fast-gpu', name: 'Fast GPU', enabled: true }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTopics()

    expect(data[0].key).toBe('fast-gpu')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/topics',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
  })

  it('listTopics passes enabled filter', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await listTopics(true)

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/topics?enabled=true',
      expect.anything(),
    )
  })

  it('createTopic POSTs key/name JSON body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ key: 'fast-gpu', name: 'Fast GPU', enabled: true }),
        { status: 201, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await createTopic({ key: 'fast-gpu', name: 'Fast GPU' })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8081/api/v1/topics')
    expect((init as RequestInit).method).toBe('POST')
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      key: 'fast-gpu',
      name: 'Fast GPU',
    })
  })

  it('updateTopic PUTs name/enabled to keyed route', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ key: 'fast-gpu', name: 'Fast GPU', enabled: false }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await updateTopic('fast-gpu', { enabled: false })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8081/api/v1/topics/fast-gpu')
    expect((init as RequestInit).method).toBe('PUT')
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      enabled: false,
    })
  })

  it('getTopic/deleteTopic hit the keyed route', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await getTopic('fast-gpu')
    await deleteTopic('fast-gpu')

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/topics/fast-gpu',
    )
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://127.0.0.1:8081/api/v1/topics/fast-gpu',
    )
    expect((fetchMock.mock.calls[1][1] as RequestInit).method).toBe('DELETE')
  })

  it('getTopicStats GETs /api/v1/topics/:key/stats', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          task_count: 3,
          status: { succeeded: 2, failed: 1 },
          success_rate: 0.66,
          error_codes: [{ code: 'timeout', count: 1 }],
          runtime_ms: { sum_ms: 35000, avg_ms: 11666, count: 3 },
          throughput: [{ ts: '2026-08-21T00:00:00Z', count: 3 }],
          from: '2026-08-21T00:00:00Z',
          to: '2026-08-22T00:00:00Z',
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await getTopicStats('fast-gpu', {
      from: '2026-08-15',
      to: '2026-08-21',
    })

    expect(data.task_count).toBe(3)
    expect(data.error_codes[0].code).toBe('timeout')
    const url = new URL(fetchMock.mock.calls[0][0] as string)
    expect(url.pathname).toBe('/api/v1/topics/fast-gpu/stats')
    expect(url.searchParams.get('from')).toBe(
      new Date('2026-08-15T00:00:00').toISOString()
    )
    expect(url.searchParams.get('to')).toBe(
      new Date('2026-08-21T23:59:59.999').toISOString()
    )
  })
})
