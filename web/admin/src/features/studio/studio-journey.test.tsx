import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import {
  getStudioSession,
  listStudioModels,
  listStudioSessions,
  type StudioAsset,
  type StudioFlowNode,
  type StudioSessionDetail,
} from '@/lib/api/studio'
import { StudioWorkspace } from './studio-workspace'

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  getStudioSession: vi.fn(),
  listStudioSessions: vi.fn(),
  listStudioModels: vi.fn(),
  listStudioSkills: vi.fn(async () => []),
  listStudioLibraryFolders: vi.fn(async () => []),
  listStudioLibraryAssets: vi.fn(async () => []),
}))

vi.mock('./studio-sidebar', () => ({
  StudioSidebar: () => <nav aria-label='Studio 导航' />,
}))

const timestamp = '2026-09-24T10:00:00Z'
const session = {
  id: 'session-comic',
  title: '雨夜侦探分镜',
  permission_mode: 'full_access' as const,
  status: 'active' as const,
  created_at: timestamp,
  updated_at: timestamp,
}
const asset = (
  id: string,
  name: string,
  kind: StudioAsset['kind'],
  origin: StudioAsset['origin'],
  mimeType: string
): StudioAsset => ({
  id,
  session_id: session.id,
  name,
  kind,
  origin,
  source_run_id: 'run-comic',
  current_version: 1,
  saved_to_library: false,
  versions: [
    {
      id: `${id}-v1`,
      version: 1,
      mime_type: mimeType,
      size_bytes: 128,
      content_url: `/api/v1/studio/assets/${id}/content`,
      created_at: timestamp,
    },
  ],
  created_at: timestamp,
  updated_at: timestamp,
})

const assets: StudioAsset[] = [
  asset('asset-outline', '故事大纲.md', 'document', 'agent', 'text/markdown'),
  asset(
    'asset-storyboard',
    '雨夜侦探-分镜预览.svg',
    'image',
    'workflow',
    'image/svg+xml'
  ),
]
const node = (
  id: string,
  type: StudioFlowNode['type'],
  title: string,
  x: number,
  assetId?: string
): StudioFlowNode => ({
  id,
  type,
  title,
  ...(assetId
    ? { asset_id: assetId, asset_version_id: `${assetId}-v1`, asset_version: 1 }
    : {}),
  position: { x, y: 120 },
  sort_order: x,
  updated_at: timestamp,
})
const initialDetail: StudioSessionDetail = {
  session,
  messages: [],
  transcript: { messages: [], events: [] },
  assets: [],
  flow: { nodes: [], edges: [] },
}
const completedDetail: StudioSessionDetail = {
  session: {
    ...session,
    latest_run: {
      id: 'run-comic',
      session_id: session.id,
      trigger_message_id: 'user-comic',
      status: 'succeeded',
      model_config_id: 'model-chat',
      created_at: timestamp,
      completed_at: timestamp,
      updated_at: timestamp,
    },
  },
  messages: [
    {
      id: 'user-comic',
      session_id: session.id,
      run_id: 'run-comic',
      role: 'user',
      content: [{ type: 'text', text: '把雨夜侦探做成分镜' }],
      created_at: timestamp,
    },
    {
      id: 'assistant-comic',
      session_id: session.id,
      run_id: 'run-comic',
      role: 'assistant',
      content: [{ type: 'text', text: '分镜预览已生成' }],
      created_at: timestamp,
    },
  ],
  transcript: {
    messages: [
      { id: 'user-comic', role: 'user', content: '把雨夜侦探做成分镜' },
      { id: 'assistant-comic', role: 'assistant', content: '分镜预览已生成' },
    ],
    events: [],
  },
  assets,
  flow: {
    nodes: [
      node('stage-1', 'stage', '立住故事', 40),
      node('outline-1', 'asset', '故事大纲', 280, 'asset-outline'),
      node('workflow-1', 'operation', '分镜工作流', 520),
      node('storyboard-1', 'asset', '分镜预览', 760, 'asset-storyboard'),
    ],
    edges: [
      { id: 'edge-1', source: 'stage-1', target: 'outline-1', label: '产出' },
      {
        id: 'edge-2',
        source: 'outline-1',
        target: 'workflow-1',
        label: '作为输入',
      },
      {
        id: 'edge-3',
        source: 'workflow-1',
        target: 'storyboard-1',
        label: '输出',
      },
    ],
  },
}

const originalWebSocket = globalThis.WebSocket
afterEach(() => {
  globalThis.WebSocket = originalWebSocket
})

