import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createStudioModel,
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
} from './studio'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Studio API', () => {
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

    const result = await createStudioSession()

    expect(result.id).toBe('session-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/studio/sessions',
      expect.objectContaining({ method: 'POST' })
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
      model_config_id: 'model-1',
      permission_mode: 'request_approval',
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
      enabled: false,
    })
    await updateStudioConnector({
      id: 'connector-1',
      name: 'Reference',
      url: 'https://mcp.example.com',
      enabled: false,
      policy: 'forbidden',
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
