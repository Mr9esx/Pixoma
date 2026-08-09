import { describe, it, expect, vi, afterEach } from 'vitest'
import { listTasks, cancelTask } from './tasks'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('tasks API', () => {
  it('listTasks GETs /api/v1/tasks with status filter', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 't1', status: 'pending' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTasks({ status: 'pending', limit: 20 })

    expect(data[0].id).toBe('t1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/tasks?status=pending&limit=20',
      expect.anything(),
    )
  })

  it('cancelTask POSTs /api/v1/tasks/{id}/cancel', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 't1', status: 'cancelled' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await cancelTask('t1')

    expect(data.status).toBe('cancelled')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/tasks/t1/cancel',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})
