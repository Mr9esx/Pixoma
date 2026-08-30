import { describe, expect, it } from 'vitest'
import { emptyCase } from './empty-case'

describe('emptyCase', () => {
  it('新建工作流默认有无条件 default 路由', () => {
    const routing = emptyCase().routing
    expect(routing?.rules).toEqual([
      { when: { always: true }, topic: 'default' },
    ])
  })
})
