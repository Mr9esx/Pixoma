import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createStudioModel,
  testStudioModelConfig,
  updateStudioModel,
  createStudioConnector,
  createStudioSession,
  createStudioSkill,
  getStudioSession,
  listStudioSkills,
  listStudioConnectors,
  listStudioAgentWorkflows,
  updateStudioAgentWorkflow,
  updateStudioConnector,
  updateStudioSkill,
  listStudioSessions,
  sendStudioMessage,
  studioLibraryAssetsPath,
  studioLibraryDateBoundary,
  studioLibraryTreePath,
  studioLibraryGroupQuery,
  referenceStudioAsset,
  updateStudioTextAsset,
} from './studio'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Studio API', () => {
  it('显式传递未归属项目，并组合资产筛选条件', () => {
    expect(
      studioLibraryAssetsPath({
        projectId: '',
        search: '角色',
        kind: 'image',
        format: 'png',
        categoryId: 'category-1',
        sessionId: 'session-1',
        rating: 4,
        tagIds: ['tag-1', 'tag-2'],
        cursor: 'next',
        limit: 50,
      })
    ).toBe(
      '/api/v1/studio/library/assets?project_id=&q=%E8%A7%92%E8%89%B2&kind=image&format=png&category_id=category-1&session_id=session-1&rating=4&tag_ids=tag-1%2Ctag-2&limit=50&cursor=next'
    )
  })

  it('按项目和组织方式读取文件树', () => {
    expect(
      studioLibraryTreePath({ projectId: 'project-1', mode: 'session' })
    ).toBe('/api/v1/studio/library/tree?project_id=project-1&mode=session')
    expect(
      studioLibraryTreePath({
        projectId: 'project-1',
        mode: 'category',
        parentId: 'category-1',
      })
    ).toBe(
      '/api/v1/studio/library/tree?project_id=project-1&mode=category&parent_id=category-1'
    )
  })

  it('组合尺寸、大小、添加日期和重复文件筛选', () => {
    expect(studioLibraryAssetsPath({
      projectId: 'project-1',
      widthMin: 100,
      widthMax: 2000,
      heightMin: 100,
      heightMax: 2000,
      sizeMin: 1024,
      sizeMax: 1048576,
      addedFrom: '2026-09-01T16:00:00.000Z',
      addedTo: '2026-09-30T16:00:00.000Z',
      duplicates: true,
    })).toBe('/api/v1/studio/library/assets?project_id=project-1&width_min=100&width_max=2000&height_min=100&height_max=2000&size_min=1024&size_max=1048576&added_from=2026-09-01T16%3A00%3A00.000Z&added_to=2026-09-30T16%3A00%3A00.000Z&duplicates=true&limit=50')
  })

  it('按本地日期换算添加日期范围的两端', () => {
    expect(studioLibraryDateBoundary('2026-09-28')).toBe(new Date(2026, 8, 28).toISOString())
    expect(studioLibraryDateBoundary('2026-09-28', true)).toBe(new Date(2026, 8, 29).toISOString())
  })

  it('把文件树分组转换为资产筛选条件', () => {
    expect(studioLibraryGroupQuery('session', 'session-1')).toEqual({
      sessionId: 'session-1',
    })
    expect(studioLibraryGroupQuery('format', 'png')).toEqual({
      format: 'png',
    })
    expect(studioLibraryGroupQuery('tag', 'tag-1')).toEqual({
      tagIds: ['tag-1'],
    })
    expect(studioLibraryGroupQuery('rating', '0')).toEqual({ rating: 0 })
  })

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

    const requestId = 'ed4760ca-7c62-4ca2-9f7f-e1b760265f10'
    const result = await createStudioSession(requestId)

    expect(result.id).toBe('session-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/studio/sessions',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ request_id: requestId }),
      })
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
      locale: 'en',
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
      locale: 'en',
      model_config_id: 'model-1',
      permission_mode: 'request_approval',
    })
  })

  it('updates a text asset by appending a new immutable version', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'asset-1', current_version: 2 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await updateStudioTextAsset('asset-1', '# 雨夜侦探\n补充旧案线索。')

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/assets/asset-1/text'
    )
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(String(init.body))).toMatchObject({
      content: '# 雨夜侦探\n补充旧案线索。',
    })
    expect(JSON.parse(String(init.body)).request_id).toEqual(expect.any(String))
  })

  it('records a selected asset version in the active session', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'asset-imported' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await referenceStudioAsset(
      'session-1',
      'asset-library',
      'version-2',
      'request-1',
      'project-asset-1'
    )

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/sessions/session-1/assets/references'
    )
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      asset_id: 'asset-library',
      asset_version_id: 'version-2',
      request_id: 'request-1',
      source_project_asset_id: 'project-asset-1',
    })
  })

  it('creates an encrypted server-side model configuration', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'model-1', name: 'Claude' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await createStudioModel({
      name: 'Claude',
      protocol: 'anthropic_messages_compatible',
      baseUrl: 'https://api.anthropic.com/v1',
      model: 'claude-sonnet',
      apiKey: 'only-in-request',
      enabled: true,
      agentEnabled: true,
      default: false,
      thinking: { enabled: true, budget_tokens: 2048 },
      capabilities: {
        tools: true,
        vision: false,
        image_output: false,
        streaming: true,
      },
    })

    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/models'
    )
    expect(JSON.parse(String(init.body))).toMatchObject({
      protocol: 'anthropic_messages_compatible',
      api_key: 'only-in-request',
      agent_enabled: true,
    })
  })

  it('tests an unsaved model configuration without using a model id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ success: true, latency_ms: 42 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await testStudioModelConfig({
      name: 'Claude',
      protocol: 'anthropic_messages_compatible',
      baseUrl: 'https://api.anthropic.com/v1',
      model: 'claude-sonnet',
      apiKey: 'only-in-request',
      enabled: true,
      agentEnabled: true,
      default: false,
      thinking: { enabled: false },
      capabilities: {
        tools: true,
        vision: false,
        image_output: false,
        streaming: true,
      },
    })

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/models/test'
    )
    expect(
      JSON.parse(String((fetchMock.mock.calls[0][1] as RequestInit).body))
    ).toMatchObject({
      model: 'claude-sonnet',
      api_key: 'only-in-request',
    })
  })

  it('can test edited fields with the saved model key without sending that key', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ success: true, latency_ms: 12 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await testStudioModelConfig({
      name: 'Edited',
      protocol: 'openai_chat_compatible',
      baseUrl: 'https://api.example.com/v1',
      model: 'model-edited',
      apiKey: '',
      existingModelId: 'model-1',
      enabled: true,
      agentEnabled: true,
      default: false,
      thinking: { enabled: false },
      capabilities: {
        tools: true,
        vision: false,
        image_output: false,
        streaming: true,
      },
    })

    const body = JSON.parse(
      String((fetchMock.mock.calls[0][1] as RequestInit).body)
    )
    expect(body.existing_model_id).toBe('model-1')
    expect(body).not.toHaveProperty('api_key')
  })

  it('updates a saved model without requiring the existing API key', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'model-1', name: 'Updated' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await updateStudioModel('model-1', {
      name: 'Updated',
      protocol: 'openai_chat_compatible',
      baseUrl: 'https://api.example.com/v1',
      model: 'model-updated',
      apiKey: '',
      enabled: true,
      agentEnabled: true,
      default: false,
      thinking: { enabled: false },
      capabilities: {
        tools: true,
        vision: false,
        image_output: false,
        streaming: true,
      },
    })

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/models/model-1'
    )
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe('PATCH')
  })

  it('lists and creates account-scoped Studio Skills', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify([{ id: 'skill-1', name: '漫画分镜', enabled: true }]),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: 'skill-2', name: '角色设定' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    vi.stubGlobal('fetch', fetchMock)

    const skills = await listStudioSkills()
    await createStudioSkill({
      name: '角色设定',
      description: '保持角色一致',
      prompt: '锁定角色特征',
      enabled: true,
    })

    expect(skills[0].id).toBe('skill-1')
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/skills'
    )
    expect(JSON.parse(String(fetchMock.mock.calls[1][1].body))).toMatchObject({
      name: '角色设定',
      enabled: true,
    })
  })

  it('lists and creates Studio MCP connectors without retaining browser-side credentials', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            { id: 'connector-1', name: 'Reference', enabled: true },
          ]),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ id: 'connector-2', credential_masked: '••••••••' }),
          {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          }
        )
      )
    vi.stubGlobal('fetch', fetchMock)

    const connectors = await listStudioConnectors()
    await createStudioConnector({
      name: 'Reference',
      url: 'https://mcp.example.com',
      credential: 'request-only-secret',
      enabled: true,
      policy: 'approval',
    })

    expect(connectors[0].id).toBe('connector-1')
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/connectors'
    )
    expect(JSON.parse(String(fetchMock.mock.calls[1][1].body))).toMatchObject({
      url: 'https://mcp.example.com',
      credential: 'request-only-secret',
      policy: 'approval',
    })
  })

  it('lists existing workflows and saves only their Agent availability', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            { id: '12', name: '角色三视图', agent_enabled: false },
          ]),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: '12', agent_enabled: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    vi.stubGlobal('fetch', fetchMock)

    const workflows = await listStudioAgentWorkflows()
    await updateStudioAgentWorkflow('12', true)

    expect(workflows[0].id).toBe('12')
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/workflows'
    )
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/workflows/12'
    )
    expect(JSON.parse(String(fetchMock.mock.calls[1][1].body))).toEqual({
      agent_enabled: true,
    })
  })

  it('updates Skill and connector availability without resending a connector credential', async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ id: 'capability-1', enabled: false }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    await updateStudioSkill({
      id: 'skill-1',
      name: '分镜',
      description: '',
      prompt: '输出镜头表',
      version: '1.0.1',
      enabled: false,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-02T00:00:00Z',
    })
    await updateStudioConnector({
      id: 'connector-1',
      name: 'Reference',
      url: 'https://mcp.example.com',
      enabled: false,
      policy: 'forbidden',
    })

    expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toMatchObject({
      version: '1.0.1',
      updated_at: '2026-01-02T00:00:00Z',
    })
    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/skills/skill-1'
    )
    expect(fetchMock.mock.calls[1][0]).toBe(
      'http://127.0.0.1:8081/api/v1/studio/connectors/connector-1'
    )
    expect(
      JSON.parse(String(fetchMock.mock.calls[1][1].body))
    ).not.toHaveProperty('credential')
  })
})
