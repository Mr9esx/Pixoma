import { describe, it, expect, vi, afterEach } from 'vitest'
import { listCases } from './cases'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('cases API', () => {
  it('listCases GETs /api/v1/cases with query', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'c1', enabled: true }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listCases({ q: 'alpha', enabled: true, limit: 10 })

    expect(data[0].id).toBe('c1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/cases?q=alpha&enabled=true&limit=10',
      expect.anything()
    )
  })
})
