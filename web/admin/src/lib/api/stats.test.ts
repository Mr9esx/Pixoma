import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  listTaskDailyStats,
  listTaskEdgeStats,
  listTaskErrorStats,
} from './stats'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('stats API', () => {
  it('listTaskDailyStats GETs daily endpoint with from/to', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          range: { from: '2026-08-01', to: '2026-08-02' },
          days: [],
          summary: { success_rate: null },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTaskDailyStats({
      from: '2026-08-01',
      to: '2026-08-02',
    })

    expect(data.range.from).toBe('2026-08-01')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/stats/tasks/daily?from=2026-08-01&to=2026-08-02',
      expect.anything()
    )
  })

  it('listTaskErrorStats passes limit', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await listTaskErrorStats({ from: '2026-08-01', to: '2026-08-02', limit: 5 })

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/stats/tasks/errors?from=2026-08-01&to=2026-08-02&limit=5',
      expect.anything()
    )
  })

  it('listTaskEdgeStats returns items and total', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [], total: 0 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTaskEdgeStats({
      from: '2026-08-01',
      to: '2026-08-02',
    })

    expect(data.total).toBe(0)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/stats/tasks/edges?from=2026-08-01&to=2026-08-02',
      expect.anything()
    )
  })
})
