import { describe, it, expect } from 'vitest'
import { queryKeys } from './query-keys'

describe('queryKeys', () => {
  it('exposes stable all / detail factories for every resource', () => {
    expect(queryKeys.edges.all).toEqual(['edges'])
    expect(queryKeys.edges.detail('gpu-1')).toEqual(['edges', 'gpu-1'])
    expect(queryKeys.edges.presence).toEqual(['edges', 'presence'])
    expect(queryKeys.edges.stats('gpu-1')).toEqual(['edges', 'gpu-1', 'stats'])

    expect(queryKeys.cases.all).toEqual(['cases'])
    expect(queryKeys.cases.detail('c1')).toEqual(['cases', 'c1'])
    expect(queryKeys.cases.menuPlacements('c1')).toEqual([
      'cases',
      'c1',
      'menu-placements',
    ])

    expect(queryKeys.tasks.all).toEqual(['tasks'])
    expect(queryKeys.tasks.detail('t1')).toEqual(['tasks', 't1'])

    expect(queryKeys.users.all).toEqual(['users'])
    expect(queryKeys.users.detail('u1')).toEqual(['users', 'u1'])

    expect(queryKeys.sessions.all).toEqual(['sessions'])
    expect(queryKeys.sessions.detail('s1')).toEqual(['sessions', 's1'])

    expect(queryKeys.channels.all).toEqual(['channels'])
    expect(queryKeys.channels.detail('ch1')).toEqual(['channels', 'ch1'])
    expect(queryKeys.channels.menu('ch1')).toEqual(['channels', 'ch1', 'menu'])
    expect(queryKeys.topics.all).toEqual(['topics'])
    expect(queryKeys.topics.detail('fast-gpu')).toEqual(['topics', 'fast-gpu'])
    expect(queryKeys.topics.stats('fast-gpu')).toEqual(['topics', 'fast-gpu', 'stats'])
    expect(queryKeys.settings.all).toEqual(['settings'])
  })
})
