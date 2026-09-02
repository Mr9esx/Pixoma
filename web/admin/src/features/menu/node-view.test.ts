import { describe, expect, it } from 'vitest'
import { describeAction, type WorkflowRef } from './node-view'

const wl: WorkflowRef[] = [
  { id: 10, name: '写实' },
  { id: 20, name: '二次元' },
]

describe('describeAction', () => {
  it('open_workflow 展示绑定工作流名', () => {
    const v = describeAction(
      { type: 'open_workflow', workflow_id: '10' },
      wl
    )
    expect(v.kind).toBe('workflow')
    expect(v.summary).toBe('打开工作流「写实」')
    expect(v.params[0].value).toBe('写实')
  })

  it('open_card 展示内嵌卡片文字', () => {
    const v = describeAction(
      { type: 'open_card', card: { text: '选风格' } },
      wl
    )
    expect(v.kind).toBe('card')
    expect(v.summary).toBe('打开卡片「选风格」')
  })

  it('send_text 展示文本内容', () => {
    const v = describeAction({ type: 'send_text', text: '欢迎使用' }, wl)
    expect(v.kind).toBe('text')
    expect(v.summary).toBe('发文字「欢迎使用」')
  })

  it('copy_text 归为 text', () => {
    const v = describeAction({ type: 'copy_text', text: '复制我' }, wl)
    expect(v.kind).toBe('text')
    expect(v.summary).toContain('复制文本')
  })

  it('send_media 展示媒体数量与链接', () => {
    const v = describeAction(
      {
        type: 'send_media',
        media: [{ kind: 'image', url: 'https://a/b.png' }],
      },
      wl
    )
    expect(v.kind).toBe('media')
    expect(v.summary).toBe('发媒体（1 个）')
    expect(v.params[0].value).toBe('https://a/b.png')
  })

  it('list_tasks 展示为我的任务', () => {
    const v = describeAction({ type: 'list_tasks' }, wl)
    expect(v.kind).toBe('tasks')
    expect(v.summary).toBe('我的任务')
  })
})
