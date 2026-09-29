import { expect, it } from 'vitest'
import { studioNewSessionActivity } from './studio-new-session'

it('shows the current stage while sending a new session message', () => {
  expect(studioNewSessionActivity(true, false)).toBe('正在创建对话…')
  expect(studioNewSessionActivity(true, true)).toBe('正在上传图片…')
  expect(studioNewSessionActivity(false, false)).toBeUndefined()
})
