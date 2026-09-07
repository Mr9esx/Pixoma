import { afterEach, describe, expect, it, vi } from 'vitest'
import { kickChannelProbe } from './channels'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('kickChannelProbe', () => {
  it('POSTs /api/v1/channels/probe and does not wait on a body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 202 })
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(kickChannelProbe()).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/channels/probe',
      expect.objectContaining({ method: 'POST' })
    )
  })
})
