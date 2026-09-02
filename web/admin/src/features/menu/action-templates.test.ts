import { describe, expect, it } from 'vitest'
import type { ActionType } from '@/lib/api/channel-menu'
import {
  ACTION_FIELD_KIND,
  getActionField,
  hasTemplate,
  listActionTypes,
} from './action-templates'

describe('action-templates', () => {
  it('listActionTypes 包含 7 种动作，含 list_tasks', () => {
    const types = listActionTypes()
    expect(types).toHaveLength(7)
    expect(types).toContain('open_workflow')
    expect(types).toContain('list_tasks')
    expect(types).toContain('open_card')
    expect(types).toContain('send_text')
    expect(types).toContain('send_media')
    expect(types).toContain('open_url')
    expect(types).toContain('copy_text')
    expect(types).not.toContain('placeholder')
  })

  it('listActionTypes 严格匹配 ActionType union', () => {
    const types: ActionType[] = listActionTypes()
    // 编译期 + 运行期：listActionTypes 应该 exhaust ActionType
    expect(new Set(types).size).toBe(types.length)
  })

  it('ACTION_FIELD_KIND：list_tasks 无额外字段，其余有字段', () => {
    expect(getActionField('list_tasks')).toBe('none')
    for (const type of listActionTypes()) {
      expect(ACTION_FIELD_KIND[type]).toBeTruthy()
      if (type === 'list_tasks') {
        expect(ACTION_FIELD_KIND[type]).toBe('none')
        continue
      }
      expect(ACTION_FIELD_KIND[type]).not.toBe('none')
    }
  })

  it('open_workflow 映射到 workflow 字段（单值 Select）', () => {
    expect(getActionField('open_workflow')).toBe('workflow')
  })

  it('open_card 映射到 card 字段', () => {
    expect(getActionField('open_card')).toBe('card')
  })

  it('send_text / copy_text 映射到 text 字段', () => {
    expect(getActionField('send_text')).toBe('text')
    expect(getActionField('copy_text')).toBe('text')
  })

  it('send_media 映射到 media 字段', () => {
    expect(getActionField('send_media')).toBe('media')
  })

  it('open_url 映射到 url 字段', () => {
    expect(getActionField('open_url')).toBe('url')
  })

  it('hasTemplate：list_tasks 为 false，其余为 true', () => {
    expect(hasTemplate('list_tasks')).toBe(false)
    for (const type of listActionTypes()) {
      if (type === 'list_tasks') continue
      expect(hasTemplate(type)).toBe(true)
    }
  })
})
