import { describe, expect, it } from 'vitest'
import {
  healthProblems,
  resolvedEntityHealth,
  type LinkHealthGraph,
} from './link-health'

const graph: LinkHealthGraph = {
  nodes: [
    {
      id: 'case:1',
      kind: 'case',
      ref_id: '1',
      name: '海报',
      health: 'ok',
      breakpoints: [],
      upstream: [],
      downstream: [],
    },
  ],
  edges: [],
}

describe('resolvedEntityHealth', () => {
  it('is pending when the query is not ready', () => {
    expect(resolvedEntityHealth(undefined, 'case', '1', false).state).toBe(
      'pending'
    )
    expect(healthProblems(undefined, false)).toBe(1)
  })

  it('is pending when the node is missing after a successful fetch', () => {
    expect(resolvedEntityHealth(graph, 'case', '99', true).state).toBe(
      'pending'
    )
    expect(healthProblems(undefined, true)).toBe(1)
  })

  it('uses assembler health when the node exists', () => {
    expect(resolvedEntityHealth(graph, 'case', '1', true).state).toBe('ok')
    expect(healthProblems({ state: 'ok', breakpoints: [] }, true)).toBe(0)
    expect(healthProblems({ state: 'warn', breakpoints: [] }, true)).toBe(1)
    expect(healthProblems({ state: 'pending', breakpoints: [] }, true)).toBe(1)
  })
})
