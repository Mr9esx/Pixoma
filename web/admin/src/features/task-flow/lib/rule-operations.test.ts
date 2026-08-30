import { describe, expect, it } from 'vitest'
import { describeCondition } from './rule-operations'

describe('describeCondition', () => {
  it('always 显示为无条件', () => {
    expect(describeCondition({ always: true })).toBe('无条件')
  })
})