describe('Studio comic journey', () => {
  it('shows a workflow output in both the Flow and Session assets after a Chat turn', async () => {
    const originalViewport = {
      width: window.innerWidth,
      height: window.innerHeight,
    }
    await page.viewport(1440, 900)
    let completed = false
    const requests: Array<Record<string, unknown>> = []
    vi.mocked(listStudioSessions).mockResolvedValue([session])
    vi.mocked(getStudioSession).mockImplementation(async () =>
      completed ? completedDetail : initialDetail
    )
    vi.mocked(listStudioModels).mockResolvedValue([
      {
        id: 'model-chat',
        name: 'Pixoma Chat',
        protocol: 'openai_chat_compatible',
        base_url: 'https://model.test/v1',
        model: 'chat-model',
        has_api_key: true,
        enabled: true,
        agent_enabled: true,
        default: true,
        limits: {
          context_window_tokens: 128000,
          max_input_tokens: 32000,
          max_output_tokens: 8000,
        },
        thinking: { enabled: false },
        capabilities: {
          tools: true,
          vision: false,
          image_output: false,
          streaming: true,
        },
      },
    ])

    class ComicSocket {
      static OPEN = 1
      static CONNECTING = 0
      readyState = 0
      onopen: (() => void) | null = null
      onmessage: ((event: { data: string }) => void) | null = null
      onclose: (() => void) | null = null
      onerror: (() => void) | null = null
      constructor(_url: string) {
        queueMicrotask(() => {
          this.readyState = 1
          this.onopen?.()
        })
      }
      send(raw: string) {
        const request = JSON.parse(raw) as Record<string, unknown>
        requests.push(request)
        const emit = (event: Record<string, unknown>) =>
          this.onmessage?.({ data: JSON.stringify(event) })
        queueMicrotask(() => {
          emit({
            type: 'RUN_STARTED',
            threadId: session.id,
            runId: request.runId,
            metadata: { studioRunId: 'run-comic' },
          })
          emit({
            type: 'TEXT_MESSAGE_START',
            messageId: 'assistant-comic',
            role: 'assistant',
            sequence: 1,
          })
          emit({
            type: 'TEXT_MESSAGE_CONTENT',
            messageId: 'assistant-comic',
            delta: '分镜预览已生成',
            sequence: 2,
          })
          emit({
            type: 'TEXT_MESSAGE_END',
            messageId: 'assistant-comic',
            sequence: 3,
          })
          emit({
            type: 'TOOL_CALL_START',
            toolCallId: 'call-storyboard',
            toolCallName: '分镜工作流',
            sequence: 4,
          })
          emit({
            type: 'TOOL_CALL_ARGS',
            toolCallId: 'call-storyboard',
            delta: '{"input_asset_ids":["asset-outline"]}',
            sequence: 5,
          })
          emit({
            type: 'TOOL_CALL_RESULT',
            toolCallId: 'call-storyboard',
            content: '已生成分镜预览资产',
            role: 'tool',
            isError: false,
            sequence: 6,
          })
          emit({
            type: 'TOOL_CALL_END',
            toolCallId: 'call-storyboard',
            sequence: 7,
          })
          completed = true
          emit({
            type: 'RUN_FINISHED',
            threadId: session.id,
            runId: request.runId,
            outcome: { type: 'success' },
            sequence: 8,
          })
        })
      }
      close() {
        this.readyState = 3
        this.onclose?.()
      }
    }
    globalThis.WebSocket = ComicSocket as never
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    try {
      const screen = await render(
        <QueryClientProvider client={client}>
          <StudioWorkspace />
        </QueryClientProvider>
      )
      await screen
        .getByPlaceholder('描述你想创作的内容，或让 Agent 调用工作流…')
        .fill('把雨夜侦探做成分镜')
      await screen.getByRole('button', { name: '发送消息' }).click()
      await expect.element(screen.getByText('分镜预览已生成')).toBeVisible()
      expect(JSON.stringify(requests[0].messages)).toContain(
        '把雨夜侦探做成分镜'
      )
      expect(requests[0].forwardedProps).toMatchObject({
        runConfig: {
          modelConfigId: 'model-chat',
          permissionMode: 'full_access',
        },
      })
      await expect
        .element(screen.getByText('分镜预览', { exact: true }))
        .toBeVisible()
      await screen.getByRole('tab', { name: /Session 资产/ }).click()
      await expect.element(screen.getByText('故事大纲.md')).toBeVisible()
      await expect
        .element(screen.getByText('雨夜侦探-分镜预览.svg'))
        .toBeVisible()
    } finally {
      await page.viewport(originalViewport.width, originalViewport.height)
    }
  })
})
