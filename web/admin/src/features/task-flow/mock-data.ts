import type {
  AttributesCatalogResponse,
  CaseWithRouting,
  EdgePresence,
  EdgeRecord,
  RoutingConfig,
  TopicRecord,
} from './types'

/** 后端 GET /api/v1/topics 的真实响应形状。 */
export const mockTopics: TopicRecord[] = [
  {
    key: 'default',
    name: '默认 Topic',
    enabled: true,
    created_at: '2026-08-20T08:00:00Z',
    updated_at: '2026-08-20T08:00:00Z',
  },
  {
    key: 'fast-gpu',
    name: '高规格 GPU',
    enabled: true,
    created_at: '2026-08-20T08:05:00Z',
    updated_at: '2026-08-20T08:05:00Z',
  },
  {
    key: 'batch-night',
    name: '夜间批量',
    enabled: true,
    created_at: '2026-08-20T08:10:00Z',
    updated_at: '2026-08-20T08:10:00Z',
  },
  {
    key: 'cpu-low',
    name: '低成本 CPU',
    enabled: true,
    created_at: '2026-08-20T08:15:00Z',
    updated_at: '2026-08-20T08:15:00Z',
  },
]

/** 后端 GET /api/v1/routing/attributes 的真实响应形状（与 topic-routing 内置 provider 一致）。 */
export const mockAttributes: AttributesCatalogResponse = {
  attributes: [
    {
      key: 'user.is_premium',
      context: 'user',
      label: '用户是否付费（Premium）',
      schema: { type: 'boolean', description: '用户 Telegram Premium 状态' },
    },
    {
      key: 'case.category',
      context: 'case',
      label: 'Case 分类',
      schema: {
        type: 'string',
        enum: ['image', 'video', 'audio'],
        description: 'Case 类别',
      },
    },
    {
      key: 'case.tags',
      context: 'case',
      label: 'Case 标签',
      schema: {
        type: 'array',
        items: { type: 'string' },
        description: 'Case 标签列表',
      },
    },
  ],
}

/** 一个真实的 Case 路由配置（保存后发送给 case-admin-api 的 routing 字段）。 */
export const mockRouting: RoutingConfig = {
  rules: [
    {
      when: { field: 'user.is_premium', op: 'eq', value: true },
      topic: 'fast-gpu',
    },
    {
      when: { field: 'case.category', op: 'in', value: ['image'] },
      topic: 'default',
    },
    {
      when: {
        or: [
          { field: 'case.tags', op: 'in', value: ['night'] },
          { field: 'case.category', op: 'eq', value: 'audio' },
        ],
      },
      topic: 'batch-night',
    },
  ],
}

export const mockCase: CaseWithRouting = {
  id: 1001,
  name: '电商主图',
  routing: mockRouting,
}

/** 左侧 Case 列表示例：每个 Case 拥有独立 routing（拖入画布即加载该配置）。 */
export const mockCases: CaseWithRouting[] = [
  mockCase,
  {
    id: 1002,
    name: '视频配乐',
    routing: {
      rules: [
        { when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' },
        { when: { field: 'case.tags', op: 'in', value: ['4k'] }, topic: 'fast-gpu' },
      ],
    },
  },
  {
    id: 1003,
    name: '音频转写',
    routing: {
      rules: [{ when: { field: 'case.category', op: 'eq', value: 'audio' }, topic: 'batch-night' }],
    },
  },
]

/** 后端 GET/PATCH /api/v1/edges/{id} 的真实响应形状：effective_topics 即后台绑定关系。 */
export const mockEdges: EdgeRecord[] = [
  {
    id: 'gpu-a',
    name: '客厅 4090',
    enabled: true,
    subscribe_topics: ['default', 'fast-gpu'],
    effective_topics: ['default'],
  },
  {
    id: 'gpu-b',
    name: '工作室双卡',
    enabled: true,
    subscribe_topics: ['fast-gpu'],
    effective_topics: ['fast-gpu'],
  },
  {
    id: 'gpu-c',
    name: '备用夜机',
    enabled: true,
    subscribe_topics: ['batch-night'],
    effective_topics: ['batch-night'],
  },
]

/** agent 心跳：gpu-a/gpu-b 在线（可拉取），gpu-c 离线 → batch-night 处于「已绑定但无在线 agent」。 */
export const mockPresence: EdgePresence[] = [
  { id: 'gpu-a', edge_online: true, comfy_running: true },
  { id: 'gpu-b', edge_online: true, comfy_running: true },
  { id: 'gpu-c', edge_online: false, comfy_running: false },
]
