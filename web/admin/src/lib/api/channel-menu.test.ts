import { describe, it, expect, vi, afterEach } from 'vitest'
import { getCaseMenuPlacements } from './channel-menu'
import { listCapabilities } from './channels'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('channel-menu API', () => {
  it('getCaseMenuPlacements GETs /api/v1/cases/{id}/menu-placements', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify([
          {
            channel_id: 'tg-default',
            item_id: 'btn-image',
            path: [{ id: 'folder-1', label: '图片' }],
          },
        ]),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await getCaseMenuPlacements('c1')

    expect(data[0].item_id).toBe('btn-image')
    expect(data[0].path[0].label).toBe('图片')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/cases/c1/menu-placements',
      expect.anything()
    )
  })

  it('capabilities carry params_schema', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify([
          {
            id: 'open_case',
            display_name: '打开工作流',
            params_schema: {
              type: 'object',
              properties: {
                case_ids: { type: 'array', items: { type: 'string' } },
              },
            },
          },
        ]),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const caps = await listCapabilities()

    expect(caps[0].id).toBe('open_case')
    expect(caps[0].display_name).toBe('打开工作流')
    expect(
      (caps[0].params_schema.properties as Record<string, { type: string }>)
        .case_ids.type
    ).toBe('array')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/channels/capabilities',
      expect.anything()
    )
  })
})
