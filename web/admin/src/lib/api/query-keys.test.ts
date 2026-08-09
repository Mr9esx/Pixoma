import { describe, it, expect } from 'vitest'
import { queryKeys } from './query-keys'

describe('queryKeys', () => {
  it('exposes stable all / detail factories for every resource', () => {
    expect(queryKeys.instances.all).toEqual(['instances'])
    expect(queryKeys.instances.detail('gpu-1')).toEqual(['instances', 'gpu-1'])

    expect(queryKeys.cases.all).toEqual(['cases'])
    expect(queryKeys.cases.detail('c1')).toEqual(['cases', 'c1'])

    expect(queryKeys.tasks.all).toEqual(['tasks'])
    expect(queryKeys.tasks.detail('t1')).toEqual(['tasks', 't1'])

    expect(queryKeys.users.all).toEqual(['users'])
    expect(queryKeys.users.detail('u1')).toEqual(['users', 'u1'])

    expect(queryKeys.sessions.all).toEqual(['sessions'])
    expect(queryKeys.sessions.detail('s1')).toEqual(['sessions', 's1'])

    expect(queryKeys.tgMenu.all).toEqual(['tg-menu'])
  })
})
