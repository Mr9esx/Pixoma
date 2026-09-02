import { describe, expect, it } from 'vitest'
import { composeListTasksPreview } from './list-tasks-copy'

describe('list-tasks preview copy', () => {
  it('matches the Telegram empty-current + recent shape', () => {
    expect(composeListTasksPreview()).toBe(
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

  it('uses text-template overrides', () => {
    expect(
      composeListTasksPreview({ list_tasks: '任务清单\n{{ current }}{{ recent }}' })
    ).toMatch(/^任务清单\n/)
  })
})
