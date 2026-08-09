import { describe, it, expect } from 'vitest'
import { aggregateDashboard } from './aggregate'

describe('aggregateDashboard', () => {
  it('counts instances and task statuses', () => {
    const stats = aggregateDashboard({
      instances: [
        { id: 'a', enabled: true },
        { id: 'b', enabled: false },
      ],
      cases: [
        { id: 'c1', enabled: true },
        { id: 'c2', enabled: false },
      ],
      tasks: [
        { id: 't1', status: 'pending' },
        { id: 't2', status: 'pending' },
        { id: 't3', status: 'succeeded' },
      ],
    })
    expect(stats.instanceTotal).toBe(2)
    expect(stats.instanceEnabled).toBe(1)
    expect(stats.caseEnabled).toBe(1)
    expect(stats.taskByStatus).toEqual({ pending: 2, succeeded: 1 })
  })
})
