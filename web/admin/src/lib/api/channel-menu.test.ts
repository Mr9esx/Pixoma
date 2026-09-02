import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  type Action,
  getCaseMenuPlacements,
  getMenu,
  putMenu,
  type Menu,
} from './channel-menu'
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

  it('getMenu GETs the channel menu', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ id: 'm', name: '主', columns: 2, items: [] }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
    vi.stubGlobal('fetch', fetchMock)
    const menu = await getMenu('ch1')
    expect(menu.id).toBe('m')
    expect(menu.columns).toBe(2)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/channels/ch1/menu',
      expect.anything()
    )
  })
})


describe('Action schema (v2: workflow_id single, no mode)', () => {
  it('Action serializes workflow_id (single), drops workflow_ids/mode', () => {
    // 编译期：Action 接受 workflow_id 字段
    const action: Action = { type: 'open_workflow', workflow_id: '10' }
    const json = JSON.stringify(action)
    expect(json).toBe('{"type":"open_workflow","workflow_id":"10"}')
    expect(json).not.toContain('workflow_ids')
    expect(json).not.toContain('mode')
  })

  it('serializes open_workflow with workflow_id (single) and drops mode/workflow_ids', () => {
    const action: Action = { type: 'open_workflow', workflow_id: '10' }
    const json = JSON.stringify(action)
    expect(json).toBe('{"type":"open_workflow","workflow_id":"10"}')
    expect(json).not.toContain('workflow_ids')
    expect(json).not.toContain('mode')
  })

  it('Menu with open_workflow action PUTs workflow_id (not workflow_ids, no mode)', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'm', name: '主', columns: 2, items: [] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    const menu: Menu = {
      id: 'm',
      columns: 2,
      items: [
        {
          id: 'mi-1',
          label: '图片生成',
          action: { type: 'open_workflow', workflow_id: '10' },
        },
      ],
    }
    await putMenu('ch1', menu)

    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string)
    expect(body.items[0].action.workflow_id).toBe('10')
    expect(body.items[0].action).not.toHaveProperty('workflow_ids')
    expect(body.items[0].action).not.toHaveProperty('mode')
  })
})
