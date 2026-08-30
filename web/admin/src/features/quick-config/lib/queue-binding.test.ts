import { describe, expect, it } from 'vitest'
import { nodeStepCanAdvance, queueHasSubscribers } from './queue-binding'

const edge = (
  id: string,
  topics: string[],
  enabled = true
): {
  id: string
  enabled: boolean
  subscribe_topics: string[]
  effective_topics: string[]
} => ({
  id,
  enabled,
  subscribe_topics: topics,
  effective_topics: topics,
})

describe('queueHasSubscribers', () => {
  it('无人订阅时为 false', () => {
    expect(queueHasSubscribers([edge('a', [])], 'default')).toBe(false)
  })

  it('启用节点订了该队列时为 true', () => {
    expect(queueHasSubscribers([edge('a', ['default'])], 'default')).toBe(
      true
    )
  })

  it('仅停用节点订了不算', () => {
    expect(
      queueHasSubscribers([edge('a', ['default'], false)], 'default')
    ).toBe(false)
  })
})

describe('nodeStepCanAdvance', () => {
  it('队上有人时不选节点也能过', () => {
    expect(nodeStepCanAdvance(true, null)).toBe(true)
  })

  it('队上无人且未选节点不能过', () => {
    expect(nodeStepCanAdvance(false, null)).toBe(false)
  })

  it('队上无人但选了节点能过', () => {
    expect(nodeStepCanAdvance(false, 'gpu-1')).toBe(true)
  })
})
