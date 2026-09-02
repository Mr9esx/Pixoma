import { describe, expect, it } from 'vitest'
import { LIST_TASKS_SAMPLE_ZH } from './list-tasks-preview'

describe('list-tasks preview copy', () => {
  it('matches the Telegram empty-current + recent shape', () => {
    expect(LIST_TASKS_SAMPLE_ZH).toBe(
      [
        '我的任务',
        '',
        '当前任务',
        '✅ 当前没有排队中的任务',
        '',
        '最近任务',
        '✅ #1120186 · 工作流名 · 已完成 · 08-30 07:50',
      ].join('\n')
    )
  })
})
