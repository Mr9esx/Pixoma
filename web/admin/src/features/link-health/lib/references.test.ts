import { describe, expect, it } from 'vitest'
import type { MenuPlacement } from '@/lib/api/channel-menu'
import type { CaseRecord } from '@/lib/api/types'
import type { EdgePresence, EdgeRecord } from '@/features/task-flow/types'
import {
  caseReferences,
  caseRoutingTopics,
  channelReferences,
  edgeReferences,
  edgeIsReady,
  edgeTopics,
  topicReferences,
} from './references'

const caseA = {
  id: 1,
  name: '案例 A',
  routing: {
    rules: [{ when: { field: 'x', op: 'eq', value: 1 }, topic: 't1' }],
  },
} as unknown as CaseRecord
const caseDefault = {
  id: 2,
  name: '案例 B',
  routing: {
    rules: [{ when: { always: true }, topic: 'default' }],
  },
} as unknown as CaseRecord
const edgeT1: EdgeRecord = {
  id: 'node-1',
  name: '节点 1',
  enabled: true,
  subscribe_topics: ['t1'],
  effective_topics: ['t1'],
}
const edgeDefault: EdgeRecord = {
  id: 'node-2',
  name: '节点 2',
  enabled: true,
  subscribe_topics: ['default'],
  effective_topics: [],
}
const presence: EdgePresence[] = [
  { id: 'node-1', edge_online: true, comfy_running: true },
]
const placement: MenuPlacement = {
  channel_id: 'c1',
  channel_name: '消息平台 1',
  item_id: 'm1',
  kind: 'open_case',
  path: [{ id: 'm1', label: '入口' }],
}

describe('caseRoutingTopics', () => {
  it('无 routing 时不推断默认 Topic', () => {
    expect(caseRoutingTopics(undefined)).toEqual([])
  })
  it('返回去重后的规则 Topic 列表', () => {
    expect(
      caseRoutingTopics({
        rules: [
          { when: { field: 'a', op: 'eq', value: 1 }, topic: 't1' },
          { when: { field: 'b', op: 'eq', value: 2 }, topic: 't1' },
        ],
      })
    ).toEqual(['t1'])
  })
})

describe('edgeIsReady', () => {
  it('启用且在线且 Comfy 运行时为就绪', () => {
    expect(edgeIsReady(edgeT1, presence)).toBe(true)
  })
  it('停用节点不算就绪', () => {
    expect(edgeIsReady({ ...edgeT1, enabled: false }, presence)).toBe(false)
  })
})

describe('edgeTopics', () => {
  it('未显式订阅的节点按默认订阅 Default 计', () => {
    expect(
      edgeTopics({
        id: 'n1',
        name: 'n',
        enabled: true,
        subscribe_topics: [],
        effective_topics: ['default'],
      })
    ).toEqual(['default'])
  })
  it('effective_topics 非空时优先于显式订阅', () => {
    expect(
      edgeTopics({
        id: 'n1',
        name: 'n',
        enabled: true,
        subscribe_topics: ['a'],
        effective_topics: ['a', 'default'],
      })
    ).toEqual(['a', 'default'])
  })
})

describe('topicReferences', () => {
  const input = {
    cases: [caseA, caseDefault],
    edges: [edgeT1, edgeDefault],
    presence,
  }
  it('t1 被 Case A 引用且节点在线 → ok', () => {
    const refs = topicReferences('t1', input)
    expect(refs.cases.map((c) => c.id)).toEqual(['1'])
    expect(refs.edges.map((e) => e.id)).toEqual(['node-1'])
    expect(refs.health.state).toBe('ok')
    expect(refs.health.breakpoints).toEqual([])
  })
  it('default 只有离线节点 → warn、断点 runtime、带行动', () => {
    const refs = topicReferences('default', input)
    expect(refs.cases.map((c) => c.id)).toEqual(['2'])
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find((b) => b.stage === 'node')
    expect(bp?.fix).toBe('runtime')
    expect(bp?.action.to).toBe('/edges')
    expect(bp?.action.key).toBe('linkHealth.actionManageNodes')
    expect(bp?.guide).toBe('linkHealth.guideSubscribersOffline')
  })
  it('未被引用的 Topic → warn、断点 config、带行动', () => {
    const refs = topicReferences('t-ghost', input)
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints[0]
    expect(bp?.stage).toBe('workflow')
    expect(bp?.fix).toBe('config')
    expect(bp?.action.key).toBe('linkHealth.actionConfigureRouting')
  })
})

