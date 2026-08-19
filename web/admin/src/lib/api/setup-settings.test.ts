import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  changeAdminPassword,
  fetchPlatformSettings,
  savePlatformSettings,
} from './setup'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('platform settings API', () => {
  it('fetchPlatformSettings GETs /api/v1/setup/settings', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ configured: true, settings: { placement: 'local' } }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await fetchPlatformSettings()

    expect(data.configured).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/setup/settings',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      })
    )
  })

  it('savePlatformSettings PUTs the draft', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true, restarting: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const draft = {
      placement: 'local' as const,
      db_driver: 'sqlite',
      db_dsn: 'data/app.db',
      blob_driver: 'localfs',
      blob_root: 'data/blob',
      comfy_mock: false,
      comfyui_base_url: 'http://127.0.0.1:8188',
    }
    const data = await savePlatformSettings(draft)

    expect(data.restarting).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/setup/settings',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(draft),
      })
    )
  })

  it('changeAdminPassword sends old_password after init', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await changeAdminPassword({
      oldPassword: 'current-secret',
      newPassword: 'later-secret-1',
    })

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/setup/password',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          new_password: 'later-secret-1',
          old_password: 'current-secret',
        }),
      })
    )
  })

  it('changeAdminPassword omits old_password during first-time set', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await changeAdminPassword({ newPassword: 'new-secret-9' })

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/setup/password',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          new_password: 'new-secret-9',
        }),
      })
    )
  })
})
