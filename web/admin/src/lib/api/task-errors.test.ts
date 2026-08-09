import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { taskActionErrorMessage } from './task-errors'

const t = (key: string) => {
  const map: Record<string, string> = {
    'tasks.cannotCancel': '不可取消',
    'tasks.notFound': '资源不存在',
    'common.errorGeneric': '请求失败',
  }
  return map[key] ?? key
}

describe('taskActionErrorMessage', () => {
  it('maps 409 to cannotCancel', () => {
    expect(taskActionErrorMessage(new ApiError(409, 'conflict'), t)).toBe(
      '不可取消',
    )
  })

  it('maps 404 to notFound', () => {
    expect(taskActionErrorMessage(new ApiError(404, 'missing'), t)).toBe(
      '资源不存在',
    )
  })

  it('uses backend message for other ApiError statuses', () => {
    expect(taskActionErrorMessage(new ApiError(500, 'boom'), t)).toBe('boom')
  })

  it('falls back to generic for unknown errors', () => {
    expect(taskActionErrorMessage('x', t)).toBe('请求失败')
  })
})
