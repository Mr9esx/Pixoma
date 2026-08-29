import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  listTextTemplates,
  saveTextTemplates,
  resetTextTemplates,
} from './text-templates'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('text templates API', () => {
  it('listTextTemplates GETs the platform default without a channelId', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ key: 'welcome' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTextTemplates()

    expect(data).toHaveLength(1)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/text-templates/',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      })
    )
  })

  it('listTextTemplates scopes to /channels/{id} when a channelId is given', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await listTextTemplates('ch-9')

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/channels/ch-9/text-templates',
      expect.anything()
    )
  })

  it('saveTextTemplates PUTs the drafts', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ key: 'workflow_done' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const templates = { welcome: 'Hi {{ task_id }}' }
    const data = await saveTextTemplates('', templates)

    expect(data).toHaveLength(1)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/text-templates/',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ templates }),
      })
    )
  })

  it('resetTextTemplates POSTs the keys to the reset endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await resetTextTemplates('ch-9', ['welcome'])

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/channels/ch-9/text-templates/reset',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ keys: ['welcome'] }),
      })
    )
  })
})