describe('edgeReferences', () => {
  const input = {
    cases: [caseA, caseDefault],
    edges: [edgeT1, edgeDefault],
    presence,
  }
  it('node-1 被 t1 绑定并可到达 Case A → ok', () => {
    const refs = edgeReferences('node-1', input)
    expect(refs.topics.map((t) => t.id)).toEqual(['t1'])
    expect(refs.cases.map((c) => c.id)).toEqual(['1'])
    expect(refs.health.state).toBe('ok')
  })
  it('node-2 离线 → warn、断点 runtime、行动指向节点', () => {
    const refs = edgeReferences('node-2', input)
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find(
      (b) => b.key === 'linkHealth.edgeNotReady'
    )
    expect(bp?.fix).toBe('runtime')
    expect(bp?.action.to).toBe('/edges/node-2')
  })
  it('未绑定 Topic 的节点只提示去部署，不再引导去配置路由', () => {
    const unbound: EdgeRecord = {
      id: 'node-3',
      name: '节点 3',
      enabled: true,
      subscribe_topics: [],
      effective_topics: [],
    }
    const refs = edgeReferences('node-3', {
      ...input,
      edges: [edgeT1, edgeDefault, unbound],
    })
    const keys = refs.health.breakpoints.map((b) => b.key)
    expect(keys).toContain('linkHealth.noTopicBinding')
    expect(keys).not.toContain('linkHealth.noCaseReachable')
    const bp = refs.health.breakpoints.find(
      (b) => b.key === 'linkHealth.noTopicBinding'
    )
    expect(bp?.action.key).toBe('linkHealth.actionDeployNode')
  })
})

describe('caseReferences', () => {
  const input = {
    cases: [caseA, caseDefault],
    edges: [edgeT1, edgeDefault],
    presence,
    placements: [placement],
  }
  it('无入口时 warn、断点 config、行动去添加入口', () => {
    const refs = caseReferences(1, { ...input, placements: [] })
    expect(refs.menuEntries).toEqual([])
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find(
      (b) => b.key === 'linkHealth.noMenuEntry'
    )
    expect(bp?.fix).toBe('config')
    expect(bp?.action.key).toBe('linkHealth.actionAddEntry')
    expect(bp?.guide).toBe('linkHealth.guideNoMenuEntry')
  })
  it('有入口但节点离线 → warn、断点 runtime', () => {
    const refs = caseReferences(2, input)
    expect(refs.menuEntries).toHaveLength(1)
    expect(refs.topics.map((t) => t.id)).toEqual(['default'])
    expect(refs.health.state).toBe('warn')
    expect(
      refs.health.breakpoints.some(
        (b) => b.stage === 'node' && b.fix === 'runtime'
      )
    ).toBe(true)
  })
})

describe('channelReferences', () => {
  it('网络不可达 → warn、断点 runtime、行动去设置代理', () => {
    const health = channelReferences('c1', {
      ok: false,
      kind: 'network',
      message: 'dial timeout',
    })
    expect(health.state).toBe('warn')
    expect(health.breakpoints[0]).toMatchObject({
      fix: 'runtime',
      action: { to: '/settings', key: 'linkHealth.actionConfigureProxy' },
    })
  })

  it('Token 无效 → warn、断点 config、行动指向编辑消息平台', () => {
    const health = channelReferences('c1', {
      ok: false,
      kind: 'auth',
      message: 'unauthorized',
    })
    expect(health.breakpoints[0]).toMatchObject({
      fix: 'config',
      action: { to: '/channels/c1', key: 'linkHealth.actionEditChannel' },
    })
  })

  it('连接正常 → ok、无断点', () => {
    const health = channelReferences('c1', {
      ok: true,
      kind: 'ok',
      message: '',
    })
    expect(health.state).toBe('ok')
    expect(health.breakpoints).toEqual([])
  })

  it('其他错误 → warn、断点带 message 参数', () => {
    const health = channelReferences('c1', {
      ok: false,
      kind: 'other',
      message: 'boom',
    })
    expect(health.breakpoints[0]).toMatchObject({
      params: { message: 'boom' },
    })
  })
})
